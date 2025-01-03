package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
	"github.com/mmcdole/viking-ftpd/pkg/ftpserver"
	"github.com/mmcdole/viking-ftpd/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	version     = "dev" // Will be set during build
	cfgFile     string
	showVersion bool
)

func main() {
	cobra.CheckErr(rootCmd.Execute())
}

var rootCmd = &cobra.Command{
	Use:          "vkftpd",
	Short:        "VikingMUD FTP Server",
	SilenceUsage: false,
	SilenceErrors: true,
	Long: `VikingMUD FTP Server (vkftpd) - Secure FTP access to VikingMUD

This server integrates with VikingMUD's authentication and access control systems,
providing secure FTP access while respecting the MUD's permissions system.

Configuration file must be in JSON format with the following structure:
{
    "listen_addr": "0.0.0.0",
    "port": 2121,
    "ftp_root_dir": "/mud/lib",
    "home_pattern": "players/%s",
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
}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Printf("VikingMUD FTP Server %s\n", version)
			return nil
		}

		if cfgFile == "" {
			return fmt.Errorf("config file is required (use --config)")
		}

		// Convert to absolute path if needed
		if !filepath.IsAbs(cfgFile) {
			var err error
			cfgFile, err = filepath.Abs(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %v", err)
			}
		}

		// Load configuration
		var config Config
		if err := LoadConfig(cfgFile, &config); err != nil {
			return fmt.Errorf("failed to load config: %v", err)
		}

		// Initialize logging
		logConfig := logging.Config{
			AccessLogPath: config.AccessLogPath,
		}
		if err := logging.Initialize(&logConfig); err != nil {
			return fmt.Errorf("failed to initialize logging: %v", err)
		}

		// Create authorizer for permission checks
		source := authorization.NewFileSource(config.AccessFilePath)
		authorizer, err := authorization.NewAuthorizer(source, time.Duration(config.AccessCacheTime)*time.Second)
		if err != nil {
			return fmt.Errorf("failed to create authorizer: %v", err)
		}

		// Create authenticator
		charSource := authentication.NewFileSource(config.CharacterDirPath)
		authenticator, err := authentication.NewAuthenticator(charSource, nil, time.Duration(config.CharacterCacheTime)*time.Second)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %v", err)
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
			return fmt.Errorf("failed to create FTP server: %v", err)
		}

		fmt.Printf("Starting VikingMUD FTP Server %s on %s:%d\n", version, config.ListenAddr, config.Port)
		return server.ListenAndServe()
	},
}

func init() {
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "path to config file (required)")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "show version information")
}
