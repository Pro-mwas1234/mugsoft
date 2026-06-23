MugSoft Agent
=============

A VSCode-like coding agent with custom app support and AI integration, built with Go and Wails.

Features
--------

- Full Code Editor - Monaco Editor (same as VSCode) with syntax highlighting for 50+ languages
- File Explorer - Browse and manage project files in a sidebar
- Integrated Terminal - Run commands directly in the app with output streaming
- Custom Apps System - Load and run custom plugins/apps like Hermes skills
- AI Integration - Chat with AI assistants from multiple providers (OpenRouter, NVIDIA, Anthropic, OpenAI)
- Single .exe - Bundled Windows application, no dependencies needed
- Clean Architecture - Separated CSS, modular Go backend, easy to extend

AI Providers
------------

MugSoft supports multiple AI providers out of the box:

Provider    | Description
----------- | ----------------------------------------------------------------
OpenRouter  | Access to 100+ models (Claude, GPT-4, Llama, Mistral, etc.)
NVIDIA      | NVIDIA's Nemotron and other optimized models
Anthropic   | Direct Claude API access (Claude 3.5 Sonnet, Opus, etc.)
OpenAI      | GPT-4, GPT-4 Turbo, and other OpenAI models

Configuration Steps:

1. Click the gear icon (Settings) in the status bar
2. Select your preferred provider from the list
3. Enter your API key
4. Choose the model you want to use
5. Click "Save Changes"

The active provider will be shown in the status bar with a checkmark when configured.

Project Structure
-----------------

mugsoft/
├── main.go                  Go backend with Wails bindings and AI API calls
├── go.mod                   Go module dependencies
├── go.sum                   Go module checksums
├── wails.json               Wails build configuration
├── frontend/
│   ├── index.html           Main UI with Monaco Editor and AI chat panel
│   └── style.css            All CSS styles (separated from HTML)
├── build/                   Build output directory (generated)
│   └── bin/
│       └── MugSoft-Agent.exe   Compiled Windows executable
└── README.md                This file

Building
--------

Prerequisites:

- Go 1.21 or higher
- Wails CLI v2.8.0 or higher

Installation:

1. Install Go from https://go.dev/dl/
2. Install Wails CLI:
  * go install github.com/wailsapp/wails/v2/cmd/wails@latest*
3. Clone or download this project
4. Navigate to the mugsoft directory

Development Mode:

cd mugsoft
wails dev

This will launch the app with hot-reloading. Changes to frontend files will automatically refresh.

Build .exe:

cd mugsoft
wails build

The compiled executable will be placed in build/bin/MugSoft-Agent.exe

The .exe is standalone - you can copy it anywhere and run it without Go or Wails installed.

Custom Apps
-----------

The Custom Apps panel (right sidebar) allows you to extend MugSoft with custom tools and scripts.

To add a custom app:

1. Create your app as a Go binary, Python script, or any executable
2. Place it in an apps/ directory in your project root
3. The app will appear in the Custom Apps panel
4. Click "Run" to execute it, or pass arguments via the terminal

Example custom apps:
- Code formatters (gofmt, black, prettier)
- Linters (golangci-lint, pylint, eslint)
- Test runners (go test, pytest, jest)
- Build tools (webpack, vite, esbuild)
- Deployment scripts

Keyboard Shortcuts
------------------

Shortcut      | Action
------------- | ------------------------------------
Ctrl+S        | Save current file
Ctrl+`        | Focus terminal
Ctrl+Shift+A  | Open AI Assistant panel
Ctrl+Q        | Quick file search (coming soon)
Ctrl+Shift+P  | Command palette (coming soon)

AI Usage
--------

Opening the AI Panel:

- Click the AI status indicator in the status bar, or
- Press Ctrl+Shift+A

The AI panel will slide in from the right with a chat interface.

What You Can Ask:

- "Explain this code" - AI will analyze your currently open files
- "How do I fix this error?" - Paste the error message
- "Refactor this function" - AI will suggest improvements
- "Write a test for this" - AI will generate test code
- "What does this do?" - AI will explain the logic

Context Awareness:

The AI automatically includes content from your currently open files when answering questions. This means you can ask things like:

- "Is there a bug in main.go?"
- "How are these files connected?"
- "Can you optimize this function?"

The AI will read your open files and provide contextual answers.

Configuration
-------------

AI settings are stored in:

- Windows: %USERPROFILE%/.mugsoft/mugsoft-config.json
- Linux/Mac: ~/.mugsoft/mugsoft-config.json

Configuration includes:

- Provider API keys (encrypted at rest)
- Selected models for each provider
- Active provider setting
- Custom app paths

Manual Configuration:

You can also edit the config file directly:

{
  "providers": {
    "openrouter": {
      "name": "OpenRouter",
      "apiKey": "your-key-here",
      "baseURL": "https://openrouter.ai/api/v1",
      "model": "anthropic/claude-3.5-sonnet",
      "enabled": true
    }
  },
  "activeProvider": "openrouter"
}

Troubleshooting
---------------

Problem: "No active provider configured"
Solution: Go to Settings and enter your API key for at least one provider.

Problem: Build fails with "no Go files"
Solution: Ensure main.go is in the project root, not in a subdirectory.

Problem: AI responses are slow
Solution: Try a different model or provider. Some models are faster than others.

Problem: Terminal commands not working
Solution: On Windows, commands run through cmd.exe. Use POSIX-style commands in git-bash.

License
-------

MIT License - Feel free to use, modify, and distribute.

Attribution appreciated but not required.

Credits
-------

Built with:
- Wails - https://wails.io
- Monaco Editor - https://microsoft.github.io/monaco-editor/
- Go - https://go.dev
