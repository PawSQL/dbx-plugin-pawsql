package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config is the persisted plugin configuration. The password is deliberately
// never stored: after a successful login only the returned userKey (and email,
// for display) are kept, exactly like the DBeaver plugin.
type Config struct {
	Options
}

var configMu sync.Mutex

// pluginDataDir is where the sidecar keeps its own bookkeeping (config.json).
// The host injects DBX_PLUGIN_DATA_DIR; it survives upgrades and uninstalls.
func pluginDataDir() string {
	if dir := strings.TrimSpace(os.Getenv("DBX_PLUGIN_DATA_DIR")); dir != "" {
		return dir
	}
	if base, err := os.UserConfigDir(); err == nil && base != "" {
		return filepath.Join(base, "dbx", "plugins", pluginID)
	}
	return filepath.Join(".", "data")
}

func configPath() string {
	return filepath.Join(pluginDataDir(), "config.json")
}

// loadConfig reads config.json, falling back to defaults on any failure. A blank
// stored anonymous key means "not signed in yet".
func loadConfig() Config {
	cfg := Config{Options: DefaultOptions()}
	payload, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	var stored Options
	if json.Unmarshal(payload, &stored) != nil {
		return cfg
	}
	// Merge stored values over defaults; an empty server URL falls back.
	if strings.TrimSpace(stored.ServerURL) == "" {
		stored.ServerURL = defaultServerURL
	}
	cfg.Options = stored
	return cfg
}

// saveConfig atomically persists the configuration (0600, temp file + rename).
func saveConfig(cfg Config) error {
	dir := pluginDataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg.Options, "", "  ")
	if err != nil {
		return err
	}
	tmp := configPath() + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, configPath()); err != nil {
		// A few sandboxes refuse to replace an existing file; fall back to a
		// direct write so settings still persist.
		if writeErr := os.WriteFile(configPath(), payload, 0o600); writeErr != nil {
			_ = os.Remove(tmp)
			return writeErr
		}
		_ = os.Remove(tmp)
	}
	return nil
}
