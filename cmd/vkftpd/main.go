package main

import (
	"flag"
	"log"
	"path/filepath"
	"time"

	"github.com/mmcdole/vkftpd/pkg/authn"
	"github.com/mmcdole/vkftpd/pkg/authz"
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
	source := authz.NewFileSource(config.AccessFilePath)
	authorizer, err := authz.NewAuthorizer(source, time.Duration(config.AccessCacheTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to create authorizer: %v", err)
	}

	// Create authenticator
	charSource := authn.NewFileSource(config.CharacterDirPath, "passwd")
	authenticator, err := authn.NewAuthenticator(charSource, nil, time.Duration(config.CharacterCacheTime)*time.Second)
	if err != nil {
		log.Fatalf("Failed to create authenticator: %v", err)
	}

	// TODO: Initialize and start FTP server
	log.Printf("Starting FTP server on %s:%d", config.ListenAddr, config.Port)
	_ = authenticator
	_ = authorizer
}
