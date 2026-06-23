package main

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

// PendingFileChange represents a file change proposed by the AI
// that the user hasn't accepted yet
type PendingFileChange struct {
	FilePath string `json:"filePath"`
	Content  string `json:"content"`
}

// AIProvider represents an AI API provider
type AIProvider struct {
	Name       string `json:"name"`
	APIKey     string `json:"apiKey"`
	BaseURL    string `json:"baseURL"`
	Model      string `json:"model"`
	Enabled    bool   `json:"enabled"`
}

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AgentConfig holds the application configuration
type AgentConfig struct {
	Providers      map[string]*AIProvider `json:"providers"`
	ActiveProvider string                 `json:"activeProvider"`
	ProjectRoot    string                 `json:"projectRoot"`
}

// App struct - the main application logic
type App struct {
	ctx             context.Context
	projectRoot     string
	config          *AgentConfig
	watcher         *fsnotify.Watcher
	watcherMu       sync.Mutex
	refreshTmr      *time.Timer
	pendingChanges  []PendingFileChange
	termCmd         *exec.Cmd
	termStdin       io.WriteCloser
	termRunning     bool
	termMu          sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		projectRoot: getDefaultProjectRoot(),
		config:      getDefaultConfig(),
	}
}

func getDefaultConfig() *AgentConfig {
	return &AgentConfig{
		Providers: map[string]*AIProvider{
			"openrouter": {
				Name:    "OpenRouter",
				APIKey:  "",
				BaseURL: "https://openrouter.ai/api/v1",
				Model:   "anthropic/claude-3.5-sonnet",
				Enabled: false,
			},
			"nvidia": {
				Name:    "NVIDIA",
				APIKey:  "",
				BaseURL: "https://integrate.api.nvidia.com/v1",
				Model:   "meta/llama-3.1-8b-instruct",
				Enabled: false,
			},
			"anthropic": {
				Name:    "Anthropic",
				APIKey:  "",
				BaseURL: "https://api.anthropic.com/v1",
				Model:   "claude-3-5-sonnet-20241022",
				Enabled: false,
			},
			"openai": {
				Name:    "OpenAI",
				APIKey:  "",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4o",
				Enabled: false,
			},
		},
		ActiveProvider: "openrouter",
	}
}

// GetProviders returns all configured AI providers
func (a *App) GetProviders() map[string]*AIProvider {
	return a.config.Providers
}

// GetActiveProvider returns the currently active provider
func (a *App) GetActiveProvider() string {
	return a.config.ActiveProvider
}

// SetActiveProvider sets the active AI provider
func (a *App) SetActiveProvider(providerName string) error {
	if _, exists := a.config.Providers[providerName]; !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	a.config.ActiveProvider = providerName
	a.saveConfig()
	return nil
}

// UpdateProvider updates a provider's configuration
func (a *App) UpdateProvider(providerName string, apiKey string, model string, enabled bool) (string, error) {
	provider, exists := a.config.Providers[providerName]
	if !exists {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}
	if provider == nil {
		// Re-initialize nil provider entry
		a.config.Providers[providerName] = &AIProvider{
			Name:    providerName,
			APIKey:  apiKey,
			BaseURL: "",
			Model:   model,
			Enabled: enabled,
		}
		a.saveConfig()
		return providerName + " initialized", nil
	}

	if apiKey != "" {
		provider.APIKey = apiKey
	}
	if model != "" {
		provider.Model = model
	}
	provider.Enabled = enabled

	a.saveConfig()
	return providerName + " saved", nil
}

// SendMessage sends a message to the active AI provider and gets a response
func (a *App) SendMessage(message string, context string) (string, error) {
	provider := a.config.Providers[a.config.ActiveProvider]
	if provider == nil || !provider.Enabled {
		return "", fmt.Errorf("no active provider configured. Please set up an API key in Settings.")
	}
	
	// Build the prompt with context
	systemPrompt := `You are MugSoft, a helpful coding assistant integrated into a VSCode-like editor. You can create and edit files directly.

When you need to create or modify a file, use the following format:

--- FILE: path/to/file.ext ---
file content goes here
--- END FILE ---

For example:
--- FILE: game.py ---
import pygame
print("Hello")
--- END FILE ---

You can create multiple files in one response. Each file section will be automatically written to disk. After creating files, explain what you created.`
	if context != "" {
		systemPrompt += "\n\nContext from open files:\n" + context
	}	// Build request body based on provider type
	var reqBody map[string]interface{}

	maxTokens := 8192

	switch a.config.ActiveProvider {
	case "anthropic":
		reqBody = map[string]interface{}{
			"model":       provider.Model,
			"max_tokens":  maxTokens,
			"system":      systemPrompt,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
		}
	default:
		// OpenRouter, NVIDIA, OpenAI use OpenAI-compatible format
		reqBody = map[string]interface{}{
			"model":       provider.Model,
			"max_tokens":  maxTokens,
			"messages": []map[string]string{
				{"role": "system", "content": systemPrompt},
				{"role": "user", "content": message},
			},
		}
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	// Determine correct API endpoint for each provider
	var apiURL string
	if a.config.ActiveProvider == "anthropic" {
		apiURL = "https://api.anthropic.com/v1/messages"
	} else {
		apiURL = provider.BaseURL + "/chat/completions"
	}
	
	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Set auth header based on provider type
	if a.config.ActiveProvider == "anthropic" {
		req.Header.Set("x-api-key", provider.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}
	
	// Special headers for different providers
	if a.config.ActiveProvider == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://mugsoft.local")
		req.Header.Set("X-Title", "MugSoft Agent")
	}
	
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v (URL: %s)", err, apiURL)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	// Extract response based on provider
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := msg["content"].(string); ok {
					return content, nil
				}
			}
		}
	}
	
	// Try Anthropic format
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if block, ok := content[0].(map[string]interface{}); ok {
			if text, ok := block["text"].(string); ok {
				return text, nil
			}
		}
	}
	
	return "", fmt.Errorf("unexpected response format from API")
}

// ParseAIResponse parses AI response for file operation markers and returns
// the cleaned response text plus a list of pending file changes (does NOT write files).
// To apply the changes, call ApplyPendingChanges.
func (a *App) ParseAIResponse(response string) (string, []PendingFileChange) {
	var changes []PendingFileChange
	
	// Pattern: --- FILE: <path> ---
	//          <content>
	//          --- END FILE ---
	fileRegex := regexp.MustCompile(`(?s)---\s*FILE:\s*(.+?)---\r?\n([\s\S]*?)---\s*END FILE\s*---`)
	matches := fileRegex.FindAllStringSubmatch(response, -1)
	
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		filePath := strings.TrimSpace(match[1])
		content := strings.TrimSpace(match[2])
		
		if filePath == "" {
			continue
		}
		
		changes = append(changes, PendingFileChange{
			FilePath: filePath,
			Content:  content,
		})
	}
	
	// Remove the file markers from the response for cleaner display
	cleaned := fileRegex.ReplaceAllString(response, "")
	
	return cleaned, changes
}

// GetPendingChanges returns any pending file changes from the last AI response
func (a *App) GetPendingChanges() []PendingFileChange {
	return a.pendingChanges
}

// ApplyPendingChanges writes all pending file changes to disk and clears the list
func (a *App) ApplyPendingChanges() ([]string, error) {
	var createdFiles []string
	
	for _, change := range a.pendingChanges {
		// Prevent path traversal outside project root
		fullPath := filepath.Join(a.projectRoot, change.FilePath)
		cleanRoot := filepath.Clean(a.projectRoot) + string(filepath.Separator)
		cleanPath := filepath.Clean(fullPath) + string(filepath.Separator)
		if !strings.HasPrefix(cleanPath, cleanRoot) {
			return createdFiles, fmt.Errorf("file outside project root: %s", change.FilePath)
		}
		
		if err := a.WriteFile(change.FilePath, change.Content); err != nil {
			return createdFiles, fmt.Errorf("error creating file %s: %v", change.FilePath, err)
		}
		createdFiles = append(createdFiles, change.FilePath)
		fmt.Printf("AI created/modified file: %s\n", change.FilePath)
	}
	
	a.pendingChanges = nil
	return createdFiles, nil
}

// AcceptSingleFile applies a single pending file change and removes it from the list
func (a *App) AcceptSingleFile(filePath string) (string, error) {
	for i, change := range a.pendingChanges {
		if change.FilePath == filePath {
			// Prevent path traversal outside project root
			fullPath := filepath.Join(a.projectRoot, change.FilePath)
			cleanRoot := filepath.Clean(a.projectRoot) + string(filepath.Separator)
			cleanPath := filepath.Clean(fullPath) + string(filepath.Separator)
			if !strings.HasPrefix(cleanPath, cleanRoot) {
				return "", fmt.Errorf("file outside project root: %s", change.FilePath)
			}
			
			if err := a.WriteFile(change.FilePath, change.Content); err != nil {
				return "", fmt.Errorf("error creating file %s: %v", change.FilePath, err)
			}
			
			// Remove from pending list
			a.pendingChanges = append(a.pendingChanges[:i], a.pendingChanges[i+1:]...)
			
			fmt.Printf("AI created/modified file: %s\n", change.FilePath)
			return change.FilePath, nil
		}
	}
	return "", fmt.Errorf("pending change not found: %s", filePath)
}

// RejectSingleFile removes a single pending file change without applying it
func (a *App) RejectSingleFile(filePath string) error {
	for i, change := range a.pendingChanges {
		if change.FilePath == filePath {
			a.pendingChanges = append(a.pendingChanges[:i], a.pendingChanges[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("pending change not found: %s", filePath)
}

// ClearPendingChanges discards all pending file changes without applying them
func (a *App) ClearPendingChanges() {
	a.pendingChanges = nil
}

// AskAI is a convenience method for the frontend to ask the AI a question
func (a *App) AskAI(question string, files []string) (string, error) {
	context := ""
	
	// Include content from selected files
	for _, filename := range files {
		if content, err := a.ReadFile(filename); err == nil {
			context += fmt.Sprintf("\n\n--- File: %s ---\n%s", filename, content)
		}
	}
	
	response, err := a.SendMessage(question, context)
	if err != nil {
		return "", err
	}
	
	// Parse file changes from the response (but don't apply them yet)
	// Store them as pending for the user to review
	cleaned, pendingChanges := a.ParseAIResponse(response)
	a.pendingChanges = pendingChanges
	
	// Add a note about pending changes if there are any
	if len(pendingChanges) > 0 {
		note := "\n\n📁 **Pending file changes — review below:**\n"
		for _, f := range pendingChanges {
			note += "- " + f.FilePath + "\n"
		}
		cleaned += note
	}
	
	return cleaned, nil
}

// SaveConfig saves the current configuration to disk
func (a *App) saveConfig() error {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	
	configPath := filepath.Join(configDir, "mugsoft-config.json")
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	return os.WriteFile(configPath, data, 0644)
}

func getConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".mugsoft")
}

// startFileWatcher begins watching the project directory for file changes
func (a *App) startFileWatcher() {
	a.watcherMu.Lock()
	defer a.watcherMu.Unlock()

	// Stop existing watcher if running
	if a.watcher != nil {
		a.watcher.Close()
		a.watcher = nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("Warning: could not start file watcher: %v\n", err)
		return
	}

	// Watch the project root directory
	if err := watcher.Add(a.projectRoot); err != nil {
		fmt.Printf("Warning: could not watch directory: %v\n", err)
		watcher.Close()
		return
	}

	// Watch subdirectories for new/deleted files
	filepath.Walk(a.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		// Skip .git directory to avoid excessive events
		if info.Name() == ".git" {
			return filepath.SkipDir
		}
		watcher.Add(path)
		return nil
	})

	a.watcher = watcher

	// Start the event loop in a goroutine
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Filter out CHMOD events (just permission changes) and .git directory changes
				if event.Op&fsnotify.Chmod == fsnotify.Chmod {
					continue
				}
				// Debounce: reset timer on each event
				a.watcherMu.Lock()
				if a.refreshTmr != nil {
					a.refreshTmr.Stop()
				}
				a.refreshTmr = time.AfterFunc(500*time.Millisecond, func() {
					a.emitRefreshEvent()
				})
				a.watcherMu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("File watcher error: %v\n", err)
			}
		}
	}()

	fmt.Println("File watcher started for:", a.projectRoot)
}

// emitRefreshEvent runs git status and notifies the frontend
func (a *App) emitRefreshEvent() {
	if a.ctx == nil {
		return
	}
	gitStatus := a.GetGitStatus()
	fmt.Printf("Files changed — %d files with git status\n", len(gitStatus))
	wailsRuntime.EventsEmit(a.ctx, "files-changed")
}

// stopFileWatcher stops the file system watcher
func (a *App) stopFileWatcher() {
	a.watcherMu.Lock()
	defer a.watcherMu.Unlock()
	if a.refreshTmr != nil {
		a.refreshTmr.Stop()
		a.refreshTmr = nil
	}
	if a.watcher != nil {
		a.watcher.Close()
		a.watcher = nil
		fmt.Println("File watcher stopped")
	}
}

// domReady is called after the front-end dom has been loaded
func (a *App) domReady(ctx context.Context) {
	fmt.Println("DOM ready - UI is loaded")
	a.startFileWatcher()
	a.StartTerminal()
}

// Returning false prevents the application from closing.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	fmt.Println("Shutting down MugSoft Agent...")
	a.stopFileWatcher()
	a.StopTerminal()
	return false
}

// GetAllFiles returns all files in the current project
func (a *App) GetAllFiles() []string {
	var files []string
	filepath.Walk(a.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(a.projectRoot, path)
			files = append(files, relPath)
		}
		return nil
	})
	return files
}

// ReadFile reads a file from the project
func (a *App) ReadFile(filename string) (string, error) {
	fullPath := filepath.Join(a.projectRoot, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteFile writes content to a file
func (a *App) WriteFile(filename string, content string) error {
	fullPath := filepath.Join(a.projectRoot, filename)
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	os.MkdirAll(dir, 0755)
	
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// StartTerminal spawns a persistent shell process (cmd.exe) with stdin/stdout pipes
func (a *App) StartTerminal() {
	a.termMu.Lock()
	defer a.termMu.Unlock()

	if a.termRunning {
		return
	}

	shell := "cmd.exe"
	if runtime.GOOS != "windows" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell)
	cmd.Dir = a.projectRoot

	// Hide the cmd window on Windows so it doesn't pop up externally
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Terminal: failed to create stdin pipe: %v\n", err)
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Terminal: failed to create stdout pipe: %v\n", err)
		stdin.Close()
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Printf("Terminal: failed to create stderr pipe: %v\n", err)
		stdin.Close()
		return
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Terminal: failed to start shell: %v\n", err)
		stdin.Close()
		return
	}

	a.termCmd = cmd
	a.termStdin = stdin
	a.termRunning = true

	fmt.Printf("Terminal: started %s in %s\n", shell, a.projectRoot)

	// Send initial prompt request
	fmt.Fprintf(stdin, "cd /d \"%s\"\r\n", a.projectRoot)

	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "terminal-output", line+"\r\n")
			}
		}
		fmt.Println("Terminal: stdout pipe closed")
	}()

	// Read stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "terminal-output", line+"\r\n")
			}
		}
		fmt.Println("Terminal: stderr pipe closed")
	}()

	// Wait for process to finish
	go func() {
		err := cmd.Wait()
		a.termMu.Lock()
		a.termRunning = false
		a.termMu.Unlock()
		fmt.Printf("Terminal: process exited: %v\n", err)
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "terminal-output", "\r\n[Process exited]\r\n")
		}
	}()
}

// WriteTerminal writes input to the running shell process
func (a *App) WriteTerminal(input string) error {
	a.termMu.Lock()
	defer a.termMu.Unlock()

	if !a.termRunning || a.termStdin == nil {
		return fmt.Errorf("terminal not running")
	}

	_, err := fmt.Fprint(a.termStdin, input)
	return err
}

// StopTerminal kills the running shell process
func (a *App) StopTerminal() {
	a.termMu.Lock()
	defer a.termMu.Unlock()

	if a.termRunning && a.termCmd != nil {
		a.termCmd.Process.Kill()
		a.termCmd.Wait()
		a.termRunning = false
		a.termCmd = nil
		a.termStdin = nil
		fmt.Println("Terminal: stopped")
	}
}

// RunCommand executes a terminal command
func (a *App) RunCommand(command string, workDir string) (string, error) {
	var cmd *exec.Cmd
	
	if workDir == "" {
		workDir = a.projectRoot
	}
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetProviderModels fetches available models from a provider's API
func (a *App) GetProviderModels(providerName string) ([]string, error) {
	provider, exists := a.config.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	if provider.APIKey == "" {
		return nil, fmt.Errorf("API key not configured for %s", providerName)
	}

	switch providerName {
	case "anthropic":
		return a.fetchAnthropicModels(provider)
	default:
		return a.fetchOpenAIModels(provider)
	}
}

func (a *App) fetchOpenAIModels(provider *AIProvider) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", provider.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var models []string
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
}

func (a *App) fetchAnthropicModels(provider *AIProvider) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var models []string
	for _, m := range result.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return models, nil
}

// GetFileAtRef retrieves the version of a file at a given git ref (branch, tag, or commit)
func (a *App) GetFileAtRef(filename string, gitRef string) (string, error) {
	cmd := exec.Command("git", "show", gitRef+":"+filename)
	cmd.Dir = a.projectRoot
	output, err := cmd.Output()
	if err != nil {
		// File might not exist at that ref (untracked, new file, wrong ref)
		return "", nil
	}
	return string(output), nil
}

// GetOriginalContent retrieves the committed version of a file using git show HEAD
func (a *App) GetOriginalContent(filename string) (string, error) {
	return a.GetFileAtRef(filename, "HEAD")
}

// GetGitRefs returns a list of git refs (branches and tags) for the diff ref selector
func (a *App) GetGitRefs() ([]string, error) {
	var refs []string
	
	// Get local branches
	branchCmd := exec.Command("git", "branch", "--format=%(refname:short)")
	branchCmd.Dir = a.projectRoot
	branchOutput, err := branchCmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(branchOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				refs = append(refs, line)
			}
		}
	}
	
	// Get tags (most recent 30)
	tagCmd := exec.Command("git", "tag", "--sort=-creatordate", "--format=%(refname:short)")
	tagCmd.Dir = a.projectRoot
	tagOutput, err := tagCmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(tagOutput)), "\n")
		for _, line := range lines {
			if line != "" {
				refs = append(refs, line)
			}
		}
	}
	
	// Remove duplicates while preserving order
	seen := make(map[string]bool)
	var unique []string
	for _, ref := range refs {
		if !seen[ref] {
			seen[ref] = true
			unique = append(unique, ref)
		}
	}
	
	// Limit total refs to 100 to avoid overwhelming the dropdown
	if len(unique) > 100 {
		unique = unique[:100]
	}
	
	return unique, nil
}

// GetGitStatus runs git status --porcelain in the project and returns a map of file -> status
func (a *App) GetGitStatus() map[string]string {
	result := make(map[string]string)
	
	cmd := exec.Command("git", "status", "--porcelain", "-u")
	cmd.Dir = a.projectRoot
	output, err := cmd.Output()
	if err != nil {
		// Not a git repo or git not installed — return empty
		return result
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		// --porcelain format: XY filename
		status := strings.TrimSpace(line[:2])
		filename := strings.TrimSpace(line[2:])
		if filename == "" {
			continue
		}
		
		// Map combined status to a single letter (use the more significant one)
		// X = staging area, Y = working tree
		switch {
		case strings.Contains(status, "?") && !strings.Contains(status, "!"):
			result[filename] = "U" // Untracked
		case strings.Contains(status, "!"):
			result[filename] = "I" // Ignored
		case strings.Contains(status, "A"):
			result[filename] = "A" // Added
		case strings.Contains(status, "M"):
			result[filename] = "M" // Modified
		case strings.Contains(status, "D"):
			result[filename] = "D" // Deleted
		case strings.Contains(status, "R"):
			result[filename] = "R" // Renamed
		case strings.Contains(status, "C"):
			result[filename] = "C" // Copied
		case strings.Contains(status, "U"):
			result[filename] = "U" // Updated but unmerged
		default:
			result[filename] = "M" // Treat other changes as modified
		}
	}
	
	return result
}

// CreateFolder creates a new directory at the given path relative to project root
func (a *App) CreateFolder(path string) error {
	fullPath := filepath.Join(a.projectRoot, path)
	return os.MkdirAll(fullPath, 0755)
}

// DeleteFile deletes a file or directory at the given path relative to project root
func (a *App) DeleteFile(path string) error {
	fullPath := filepath.Join(a.projectRoot, path)
	return os.RemoveAll(fullPath)
}

// RenameFile renames/moves a file from oldPath to newPath relative to project root
func (a *App) RenameFile(oldPath string, newPath string) error {
	oldFull := filepath.Join(a.projectRoot, oldPath)
	newFull := filepath.Join(a.projectRoot, newPath)
	// Ensure target directory exists
	os.MkdirAll(filepath.Dir(newFull), 0755)
	return os.Rename(oldFull, newFull)
}

// GetProjectRoot returns the current project root directory
func (a *App) GetProjectRoot() string {
	return a.projectRoot
}

// SetProjectRoot sets a new project root directory and persists it to config
func (a *App) SetProjectRoot(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	a.projectRoot = path
	if a.config != nil {
		a.config.ProjectRoot = path
		a.saveConfig()
	}
	// Restart the file watcher for the new directory
	a.startFileWatcher()
	return nil
}

// LoadApp loads a custom app/plugin
func (a *App) LoadApp(appPath string) error {
	// TODO: Implement custom app loading system
	// This will load Go plugins or external binaries
	fmt.Printf("Loading custom app: %s\n", appPath)
	return nil
}

// BrowseForProject opens a native folder picker dialog and returns the selected path
func (a *App) BrowseForProject() (string, error) {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:            "Select Project Directory",
		DefaultDirectory: a.projectRoot,
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

// GetConfig returns the full configuration
func (a *App) GetConfig() *AgentConfig {
	return a.config
}

// ListApps returns all loaded custom apps
func (a *App) ListApps() []string {
	// TODO: Implement app listing
	return []string{}
}

// RunApp executes a custom app
func (a *App) RunApp(appName string, args []string) (string, error) {
	// TODO: Implement app execution
	return "", fmt.Errorf("not implemented")
}

func getDefaultProjectRoot() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(homeDir, "mugsoft-projects", "default")
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("MugSoft Agent starting up...")
	
	// Load config from file if exists
	configPath := filepath.Join(getConfigDir(), "mugsoft-config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		loaded := &AgentConfig{}
		if err := json.Unmarshal(data, &loaded); err != nil {
			fmt.Printf("Warning: failed to parse config file: %v\n", err)
		} else {
			fmt.Println("Config loaded from file")
			// Only copy fields that were actually saved
			if loaded.Providers != nil && len(loaded.Providers) > 0 {
				a.config.Providers = loaded.Providers
			}
			if loaded.ActiveProvider != "" {
				a.config.ActiveProvider = loaded.ActiveProvider
			}
			if loaded.ProjectRoot != "" {
				a.config.ProjectRoot = loaded.ProjectRoot
				if _, err := os.Stat(loaded.ProjectRoot); err == nil {
					a.projectRoot = loaded.ProjectRoot
				}
			}
			fmt.Printf("Loaded %d providers, active: %s\n", len(a.config.Providers), a.config.ActiveProvider)
		}
	} else {
		fmt.Println("No config file found, using defaults with 4 providers")
	}
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "MugSoft Agent",
		Width:     1280,
		Height:    720,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
}