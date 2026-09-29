package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the application configuration.
type Config struct {
	AppHost         string
	AppPort         int
	DashboardHost   string
	DashboardPort   int
	ProxyHost       string
	ProxyPort       int
	DBPath          string
	MaxBodySize     int64
	RedactedHeaders []string
	NoTunnel        bool
}

// DefaultRedactedHeaders returns the default list of sensitive headers to redact.
func DefaultRedactedHeaders() []string {
	return []string{
		"authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"x-auth-token",
	}
}

// NewDefaultConfig returns a configuration with sensible defaults.
func NewDefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	dbPath := "tunnel.db"
	if err == nil {
		appDir := filepath.Join(homeDir, ".tunnel-inspector")
		_ = os.MkdirAll(appDir, 0755)
		dbPath = filepath.Join(appDir, "requests.db")
	}

	return &Config{
		AppHost:         "localhost",
		AppPort:         8000,
		DashboardHost:   "127.0.0.1",
		DashboardPort:   4040,
		ProxyHost:       "127.0.0.1",
		ProxyPort:       4041,
		DBPath:          dbPath,
		MaxBodySize:     1024 * 1024, // 1 MB
		RedactedHeaders: DefaultRedactedHeaders(),
		NoTunnel:        false,
	}
}

// UpstreamURL returns the URL of the target application.
func (c *Config) UpstreamURL() string {
	return fmt.Sprintf("http://%s:%d", c.AppHost, c.AppPort)
}

// DashboardURL returns the local URL where the dashboard is served.
func (c *Config) DashboardURL() string {
	return fmt.Sprintf("http://%s:%d", c.DashboardHost, c.DashboardPort)
}

// ProxyURL returns the URL for the inspection proxy.
func (c *Config) ProxyURL() string {
	return fmt.Sprintf("http://%s:%d", c.ProxyHost, c.ProxyPort)
}
