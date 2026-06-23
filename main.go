package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"net/http"
	"bytes"
	"encoding/json"
	"io"
	"syscall"
	"unsafe"
	"golang.org/x/sys/windows/registry"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

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

// AgentConfig holds the AI configuration
type AgentConfig struct {
	Providers  map[string]*AIProvider `json:"providers"`
	ActiveProvider string            `json:"activeProvider"`
}

// App struct - the main application logic
type App struct {
	ctx         context.Context
	projectRoot string
	config      *AgentConfig
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
				Model:   "nvidia/nemotron-4-340b-instruct",
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

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("MugSoft Agent starting up...")
	
	// Load config from file if exists
	configPath := filepath.Join(getConfigDir(), "mugsoft-config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &a.config)
		fmt.Println("Config loaded from file")
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
func (a *App) UpdateProvider(providerName string, apiKey string, model string, enabled bool) error {
	provider, exists := a.config.Providers[providerName]
	if !exists {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	
	if apiKey != "" {
		provider.APIKey = apiKey
	}
	if model != "" {
		provider.Model = model
	}
	provider.Enabled = enabled
	
	a.saveConfig()
	return nil
}

// SendMessage sends a message to the active AI provider and gets a response
func (a *App) SendMessage(message string, context string) (string, error) {
	provider := a.config.Providers[a.config.ActiveProvider]
	if provider == nil || !provider.Enabled {
		return "", fmt.Errorf("no active provider configured. Please set up an API key in Settings.")
	}
	
	// Build the prompt with context
	systemPrompt := "You are MugSoft, a helpful coding assistant integrated into a VSCode-like editor. Help the user with coding tasks, explain code, debug issues, and suggest improvements."
	if context != "" {
		systemPrompt += "\n\nContext:\n" + context
	}
	
	// Build request body based on provider type
	var reqBody map[string]interface{}
	
	switch a.config.ActiveProvider {
	case "anthropic":
		reqBody = map[string]interface{}{
			"model":       provider.Model,
			"max_tokens":  4096,
			"system":      systemPrompt,
			"messages": []map[string]string{
				{"role": "user", "content": message},
			},
		}
	default:
		// OpenRouter, NVIDIA, OpenAI use OpenAI-compatible format
		reqBody = map[string]interface{}{
			"model":       provider.Model,
			"max_tokens":  4096,
			"messages": []map[string]string{
				{"role": "system", "content": systemPrompt},
				{"role": "user", "content": message},
			},
		}
	}
	
	jsonData, _ := json.Marshal(reqBody)
	
	// Create HTTP request
	req, err := http.NewRequest("POST", provider.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	
	// Special headers for different providers
	if a.config.ActiveProvider == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://mugsoft.local")
		req.Header.Set("X-Title", "MugSoft Agent")
	}
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %v", err)
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

// AskAI is a convenience method for the frontend to ask the AI a question
func (a *App) AskAI(question string, files []string) (string, error) {
	context := ""
	
	// Include content from selected files
	for _, filename := range files {
		if content, err := a.ReadFile(filename); err == nil {
			context += fmt.Sprintf("\n\n--- File: %s ---\n%s", filename, content)
		}
	}
	
	return a.SendMessage(question, context)
}

// SaveConfig saves the current configuration to disk
func (a *App) saveConfig() error {
	configDir := getConfigDir()
	os.MkdirAll(configDir, 0755)
	
	configPath := filepath.Join(configDir, "mugsoft-config.json")
	data, _ := json.MarshalIndent(a.config, "", "  ")
	return os.WriteFile(configPath, data, 0644)
}

func getConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".mugsoft")
}

// domReady is called after the front-end dom has been loaded
func (a *App) domReady(ctx context.Context) {
	fmt.Println("DOM ready - UI is loaded")
}

// Returning false prevents the application from closing.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	fmt.Println("Shutting down MugSoft Agent...")
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

// GetProjectRoot returns the current project root directory
func (a *App) GetProjectRoot() string {
	return a.projectRoot
}

// SetProjectRoot sets a new project root directory
func (a *App) SetProjectRoot(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	a.projectRoot = path
	return nil
}

// LoadApp loads a custom app/plugin
func (a *App) LoadApp(appPath string) error {
	// TODO: Implement custom app loading system
	// This will load Go plugins or external binaries
	fmt.Printf("Loading custom app: %s\n", appPath)
	return nil
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

// checkWebView2 checks if WebView2 is installed and offers to install it if missing
func checkWebView2() error {
	// Check registry for WebView2 installation
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		return nil // WebView2 found
	}
	
	// Also check user registry
	key, err = registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, registry.QUERY_VALUE)
	if err == nil {
		key.Close()
		return nil // WebView2 found
	}
	
	// WebView2 not found, show message
	msg := "Microsoft WebView2 Runtime is required but not installed.\n\n"
	msg += "Would you like to download it now?\n\n"
	msg += "Click OK to open the download page in your browser,\n"
	msg += "then install and restart MugSoft Agent.\n\n"
	msg += "Download link: https://developer.microsoft.com/en-us/microsoft-edge/webview2/"
	
	// Show message box (Windows API)
	const MB_YESNO = 4
	const MB_ICONQUESTION = 32
	const IDYES = 6
	
	ret, _, _ := syscall.SyscallN(
		syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Addr(),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(msg))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("WebView2 Required"))),
		MB_YESNO|MB_ICONQUESTION,
	)
	
	if ret == IDYES {
		// Open download page
		exec.Command("cmd", "/C", "start", "https://developer.microsoft.com/en-us/microsoft-edge/webview2/").Start()
	}
	
	return fmt.Errorf("WebView2 not installed")
}

func main() {
	// Check for WebView2 first
	if err := checkWebView2(); err != nil {
		os.Exit(1)
	}
	
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
			Assets: nil, // Will be generated by wails build
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