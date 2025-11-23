package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Config holds the application configuration.
type Config struct {
	Theme        string       `json:"theme"`
	MainColor    string       `json:"main_color"`
	SubColor     string       `json:"sub_color"`
	BorderStyle  string       `json:"border_style"`
	Search       SearchConfig `json:"search"`
	UI           UIConfig     `json:"ui"`
	DB           DBConfig     `json:"db"`
	RecentDBs    []string     `json:"recent_dbs"`
	Localization string       `json:"localization"`

	configPath string
	mu         sync.RWMutex
}

type SearchConfig struct {
	DefaultMode   string `json:"default_mode"` // "prefix" | "substring" | "regex"
	CaseSensitive bool   `json:"case_sensitive"`
	DebounceMS    int    `json:"debounce_ms"`
}

type UIConfig struct {
	PreviewChars  int `json:"preview_chars"`
	ValuePageSize int `json:"value_page_size"`
}

type DBConfig struct {
	OpenBatchSize     int    `json:"open_batch_size"`
	AutoBackupOnWrite bool   `json:"auto_backup_on_write"`
	BackupRetention   int    `json:"backup_retention"`
	BackupPath        string `json:"backup_path"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Theme:       "dark",
		MainColor:   "pastel_blue",
		SubColor:    "pastel_purple",
		BorderStyle: "rounded",
		Search: SearchConfig{
			DefaultMode:   "prefix",
			CaseSensitive: true,
			DebounceMS:    400,
		},
		UI: UIConfig{
			PreviewChars:  100,
			ValuePageSize: 4096,
		},
		DB: DBConfig{
			OpenBatchSize:     200,
			AutoBackupOnWrite: false,
			BackupRetention:   3,
			BackupPath:        "./backups",
		},
		RecentDBs:    []string{},
		Localization: "en",
	}
}

// LoadConfig loads the configuration from the given path.
// If the file does not exist or cannot be read, it returns the default configuration.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.configPath = path

	data, err := os.ReadFile(path)
	if err != nil {
		// If file doesn't exist, return default config but keep the path for saving later.
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Save persists the configuration to the file.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configPath == "" {
		// Try to determine a default path if none set, e.g., executable dir
		ex, err := os.Executable()
		if err == nil {
			c.configPath = filepath.Join(filepath.Dir(ex), "config.json")
		} else {
			c.configPath = "config.json"
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.configPath, data, 0644)
}

// AddRecentDB adds a path to the recent DBs list.
// It keeps the list unique and limits it to 5 items.
func (c *Config) AddRecentDB(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove if exists
	var newRecent []string
	for _, p := range c.RecentDBs {
		if p != path {
			newRecent = append(newRecent, p)
		}
	}

	// Prepend new path
	newRecent = append([]string{path}, newRecent...)

	// Limit to 5
	if len(newRecent) > 5 {
		newRecent = newRecent[:5]
	}

	c.RecentDBs = newRecent
}

// GetRecentDBs returns a copy of the recent DBs list.
func (c *Config) GetRecentDBs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]string, len(c.RecentDBs))
	copy(result, c.RecentDBs)
	return result
}
