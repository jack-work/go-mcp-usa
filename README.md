# 🎭 Figaro 🎵 - Your CLI Factotum for LLMs

> *"Figaro qua, Figaro là, Figaro su, Figaro giù..."* - The Barber of Seville

## 🏰 Overview

Figaro is an extensible CLI client and library for interacting with Large Language Models (LLMs), specifically designed to work with Anthropic's Claude. Like the famous character from "The Barber of Seville" who describes himself as a factotum (a servant who can do everything), Figaro aims to be your versatile LLM helper that can perform various tasks through a flexible tool system.

## ⚔️ Features

- 🤴 Direct interaction with Claude AI models via command line
- 🏹 Support for Model Control Protocol (MCP) specification
- 🛡️ Docker container integration for running MCP servers
- 🧙‍♂️ Extensible tool framework 
- 🏺 OpenTelemetry integration for structured logging and tracing
- 👑 Persistent conversation history

## 🧩 Components

### 🎻 Core (figaro)

The central component that orchestrates the entire application:
- Manages LLM clients and tool interactions
- Processes user requests and handles conversations
- Caches tool information for efficient operation

### 🎪 AnthropicBridge

Provides seamless integration with Anthropic's Claude models:
- Converts between MCP and Anthropic tool formats
- Handles streaming responses
- Manages API authentication

### 🏛️ DockerBridge

Facilitates Docker container management for MCP servers:
- Creates, finds, and manages containers
- Handles container lifecycle and I/O streams
- Supports automatic image pulling

### 📜 JsonRPC

Implements the JSON-RPC 2.0 protocol for communication:
- Manages bidirectional channels
- Handles request/response correlation
- Provides notifications system

### 🗝️ MCP (Model Control Protocol)

Implements the Model Control Protocol specification:
- Defines tools, capabilities, and communication formats
- Manages tool discovery
- Routes tool calls to appropriate servers

### 📊 Logging

Provides comprehensive logging capabilities:
- OpenTelemetry-based structured logging
- Trace spans for detailed operation tracking
- File output with rotation

## 🧪 Installation

Currently, Figaro is in an experimental phase and can be run using:

```bash
go run .
```

## 🧙‍♂️ Usage

Basic usage:

```bash
go run . -m ModelClaude3_7SonnetLatest "Your prompt here"
```

## 🏗️ TODO

- [ ] Find a good configuration system
- [ ] Make it build properly (not just through `go run .`)
- [ ] Make it work on Windows
- [ ] Implement compatibility between model types
- [ ] Implement decorator pattern for logging
- [ ] Handle tool use more effectively
- [ ] Enhance container lifecycle management
- [ ] Formalize the console channel for output formatting
- [ ] Implement pagination for tool discovery

## 🔧 Technologies

- **Go/Golang**: Main programming language
- **Anthropic SDK**: For Claude AI models
- **Docker API**: Container management
- **JSON-RPC 2.0**: Communication protocol
- **OpenTelemetry**: Observability and tracing

## 📖 Environment Variables

Required:
- `ANTHROPIC_API_KEY`: Your Anthropic API key for Claude access

Optional (for specific MCP tools):
- Tool-specific environment variables as defined in server configurations

## 🏆 Contributing

As Figaro is currently in an experimental phase, contributions are welcome to help stabilize and extend its functionality.

## 📜 License

This project is currently in development and doesn't have a specific license yet.

---

> *"Largo al factotum della città!"* (Make way for the factotum of the city!)