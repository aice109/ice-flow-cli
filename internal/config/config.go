// Package config resolves on-disk locations and loads/saves user settings.
//
// Locations follow the XDG base directory spec on both macOS and Linux so the
// tool behaves identically on either platform. Override everything with
// IFLOW_HOME.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/aice109/ice-flow-cli/internal/buildinfo"
)

// Config is the parsed user configuration.
type Config struct {
	// DateFormat is a Go reference time layout used when rendering dates.
	DateFormat string `toml:"date_format"`
	// DefaultPriority is the priority assigned to new tasks (0=P0 .. 3=P3).
	DefaultPriority int `toml:"default_priority"`
	// Notify selects the notification backend: auto|osascript|notify-send|stdout|off.
	Notify string `toml:"notify"`
	// NotifySound requests an audible cue where supported.
	NotifySound bool `toml:"notify_sound"`
	// AutoUpdateCheck enables the once-a-day background update check.
	AutoUpdateCheck bool `toml:"auto_update_check"`
	// UpdateChannel is the release channel to follow (currently "stable").
	UpdateChannel string `toml:"update_channel"`
	// GitHubRepo is the owner/repo slug the self-updater queries.
	GitHubRepo string `toml:"github_repo"`
	// Lang is the output language: auto|en|zh-CN. "auto" follows the POSIX
	// locale environment.
	Lang string `toml:"lang"`
}

// Default returns the built-in configuration.
func Default() Config {
	return Config{
		DateFormat:      "2006-01-02",
		DefaultPriority: 2,
		Notify:          "auto",
		NotifySound:     false,
		AutoUpdateCheck: true,
		UpdateChannel:   "stable",
		GitHubRepo:      buildinfo.RepoSlug,
		Lang:            "auto",
	}
}

// HomeDir returns the IFLOW_HOME override, or "" when unset.
func HomeDir() string {
	return os.Getenv("IFLOW_HOME")
}

// configDir returns the directory holding config.toml.
func configDir() string {
	if h := HomeDir(); h != "" {
		return filepath.Join(h, "config")
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, buildinfo.AppName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", buildinfo.AppName)
}

// dataDir returns the directory holding the database and runtime state.
func dataDir() string {
	if h := HomeDir(); h != "" {
		return filepath.Join(h, "data")
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return filepath.Join(x, buildinfo.AppName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", buildinfo.AppName)
}

// ConfigDir exposes the resolved config directory.
func ConfigDir() string { return configDir() }

// DataDir exposes the resolved data directory.
func DataDir() string { return dataDir() }

// ConfigPath is the absolute path to config.toml.
func ConfigPath() string { return filepath.Join(configDir(), "config.toml") }

// DBPath is the absolute path to the SQLite database.
func DBPath() string { return filepath.Join(dataDir(), "iflow.db") }

// StatePath returns a path under the data dir for small state/cache files.
func StatePath(name string) string { return filepath.Join(dataDir(), name) }

// Load reads config.toml, returning defaults when the file is absent.
func Load() (Config, error) {
	cfg := Default()
	if _, err := os.Stat(ConfigPath()); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if _, err := toml.DecodeFile(ConfigPath(), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Save writes cfg to config.toml, creating parent directories as needed.
func Save(cfg Config) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	f, err := os.Create(ConfigPath())
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	enc.Indent = "  "
	return enc.Encode(cfg)
}

// EnsureInit creates the config file and data directory when missing and
// returns true when it created the config file.
func EnsureInit() (bool, error) {
	if err := os.MkdirAll(dataDir(), 0o755); err != nil {
		return false, err
	}
	if _, err := os.Stat(ConfigPath()); err == nil {
		return false, nil
	}
	if err := Save(Default()); err != nil {
		return false, err
	}
	return true, nil
}
