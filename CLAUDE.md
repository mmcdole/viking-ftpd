# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
make build                           # Build with version info from git
go build ./cmd/vkftpd               # Build manually 
./vkftpd --version                  # Check version
./vkftpd --config config.json       # Run with config
```

### Testing
```bash
go test ./...                       # Run all tests
go test -v ./pkg/authentication/... # Run specific package tests
go test -race ./...                 # Run with race detection
```

## Architecture Overview

This is a specialized FTP server for VikingMUD that integrates with the MUD's authentication and authorization systems. The server parses LPC serialized objects and interfaces directly with the MUD's character database.

### Core Components

- **Main Entry Point** (`cmd/vkftpd/main.go`): CLI application using Cobra framework that orchestrates all components
- **FTP Server** (`pkg/ftpserver/`): Core FTP protocol handling using ftpserverlib
- **Authentication** (`pkg/authentication/`): Validates MUD player credentials using Unix crypt
- **Authorization** (`pkg/authorization/`): Permission system based on MUD's hierarchical access tree
- **LPC Parser** (`pkg/lpc/`): Parses LPC (Lars Pensjo C) serialized object format used by LPMuds
- **User Management** (`pkg/users/`): Manages MUD character data with caching
- **Logging** (`pkg/logging/`): Structured logging for access and application events

### Key Integration Points

The server directly reads MUD data files:
- Character files for authentication (password hashes)
- `access.o` file for the hierarchical permission tree
- Both are cached with configurable TTL

### Configuration

The server requires a JSON configuration file with paths to MUD directories, network settings, TLS options, and caching parameters. See README.md for complete configuration reference.

### Security Considerations

- Supports both FTP and FTPS (TLS)
- Integrates with MUD's existing permission system
- IP verification for passive mode connections
- Respects MUD's file access restrictions