// Package config handles XDG configuration directory and file paths.
package config

import (
	"os"
	"path/filepath"
)

const (
	// AppName is the application directory name.
	AppName = "gtask"

	// OAuthClientFile is the OAuth client credentials filename.
	OAuthClientFile = "oauth_client.json"

	// TokenFile is the stored OAuth token filename.
	TokenFile = "token.json"
)

// Config holds configuration paths and settings.
type Config struct {
	// Dir is the configuration directory path.
	Dir string

	// Debug enables debug logging.
	Debug bool

	// Quiet suppresses informational output.
	Quiet bool
}

// New creates a new Config with the default or specified config directory.
// If configDir is empty, uses XDG_CONFIG_HOME/gtask or $HOME/.config/gtask.
func New(configDir string) (*Config, error) {
	dir := configDir
	if dir == "" {
		dir = DefaultConfigDir()
	}
	return &Config{Dir: dir}, nil
}

// DefaultConfigDir returns the default configuration directory.
// Uses XDG_CONFIG_HOME if set, otherwise $HOME/.config.
func DefaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home can't be determined
		return AppName
	}
	return filepath.Join(home, ".config", AppName)
}

// OAuthClientPath returns the path to the OAuth client credentials file.
func (c *Config) OAuthClientPath() string {
	return filepath.Join(c.Dir, OAuthClientFile)
}

// TokenPath returns the path to the stored OAuth token file.
func (c *Config) TokenPath() string {
	return filepath.Join(c.Dir, TokenFile)
}

// EnsureDir creates the config directory if it doesn't exist.
// Directory is created with mode 0700.
func (c *Config) EnsureDir() error {
	return os.MkdirAll(c.Dir, 0700)
}

// HasOAuthClient checks if the OAuth client credentials file exists.
func (c *Config) HasOAuthClient() bool {
	_, err := os.Stat(c.OAuthClientPath())
	return err == nil
}

// HasToken checks if the token file exists.
func (c *Config) HasToken() bool {
	_, err := os.Stat(c.TokenPath())
	return err == nil
}

// RemoveToken deletes the token file.
func (c *Config) RemoveToken() error {
	return os.Remove(c.TokenPath())
}
