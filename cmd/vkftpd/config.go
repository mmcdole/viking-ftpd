package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the FTP server configuration
type Config struct {
	// Server configuration
	ListenAddr         string `json:"listen_addr"`
	Port               int    `json:"port"`
	FTPRootDir         string `json:"ftp_root_dir"`      // Root directory for FTP access
	HomePattern        string `json:"home_pattern"`      // Pattern for user home directories (e.g., "players/%s")
	// MUD-specific paths
	CharacterDirPath  string `json:"character_dir_path"` // Path to character files directory
	AccessFilePath    string `json:"access_file_path"`   // Path to the MUD's access.o file
	// Logging configuration (optional)
	AccessLogPath     string `json:"access_log_path,omitempty"`    // Optional: Path to access log file
	// Optional settings
	PassivePortRange [2]int `json:"passive_port_range"` // Range of ports for passive mode
	MaxConnections   int    `json:"max_connections"`    // Maximum concurrent connections
	IdleTimeout      int    `json:"idle_timeout"`       // Connection idle timeout in seconds

	// Cache settings
	CharacterCacheTime int `json:"character_cache_time"` // How long to cache character data (seconds)
	AccessCacheTime    int `json:"access_cache_time"`    // How long to cache permissions (seconds)
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parsing config file: %w", err)
	}

	// Convert relative paths to absolute paths based on config file location
	configDir := filepath.Dir(path)
	if !filepath.IsAbs(config.FTPRootDir) {
		config.FTPRootDir = filepath.Join(configDir, config.FTPRootDir)
	}
	if !filepath.IsAbs(config.CharacterDirPath) {
		config.CharacterDirPath = filepath.Join(configDir, config.CharacterDirPath)
	}
	if !filepath.IsAbs(config.AccessFilePath) {
		config.AccessFilePath = filepath.Join(configDir, config.AccessFilePath)
	}

	// Only convert log paths to absolute if they are specified and not absolute
	if config.AccessLogPath != "" && !filepath.IsAbs(config.AccessLogPath) {
		config.AccessLogPath = filepath.Join(configDir, config.AccessLogPath)
	}

	// Set defaults for optional settings
	if config.Port == 0 {
		config.Port = 2121
	}
	if config.PassivePortRange[0] == 0 {
		config.PassivePortRange[0] = 50000
	}
	if config.PassivePortRange[1] == 0 {
		config.PassivePortRange[1] = 50100
	}
	if config.MaxConnections == 0 {
		config.MaxConnections = 10
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 300 // 5 minutes
	}
	if config.CharacterCacheTime == 0 {
		config.CharacterCacheTime = 60 // 1 minute
	}
	if config.AccessCacheTime == 0 {
		config.AccessCacheTime = 60 // 1 minute
	}

	return nil
}
