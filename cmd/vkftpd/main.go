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
    "listen_addr": "0.0.0.0",          // Address to listen on
    "port": 2121,                      // Port to listen on
    "ftp_root_dir": "./root",          // Root directory for FTP access
    "character_dir_path": "./chars",    // Path to character save files
    "access_file_path": "./access.o",   // Path to MUD's access.o
    "passive_port_range": [2122,2150],  // Range for passive mode
    "max_connections": 10,              // Max concurrent connections
    "idle_timeout": 300,                // Idle timeout in seconds
    "character_cache_time": 60,         // How long to cache character data
    "access_cache_time": 60             // How long to cache access permissions
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
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
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
		PassiveTransferPorts: config.PassivePortRange,
	}, authorizer, authenticator)
	if err != nil {
		log.Fatalf("Failed to create FTP server: %v", err)
	}

	log.Printf("VikingMUD FTP Server (vkftpd) v%s starting on %s:%d", version, config.ListenAddr, config.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
