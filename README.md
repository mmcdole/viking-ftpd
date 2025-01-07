# VikingMUD FTP Daemon

A custom FTP server designed specifically for [VikingMUD](https://www.vikingmud.org), providing secure file access with native integration into the MUD's [player authentication](docs/player_authentication.md) and hiearchical [authorization system](docs/viking_access_tree.md). This daemon understands [LPC serialized object format](https://github.com/mmcdole/viking-ftpd/blob/main/docs/lpc_object_format.md) and directly interfaces with the MUD's character database and access control trees.


## Installation

### Building
Requires Go 1.19 or higher. Build using the provided Makefile:

```bash
make build   # Creates vkftpd binary with version information
./vkftpd --version  # Verify build
```

Or build manually with `go build`.

### Running
To start the server with your configuration:

```bash
./vkftpd --config config.json
```

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
    "tls_cert_file": "/path/to/cert.pem",
    "tls_key_file": "/path/to/key.pem",
    "passive_port_range": [2122, 2150],
    "pasv_address": "your.public.ip.address",
    "pasv_ip_verify": true,
    "max_connections": 10,
    "idle_timeout": 300,
    "character_cache_time": 60,
    "access_cache_time": 60,
    "access_log_path": "/mud/lib/log/vkftpd-access.log"
}
```

### Network Settings
- `listen_addr`: Address to listen on (e.g., "0.0.0.0" for all interfaces)
- `port`: Port to listen on (default: 2121)
- `passive_port_range`: Range of ports for passive mode (default: [50000, 50100])
- `pasv_address`: Public IP address to advertise for passive mode connections (optional)
- `pasv_ip_verify`: Whether to verify that data connection IPs match control connection IP (optional, default: false)
- `max_connections`: Maximum concurrent connections (default: 10)
- `idle_timeout`: Connection idle timeout in seconds (default: 300)

### File System Configuration
- `ftp_root_dir`: Root directory for FTP access (required)
- `character_dir_path`: Path to character files directory (required)
- `access_file_path`: Path to the MUD's access.o file (required)
- `home_pattern`: Pattern for user home directories (e.g., "players/%s")

### Security
- `tls_cert_file`: Path to TLS certificate file for optional FTPS support (optional)
- `tls_key_file`: Path to TLS private key file for optional FTPS support (optional)

If TLS certificate and key files are provided, the server will support both FTP and FTPS connections. If not provided, the server will operate in FTP-only mode.

### Caching and Logging
- `character_cache_time`: How long to cache character data in seconds (default: 60)
- `access_cache_time`: How long to cache access.o data in seconds (default: 60)
- `access_log_path`: Path to access log file (optional)

## Package Overview

| Package | Description |
|---------|------------|
| `authentication` | Handles user authentication by verifying credentials against the MUD's [player authentication system](docs/player_authentication.md). Validates password hashes using the MUD's native format. |
| `authorization` | Implements permission checking by parsing the MUD's `access.o` object tree. Validates user access rights against the MUD's [hierarchical permission system](docs/viking_access_tree.md). The access tree is cached to reduce filesystem reads. |
| `ftpserver` | Core FTP server implementation built on [ftpserverlib](https://github.com/fclairamb/ftpserverlib). Handles FTP protocol operations while integrating with MUD-specific authentication and authorization. |
| `lpc` | Parses [LPC (Lars Pensjo C) serialized object format](https://github.com/mmcdole/viking-ftpd/blob/main/docs/lpc_object_format.md) used by LPMuds. Enables direct reading of MUD's data structures like the access control tree. |
| `users` | Manages user data by reading and caching the MUD's character files.  |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
