MugSoft Agent

A VSCode-like coding agent with custom app support and AI integration, built with Go and Wails.

Features

- Full Code Editor - Monaco Editor (same as VSCode) with syntax highlighting
- File Explorer - Browse and manage project files
- Integrated Terminal - Run commands directly in the app
- Custom Apps System - Load and run custom plugins/apps like Hermes skills
- AI Integration - Chat with AI assistants from multiple providers
- Single .exe - Bundled Windows application
- Separated CSS - Clean stylesheet in frontend/style.css

AI Providers

MugSoft supports multiple AI providers:

- OpenRouter - Access to 100+ models (Claude, GPT-4, Llama, etc.)
- NVIDIA - NVIDIA's Nemotron and other models
- Anthropic - Direct Claude API access
- OpenAI - GPT-4 and other OpenAI models

Configure your provider in Settings (click the gear icon in the status bar):
1. Enter your API key
2. Select the model you want to use
3. Click "Save Changes"

Project Structure

mugsoft/
├── backend/
│   └── main.go              Go backend with Wails bindings and AI API calls
├── frontend/
│   ├── index.html           UI with Monaco Editor and AI chat panel
│   └── style.css            All CSS styles (separated from HTML)
├── go.mod                   Go module file
└── README.md                This file

Building

Prerequisites:
- Go 1.21+
- Wails CLI: go install github.com/wailsapp/wails/v2/cmd/wails@latest

Development:
cd mugsoft
wails dev

Build .exe:
wails build

The compiled .exe will be in build/bin/.

Custom Apps

Custom apps can be loaded dynamically. Place your app binaries or scripts in the apps/ directory and they'll be available in the Custom Apps panel.

Keyboard Shortcuts

- Ctrl+S - Save current file
- Ctrl+` - Focus terminal
- Ctrl+Shift+A - Open AI Assistant panel
- Ctrl+P - Quick open file (coming soon)

AI Usage

Open the AI panel with Ctrl+Shift+A or by clicking the AI status indicator. You can:
- Ask coding questions
- Get explanations about your open files
- Request code refactoring suggestions
- Debug errors

The AI will automatically include context from your currently open files.

Configuration

AI settings are stored in ~/.mugsoft/mugsoft-config.json. The config includes:
- Provider API keys
- Selected models
- Active provider

License

MIT