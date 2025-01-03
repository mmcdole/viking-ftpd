package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
	"github.com/mmcdole/viking-ftpd/pkg/ftpserver"
	"github.com/mmcdole/viking-ftpd/pkg/logging"
)

var version = "dev" // Will be set during build

const shortUsage = `VikingMUD FTP Server (vkftpd)

Usage: vkftpd [options]

Options:
  -config string
        Path to config file (required)
  -version
        Show version information
  -help
        Show detailed help and example configuration

Use -help for more information about configuration and usage.
`

const helpText = `VikingMUD FTP Server (vkftpd) - Secure FTP access to VikingMUD

This server integrates with VikingMUD's authentication and access control systems,
providing secure FTP access while respecting the MUD's permissions system.

Usage: vkftpd [options]

Options:
  -config string
        Path to config file (required)
  -version
        Show version information
  -help
        Show this help message

Example Configuration:
The config file should be in JSON format with the following structure:

{
    "listen_addr": "0.0.0.0",
    "port": 2121,

    "ftp_root_dir": "/mud/lib",
    "home_pattern": "players/%%s",

    "character_dir_path": "/mud/lib/characters",
    "access_file_path": "/mud/lib/dgd/sys/data/access.o",

    "tls_cert_file": "/path/to/cert.pem",
    "tls_key_file": "/path/to/key.pem",

    "passive_port_range": [2122, 2150],
    "max_connections": 10,
    "idle_timeout": 300,

    "character_cache_time": 60,
    "access_cache_time": 60,

    "access_log_path": "/mud/lib/log/vkftpd-access.log"
}`

func main() {
	// Setup command line flags
	configPath := flag.String("config", "", "Path to config file (required)")
	showVersion := flag.Bool("version", false, "Show version information")
	showHelp := flag.Bool("help", false, "Show detailed help and example configuration")

	// Override default usage to show short version
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s", shortUsage)
	}

	flag.Parse()

	// Handle help flag
	if *showHelp {
		io.WriteString(os.Stdout, helpText + "\n")
		os.Exit(0)
	}

	// Handle version flag
	if *showVersion {
		fmt.Printf("VikingMUD FTP Server %s\n", version)
		os.Exit(0)
	}

	// Check for required config
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Config file path is required\nUse -help for detailed usage and example configuration\n")
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

	fmt.Printf("Starting VikingMUD FTP Server %s on %s:%d\n", version, config.ListenAddr, config.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
