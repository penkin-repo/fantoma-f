package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// FolderConfig represents a single font folder managed by the user.
type FolderConfig struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}

// Config holds the persisted application settings.
type Config struct {
	Folders           []FolderConfig    `json:"folders"`
	ActiveFonts       []string          `json:"activeFonts"`                // Global list of active font absolute paths
	CategoryOverrides map[string]string `json:"categoryOverrides"`          // fontName → category (manual overrides)
	Theme             string            `json:"theme,omitempty"`            // "dark" (default) or "light"
	RestoreOnStartup  *bool             `json:"restoreOnStartup,omitempty"` // default true
}

// shouldRestoreOnStartup returns true unless explicitly disabled in config.
func (c *Config) shouldRestoreOnStartup() bool {
	if c == nil || c.RestoreOnStartup == nil {
		return true
	}
	return *c.RestoreOnStartup
}

// effectiveTheme returns "light" or "dark" (defaulting to "dark").
func (c *Config) effectiveTheme() string {
	if c != nil && c.Theme == "light" {
		return "light"
	}
	return "dark"
}

var configFileMu sync.Mutex

func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory if Executable() fails
		return "config.json"
	}
	dir := filepath.Dir(exePath)
	return filepath.Join(dir, "config.json")
}

func loadConfig() (*Config, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{Folders: []FolderConfig{}}, err
	}
	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return &Config{Folders: []FolderConfig{}}, err
	}
	if cfg.Folders == nil {
		cfg.Folders = []FolderConfig{}
	}
	if cfg.CategoryOverrides == nil {
		cfg.CategoryOverrides = make(map[string]string)
	}
	return &cfg, nil
}

func saveConfigSnapshot(cfg Config) error {
	configFileMu.Lock()
	defer configFileMu.Unlock()

	path := getConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename to prevent corruption on crash
	return os.Rename(tmpPath, path)
}
