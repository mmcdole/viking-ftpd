# VikingMUD FTP Daemon

A custom FTP server designed specifically for VikingMUD, providing secure file access with native integration into the MUD's authentication and authorization systems. This daemon understands LPC object serialization and directly interfaces with the MUD's character database and access control trees.

## Configuration

The server is configured via a JSON file. Example configuration:

```json
{
  "listen_addr": "0.0.0.0",
  "port": 2121,
  "ftp_root_dir": "/mud",
  "character_dir_path": "/mud/characters",
  "access_file_path": "/mud/dgd/sys/data/access.o",
  "home_pattern": "characters/%s",
  "passive_port_range": [2122, 2150],
  "max_connections": 10,
  "idle_timeout": 300,
  "character_cache_time": 60,
  "access_cache_time": 60,
  "access_log_path": "/mud/log/vkftpd-access.log"
}
```

### Configuration Options

| Option | Description |
|--------|------------|
| `listen_addr` | Network address to listen on |
| `port` | Port to listen on |
| `ftp_root_dir` | Root directory of the MUD |
| `character_dir_path` | Path to character files |
| `access_file_path` | Path to access.o file |
| `home_pattern` | Pattern for user home directories (uses %s for username) |
| `passive_port_range` | Port range for passive mode transfers [start, end] |
| `max_connections` | Maximum number of concurrent connections |
| `idle_timeout` | Connection timeout in seconds |
| `character_cache_time` | How long to cache character data (seconds) |
| `access_cache_time` | How long to cache access.o data (seconds) |
| `access_log_path` | Path to access log file |

## Docker Usage

The FTP server can be run in a Docker container. The container primarily needs access to your MUD directory structure:
- Mount your MUD root directory to `/mud` in the container
- Mount your config file to `/etc/vkftpd/config.json`

### Quick Start with Docker

1. Create your configuration file based on the sample:
```bash
cp config.sample.json config.json
```

2. Edit the config.json to match your MUD's directory structure. For example, if your MUD has this structure:
```
/mud/
  ├── characters/
  │   └── character_files...
  ├── dgd/sys/data/
  │   └── access.o
  └── log/
      └── vkftpd-access.log
```

Your config.json should look like:
```json
{
    "listen_addr": "",
    "port": 21,
    "ftp_root_dir": "/mud",
    "home_pattern": "characters/%s",
    "character_dir_path": "/mud/characters",
    "access_file_path": "/mud/dgd/sys/data/access.o",
    "access_log_path": "/mud/log/vkftpd-access.log",
    "passive_port_range": [2121, 2130],
    "max_connections": 100,
    "idle_timeout": 300,
    "character_cache_time": 300,
    "access_cache_time": 300
}
```

3. Run the container:
```bash
docker run -d \
  --name viking-ftpd \
  -p 21:21 \
  -p 2121-2130:2121-2130 \
  -v $(pwd)/config.json:/etc/vkftpd/config.json \
  -v /path/to/mud:/mud \
  ghcr.io/your-username/viking-ftpd:latest
```

Note: Replace `/path/to/mud` with the actual path to your MUD's root directory. All paths in config.json are relative to the container's `/mud` directory.

### Container File Structure

Inside the container:
- `/mud`: Your MUD root directory containing:
  - `/mud/characters`: Character files
  - `/mud/dgd/sys/data`: System data including access.o
  - `/mud/log`: Log files
- `/etc/vkftpd/config.json`: Configuration file

For example:
- If your MUD's access.o is at `/home/mud/game/dgd/sys/data/access.o` locally
- And you mount `/home/mud/game` to `/mud`
- Then in config.json, use `/mud/dgd/sys/data/access.o` as the access_file_path

## Package Overview

| Package | Description |
|---------|------------|
| `authentication` | Interfaces with VikingMUD's character database to validate user credentials. Reads player files and verifies password hashes using the MUD's native format. Character data is cached to reduce filesystem load. |
| `authorization` | Implements permission checking by parsing the MUD's `access.o` object tree. Validates user access rights against the MUD's hierarchical permission system. The access tree is cached to reduce filesystem reads. |
| `ftpserver` | Core FTP server implementation built on [ftpserverlib](https://github.com/fclairamb/ftpserverlib). Handles FTP protocol operations while integrating with MUD-specific authentication and authorization. |
| `lpc` | Parses LPC (Lars Pensjo C) serialized object format used by LPMuds. Enables direct reading of MUD's data structures like the access control tree. |
| `logging` | Provides structured logging for FTP operations with configurable output paths. Logs include operation type, status, user, and affected paths. |

## Building and Running

```bash
go build
./viking-ftpd -config config.json
