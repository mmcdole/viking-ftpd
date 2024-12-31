package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/mmcdole/viking-ftpd/pkg/authentication"
	"github.com/mmcdole/viking-ftpd/pkg/authorization"
	"github.com/mmcdole/viking-ftpd/pkg/ftpserver"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	if *configPath == "" {
		log.Fatal("Config file path is required")
	}

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
		Port:                config.Port,
		RootDir:             config.FTPRootDir,
		PassiveTransferPorts: config.PassivePortRange,
	}, authorizer, authenticator)
	if err != nil {
		log.Fatalf("Failed to create FTP server: %v", err)
	}

	log.Printf("Starting FTP server on %s:%d", config.ListenAddr, config.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
