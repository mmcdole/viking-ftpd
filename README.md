# VikingMUD FTP Daemon

A custom FTP server designed specifically for VikingMUD, providing secure file access with native integration into the MUD's authentication and [hierarchical authorization system](docs/viking_access_tree.md). This daemon understands LPC object serialization and directly interfaces with the MUD's character database and access control trees.

## Configuration

Create a configuration file in JSON format. Example:

```json
{
    "listen_addr": "0.0.0.0",
    "port": 2121,
    "ftp_root_dir": "/mud/lib",
    "character_dir_path": "/mud/lib/characters",
    "access_file_path": "/mud/lib/dgd/sys/data/access.o",
    "home_pattern": "players/%s",
    "passive_port_range": [2122, 2150],
    "max_connections": 10,
    "idle_timeout": 300,
    "character_cache_time": 60,
    "access_cache_time": 60,
    "access_log_path": "/mud/lib/log/vkftpd-access.log"
}
```

| Setting | Description |
|---------|-------------|
| `listen_addr` | Address to listen on (e.g., "0.0.0.0" for all interfaces) |
| `port` | Port to listen on (e.g., 2121) |
| `ftp_root_dir` | Root directory for FTP access |
| `character_dir_path` | Path to character files directory |
| `access_file_path` | Path to the MUD's access.o file |
| `home_pattern` | Pattern for user home directories (e.g., "players/%s") |
| `passive_port_range` | Range of ports for passive mode |
| `max_connections` | Maximum concurrent connections |
| `idle_timeout` | Connection idle timeout in seconds |
| `character_cache_time` | How long to cache character data (seconds) |
| `access_cache_time` | How long to cache access.o data (seconds) |
| `access_log_path` | Path to access log file |

## Package Overview

| Package | Description |
|---------|------------|
| `authentication` | Interfaces with VikingMUD's character database to validate user credentials. Reads player files and verifies password hashes using the MUD's native format. Character data is cached to reduce filesystem load. |
| `authorization` | Implements permission checking by parsing the MUD's `access.o` object tree. Validates user access rights against the MUD's [hierarchical permission system](docs/viking_access_tree.md). The access tree is cached to reduce filesystem reads. |
| `ftpserver` | Core FTP server implementation built on [ftpserverlib](https://github.com/fclairamb/ftpserverlib). Handles FTP protocol operations while integrating with MUD-specific authentication and authorization. |
| `lpc` | Parses LPC (Lars Pensjo C) serialized object format used by LPMuds. Enables direct reading of MUD's data structures like the access control tree. |
| `logging` | Provides structured logging for FTP operations with configurable output paths. Logs include operation type, status, user, and affected paths. |

## Building and Running

```bash
go build
./viking-ftpd -config config.json
