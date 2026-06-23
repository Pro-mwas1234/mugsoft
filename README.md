MugSoft Agent
=============

A professional VSCode-like coding agent with integrated AI assistance, built with Go and Wails.

Overview
--------

MugSoft Agent is a standalone desktop application that combines a full-featured code editor with AI-powered assistance. It provides a familiar VSCode-like interface with syntax highlighting, file management, integrated terminal, and direct access to multiple AI providers for coding help.

Features
--------

Code Editor
- Monaco Editor engine (same as VSCode)
- Syntax highlighting for 50+ languages
- Tabbed file editing
- Auto-save functionality
- File explorer sidebar

Development Tools
- Integrated terminal with command execution
- Real-time output streaming
- Working directory management

AI Integration
- Multi-provider support (OpenRouter, NVIDIA, Anthropic, OpenAI)
- Context-aware responses using open files
- Chat interface with markdown formatting
- Code explanation and refactoring suggestions
- Error debugging assistance

User Interface
- Dark theme matching VSCode aesthetics
- Resizable panels
- Keyboard shortcuts for common actions
- Settings panel for API configuration

Supported AI Providers
----------------------

Provider        Models Available                      Best For
--------------- ------------------------------------- ----------------------------------
OpenRouter      Claude, GPT-4, Llama, Mistral +100    General purpose, cost-effective
NVIDIA          Nemotron, Llama                       Optimized inference speed
Anthropic       Claude 3.5 Sonnet, Opus, Haiku        Code quality, reasoning
OpenAI          GPT-4, GPT-4 Turbo                    General purpose, reliability

Installation
------------

Quick Install (Recommended)
1. Download MugSoft-Setup.exe
2. Double-click to run
3. The installer will:
   - Check if WebView2 is installed
   - Download and install WebView2 if needed (automatic)
   - Launch MugSoft Agent automatically

Note: The first run may take a minute to download WebView2 (~3 MB). Subsequent runs open immediately.

Manual Install
1. Download both files:
   - MugSoft-Agent.exe (main application)
   - MugSoft-Setup.exe (installer with WebView2)
2. Place both files in the same folder
3. Run MugSoft-Setup.exe to initialize

System Requirements
-------------------

Operating System: Windows 10/11 (64-bit)
Memory: 512 MB RAM minimum
Storage: 50 MB for installation
Display: 1280x720 minimum resolution
Runtime: Microsoft WebView2 (auto-installed by setup)

Building from Source
--------------------

Prerequisites:
- Go 1.21 or later
- Wails CLI v2.8.0 or later

Commands:
$ go install github.com/wailsapp/wails/v2/cmd/wails@latest
$ cd mugsoft
$ wails build

Build Outputs:
- build/bin/MugSoft-Agent.exe (main application)
- Build the installer separately from /installer directory

Usage Guide
-----------

File Management
- Open files from the Explorer sidebar
- Multiple files can be open simultaneously (tabs)
- Click tab to switch between files
- Click X on tab to close file

Editor Shortcuts
Ctrl+S          Save current file
Ctrl+`          Focus terminal
Ctrl+Shift+A    Toggle AI assistant panel
Ctrl+W          Close current tab

Terminal Usage
- Type commands in the input field at bottom
- Press Enter to execute
- Output displays in real-time
- Supports all Windows command-line commands

AI Assistant
- Open with Ctrl+Shift+A or status bar indicator
- Type questions in the text area
- AI automatically includes context from open files
- Responses include formatted code blocks

Example Queries:
- "Explain what this function does"
- "Find potential bugs in this code"
- "Suggest improvements for performance"
- "Write unit tests for this module"

Configuration
-------------

Settings are stored in: %USERPROFILE%\.mugsoft\mugsoft-config.json

Setup Instructions:
1. Launch MugSoft Agent
2. Click Settings (gear icon in status bar)
3. Select your preferred AI provider
4. Enter API key and model name
5. Click Save Changes

API Key Sources:
- OpenRouter: https://openrouter.ai/keys
- NVIDIA: https://build.nvidia.com/
- Anthropic: https://console.anthropic.com/
- OpenAI: https://platform.openai.com/api-keys

Manual Configuration Example:
{
  "providers": {
    "openrouter": {
      "name": "OpenRouter",
      "apiKey": "sk-or-...",
      "baseURL": "https://openrouter.ai/api/v1",
      "model": "anthropic/claude-3.5-sonnet",
      "enabled": true
    }
  },
  "activeProvider": "openrouter"
}

Troubleshooting
---------------

Issue: Setup shows "Download Failed"
Solution: Check internet connection, or download WebView2 manually from:
          https://developer.microsoft.com/en-us/microsoft-edge/webview2/

Issue: App won't start after installation
Solution: Verify WebView2 is installed (check in Windows Settings > Apps)

Issue: AI panel shows "Not configured"
Solution: Enter API key in Settings panel

Issue: Terminal commands not recognized
Solution: Use full command paths or verify system PATH

Issue: Editor not highlighting syntax
Solution: Check file extension is supported

Security Considerations
-----------------------

- API keys are stored locally in plaintext config file
- No data is sent to MugSoft servers
- All AI communication is direct to provider APIs
- Keep config file secure (user profile only)
- WebView2 installer downloaded from official Microsoft servers only

Distribution
------------

Files:
- MugSoft-Setup.exe (8.7 MB) - Recommended for users (includes WebView2 installer)
- MugSoft-Agent.exe (11 MB) - Standalone app (requires WebView2 pre-installed)

For most users: Distribute MugSoft-Setup.exe only
For enterprise: Pre-install WebView2, then deploy MugSoft-Agent.exe

License
-------

MIT License

Copyright (c) 2024 MugSoft

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.