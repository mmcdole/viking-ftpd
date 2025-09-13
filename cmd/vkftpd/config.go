package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the FTP server configuration
type Config struct {
	// Core server settings
	ListenAddr     string `json:"listen_addr"`     // Address to listen on (e.g., "0.0.0.0")
	Port           int    `json:"port"`            // Port to listen on (e.g., 2121)
	MaxConnections int    `json:"max_connections"` // Maximum concurrent connections
	IdleTimeout    int    `json:"idle_timeout"`    // Connection idle timeout in seconds
	FTPRootDir     string `json:"ftp_root_dir"`    // Root directory that FTP users will be restricted to
	HomePattern    string `json:"home_pattern"`    // Pattern for user home directories (e.g., "players/%s")

	// Transfer settings
	PasvPortRange [2]int `json:"pasv_port_range"` // Range of ports for passive mode transfers
	PasvAddress   string `json:"pasv_address"`    // Public IP for passive mode connections
	PasvIPVerify  bool   `json:"pasv_ip_verify"`  // Whether to verify data connection IPs

	// Security settings
	TLSCertFile string `json:"tls_cert_file"` // Path to TLS certificate file
	TLSKeyFile  string `json:"tls_key_file"`  // Path to TLS private key file

	// MUD-specific paths
	CharacterDirPath string `json:"character_dir_path"` // Path to character files directory
	AccessFilePath   string `json:"access_file_path"`   // Path to the MUD's access.o file

	// Cache settings
	CharacterCacheTime int `json:"character_cache_time"` // How long to cache character data (seconds)
	AccessCacheTime    int `json:"access_cache_time"`    // How long to cache access data (seconds)

	// Logging settings
	AccessLogPath string `json:"access_log_path"` // Path to access log file
	AppLogPath    string `json:"app_log_path"`    // Path to application log file
	LogLevel      string `json:"log_level"`       // Log level (debug, info, warn, error, panic)
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
	if config.AppLogPath != "" && !filepath.IsAbs(config.AppLogPath) {
		config.AppLogPath = filepath.Join(configDir, config.AppLogPath)
	}

	// Convert TLS paths to absolute if specified and not absolute
	if config.TLSCertFile != "" && !filepath.IsAbs(config.TLSCertFile) {
		config.TLSCertFile = filepath.Join(configDir, config.TLSCertFile)
	}
	if config.TLSKeyFile != "" && !filepath.IsAbs(config.TLSKeyFile) {
		config.TLSKeyFile = filepath.Join(configDir, config.TLSKeyFile)
	}

	// Set defaults for optional settings
	if config.Port == 0 {
		config.Port = 2121
	}
	if config.PasvPortRange[0] == 0 {
		config.PasvPortRange[0] = 50000
	}
	if config.PasvPortRange[1] == 0 {
		config.PasvPortRange[1] = 50100
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
