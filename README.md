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

Setup Instructions
------------------

1. Obtain API Key
   - OpenRouter: https://openrouter.ai/keys
   - NVIDIA: https://build.nvidia.com/
   - Anthropic: https://console.anthropic.com/
   - OpenAI: https://platform.openai.com/api-keys

2. Configure in Application
   - Launch MugSoft Agent
   - Click Settings (gear icon in status bar)
   - Select your preferred provider
   - Enter API key and model name
   - Click Save Changes

3. Start Using AI
   - Press Ctrl+Shift+A to open AI panel
   - Ask questions about your code
   - Request explanations or improvements

System Requirements
-------------------

Operating System: Windows 10/11 (64-bit)
Memory: 512 MB RAM minimum
Storage: 50 MB for installation
Display: 1280x720 minimum resolution
Dependency: WebView2 (included with Windows 10/11)

Installation
------------

Pre-built Executable
1. Download MugSoft-Agent.exe
2. Place in desired location (e.g., C:\Program Files\MugSoft\)
3. Double-click to run
4. (Optional) Create desktop shortcut

Building from Source
Prerequisites:
- Go 1.21 or later
- Wails CLI v2.8.0 or later

Commands:
$ go install github.com/wailsapp/wails/v2/cmd/wails@latest
$ cd mugsoft
$ wails build

Output: build/bin/MugSoft-Agent.exe

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

Configuration Options:
- API keys (encrypted storage)
- Active provider selection
- Model preferences per provider
- Custom editor settings

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

Issue: AI panel shows "Not configured"
Solution: Enter API key in Settings panel

Issue: Build fails with "no Go files"
Solution: Ensure main.go is in project root directory

Issue: Terminal commands not recognized
Solution: Use full command paths or verify system PATH

Issue: Editor not highlighting syntax
Solution: Check file extension is supported

Issue: Application won't start
Solution: Verify WebView2 is installed (Windows Update)

Security Considerations
-----------------------

- API keys are stored locally in plaintext config file
- No data is sent to MugSoft servers
- All AI communication is direct to provider APIs
- Keep config file secure (user profile only)

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