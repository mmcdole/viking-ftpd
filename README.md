# Viking FTP Daemon

A custom FTP server designed specifically for VikingMUD, providing secure file access with native integration into the MUD's authentication and authorization systems. This daemon understands LPC object serialization and directly interfaces with the MUD's character database and access control trees.

## Configuration

The server is configured via a JSON file. Here's the format with available options:

```json
{
  "listen_addr": "0.0.0.0",           // Address to listen on
  "port": 2121,                       // Port to listen on
  "ftp_root_dir": "/usr/local/viking/mud/lib",          // Root directory of the MUD
  "character_dir_path": "/usr/local/viking/mud/lib/characters",  // Path to character files
  "access_file_path": "/usr/local/viking/mud/lib/dgd/sys/data/access.o",  // Path to access.o
  "home_pattern": "players/%s",        // Pattern for user home directories
  "passive_port_range": [             // Port range for passive mode transfers
    2122,                             // Start of range
    2150                              // End of range
  ],
  "max_connections": 10,              // Maximum concurrent connections
  "idle_timeout": 300,                // Connection timeout in seconds
  "character_cache_time": 60,         // How long to cache character data (seconds)
  "access_cache_time": 60,            // How long to cache access.o data (seconds)
  "access_log_path": "/usr/local/viking/mud/lib/log/vkftpd-access.log"  // Path to access log
}

## Package Overview

| Package | Description |
|---------|------------|
| `authentication` | Interfaces with VikingMUD's character database to validate user credentials. Reads player files and verifies password hashes using the MUD's native format. Character data is cached for 60 seconds to reduce filesystem load. |
| `authorization` | Implements permission checking by parsing the MUD's `access.o` object tree. Validates user access rights against the MUD's hierarchical permission system. The access tree is cached for 60 seconds after reading. |
| `ftpserver` | Core FTP server implementation built on [ftpserverlib](https://github.com/fclairamb/ftpserverlib). Handles FTP protocol operations while integrating with MUD-specific authentication and authorization. |
| `lpc` | Parses LPC (Lars Pensjo C) serialized object format used by LPMuds. Enables direct reading of MUD's data structures like the access control tree. |
| `logging` | Provides structured logging for FTP operations with configurable output paths. Logs include operation type, status, user, and affected paths. |

## Building and Running

```bash
go build
./viking-ftpd -config config.json
