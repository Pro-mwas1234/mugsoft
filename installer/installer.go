package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	webview2URL = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
	installerName = "WebView2Installer.exe"
	appName = "MugSoft-Agent.exe"
)

// Windows API constants
const (
	MB_OK                = 0
	MB_ICONERROR         = 16
	MB_ICONINFORMATION   = 64
	SW_SHOW              = 5
	INFINITE             = 0xFFFFFFFF
	WAIT_OBJECT_0        = 0
)

func main() {
	// Show starting message
	showMessage("MugSoft Agent Setup", "Installing Microsoft WebView2 Runtime...\n\nThis may take a minute depending on your internet speed.")

	// Check if WebView2 is already installed
	if isWebView2Installed() {
		// Skip download, just launch the app
		launchApp()
		return
	}

	// Download WebView2 installer
	installerPath := filepath.Join(os.TempDir(), installerName)
	
	fmt.Println("Downloading WebView2 installer...")
	err := downloadFile(installerPath, webview2URL)
	if err != nil {
		showError("Download Failed", fmt.Sprintf("Failed to download WebView2:\n%v\n\nPlease download manually from:\n%s", err, webview2URL))
		os.Exit(1)
	}
	defer os.Remove(installerPath) // Clean up

	fmt.Println("Installing WebView2...")
	
	// Run installer with admin privileges and correct flags
	// Using /install /silent /no privilages needed for per-user install
	cmd := exec.Command(installerPath, "/install", "/silent")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		// Try alternative method - sometimes needs /passive instead
		cmd2 := exec.Command(installerPath, "/passive", "/norestart")
		cmd2.Stdout = os.Stdout
		cmd2.Stderr = os.Stderr
		err = cmd2.Run()
		if err != nil {
			showError("Installation Failed", fmt.Sprintf("Failed to install WebView2:\n%v\n\nError code: %v\n\nPlease run as Administrator or download WebView2 manually from:\n%s", err, err, webview2URL))
			os.Exit(1)
		}
	}

	fmt.Println("Installation complete!")
	
	// Wait a moment for registration
	// syscall.Sleep(2000)
	
	// Launch the app
	launchApp()
}

func isWebView2Installed() bool {
	// Try to locate WebView2 runtime
	cmd := exec.Command("powershell", "-Command", "Get-Package -Name 'Microsoft Edge WebView2*' -ErrorAction SilentlyContinue")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(output) > 0
}

func downloadFile(filepath string, url string) error {
	// Create HTTP client with redirect following
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Allow redirects
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func launchApp() {
	// Get the directory where the launcher is running from
	execPath, err := os.Executable()
	if err != nil {
		showError("Error", fmt.Sprintf("Failed to get executable path: %v", err))
		os.Exit(1)
	}
	
	appDir := filepath.Dir(execPath)
	appPath := filepath.Join(appDir, appName)
	
	// Check if app exists
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		showError("App Not Found", fmt.Sprintf("MugSoft-Agent.exe not found in:\n%s", appDir))
		os.Exit(1)
	}
	
	// Launch the app
	cmd := exec.Command(appPath)
	err = cmd.Start()
	if err != nil {
		showError("Launch Failed", fmt.Sprintf("Failed to start MugSoft Agent:\n%v", err))
		os.Exit(1)
	}
	
	// Exit the launcher
	os.Exit(0)
}

func showError(title, message string) {
	msg := fmt.Sprintf("%s\n\n%s", title, message)
	syscall.SyscallN(
		syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Addr(),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(msg))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		MB_OK|MB_ICONERROR,
	)
}

func showMessage(title, message string) {
	syscall.SyscallN(
		syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Addr(),
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(message))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		MB_OK|MB_ICONINFORMATION,
	)
}