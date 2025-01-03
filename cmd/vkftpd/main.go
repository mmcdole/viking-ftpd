package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
	"github.com/mmcdole/viking-ftpd/pkg/ftpserver"
	"github.com/mmcdole/viking-ftpd/pkg/logging"
)

const (
	version = "1.0.0"
	usage   = `VikingMUD FTP Server (vkftpd) - Secure FTP access to VikingMUD

This server integrates with VikingMUD's authentication and access control systems,
providing secure FTP access while respecting the MUD's permissions system.

Usage: vkftpd [options]

Options:
  -config string
        Path to config file (required)
  -version
        Show version information

The config file should be in JSON format with the following structure:
{
    // Core server settings
    "listen_addr": "0.0.0.0",          // Address to listen on
    "port": 2121,                      // Port to listen on

    // Directory settings
    "ftp_root_dir": "/mud/lib",        // Root directory for FTP access
    "home_pattern": "players/%s",       // Home directory pattern (%s = username)

    // MUD-specific paths
    "character_dir_path": "/mud/lib/characters",    // Path to character save files
    "access_file_path": "/mud/lib/dgd/sys/data/access.o",   // Path to MUD's access.o

    // Security settings (optional)
    "tls_cert_file": "/path/to/cert.pem",  // Path to TLS certificate file
    "tls_key_file": "/path/to/key.pem",    // Path to TLS private key file

    // Performance settings
    "passive_port_range": [2122,2150],  // Range for passive mode
    "max_connections": 10,              // Max concurrent connections
    "idle_timeout": 300,                // Idle timeout in seconds

    // Cache settings
    "character_cache_time": 60,         // How long to cache character data
    "access_cache_time": 60,            // How long to cache access permissions

    // Logging settings (optional)
    "access_log_path": "/mud/lib/log/vkftpd-access.log"  // Path to access log file
}

Paths in the config file can be relative to the config file location.
The server authenticates users against their character files and enforces
the same file access permissions as the MUD itself.
`
)

func main() {
	// Setup command line flags
	configPath := flag.String("config", "", "Path to config file (required)")
	showVersion := flag.Bool("version", false, "Show version information")

	// Override default usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s", usage)
	}

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("Viking FTP Server v%s\n", version)
		os.Exit(0)
	}

	// Check for required config
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Config file path is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Convert to absolute path if needed
	if !filepath.IsAbs(*configPath) {
		var err error
		*configPath, err = filepath.Abs(*configPath)
		if err != nil {
			log.Fatalf("Failed to get absolute path: %v", err)
		}
	}

	// Load configuration
	var config Config
	err := LoadConfig(*configPath, &config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logging
	logConfig := logging.Config{
		AccessLogPath: config.AccessLogPath,
	}
	if err := logging.Initialize(&logConfig); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}

	// Create authorizer for permission checks
	source := authorization.NewFileSource(config.AccessFilePath)
	authorizer, err := authorization.NewAuthorizer(source, time.Duration(config.AccessCacheTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to create authorizer: %v", err)
	}

	// Create authenticator
	charSource := authentication.NewFileSource(config.CharacterDirPath)
	authenticator, err := authentication.NewAuthenticator(charSource, nil, time.Duration(config.CharacterCacheTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to create authenticator: %v", err)
	}

	// Create and start FTP server
	server, err := ftpserver.New(&ftpserver.Config{
		ListenAddr:           config.ListenAddr,
		Port:                 config.Port,
		RootDir:              config.FTPRootDir,
		HomePattern:          config.HomePattern,
		PassiveTransferPorts: config.PassivePortRange,
		TLSCertFile:          config.TLSCertFile,
		TLSKeyFile:           config.TLSKeyFile,
	}, authorizer, authenticator)
	if err != nil {
		log.Fatalf("Failed to create FTP server: %v", err)
	}

	log.Printf("Starting VikingMUD FTP Server v%s on %s:%d", version, config.ListenAddr, config.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
