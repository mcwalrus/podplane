// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

const (
	configFileVersion    = 1
	configMinimumVersion = 1
	configFileBlankData  = `{ "config": { "version": 1 } }`
	configFileName       = "config"
	configFileType       = "json"

	// baseDirName is the application name used as the subdirectory under
	// each XDG base directory (or under ~/.podplane on macOS/Windows).
	baseDirName = "podplane"
)

// homeDir returns the user's home directory.
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	// Windows fallback
	return os.Getenv("USERPROFILE")
}

// ConfigDirectory returns the directory for long-term config and auth metadata.
//
//	Linux:   $XDG_CONFIG_HOME/podplane   (default ~/.config/podplane)
//	macOS:   ~/.podplane/config
//	Windows: %USERPROFILE%\.podplane\config
//
// The XDG_CONFIG_HOME env var overrides the default on all platforms.
func (c *Config) ConfigDirectory() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, baseDirName)
	}
	if runtime.GOOS == "linux" {
		return filepath.Join(homeDir(), ".config", baseDirName)
	}
	return filepath.Join(homeDir(), "."+baseDirName, "config")
}

// CacheDirectory returns the directory for downloaded or derived files.
//
//	Linux:   $XDG_CACHE_HOME/podplane   (default ~/.cache/podplane)
//	macOS:   ~/.podplane/cache
//	Windows: %USERPROFILE%\.podplane\cache
//
// The XDG_CACHE_HOME env var overrides the default on all platforms.
func (c *Config) CacheDirectory() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, baseDirName)
	}
	if runtime.GOOS == "linux" {
		return filepath.Join(homeDir(), ".cache", baseDirName)
	}
	return filepath.Join(homeDir(), "."+baseDirName, "cache")
}

// DataDirectory returns the directory for durable local VM/cluster data.
//
//	Linux:   $XDG_DATA_HOME/podplane   (default ~/.local/share/podplane)
//	macOS:   ~/.podplane/data
//	Windows: %USERPROFILE%\.podplane\data
//
// The XDG_DATA_HOME env var overrides the default on all platforms.
func (c *Config) DataDirectory() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, baseDirName)
	}
	if runtime.GOOS == "linux" {
		return filepath.Join(homeDir(), ".local", "share", baseDirName)
	}
	return filepath.Join(homeDir(), "."+baseDirName, "data")
}

// RuntimeDirectory returns the directory for ephemeral process/session files.
//
//	Linux:   $XDG_RUNTIME_DIR/podplane   (default ~/.podplane/run)
//	macOS:   ~/.podplane/run
//	Windows: %USERPROFILE%\.podplane\run
//
// The XDG_RUNTIME_DIR env var overrides the default on all platforms.
func (c *Config) RuntimeDirectory() string {
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, baseDirName)
	}
	if runtime.GOOS == "linux" {
		return filepath.Join(homeDir(), "."+baseDirName, "run")
	}
	return filepath.Join(homeDir(), "."+baseDirName, "run")
}

// File returns the filename of the config file
func (c *Config) File() string {
	return filepath.Join(c.ConfigDirectory(), fmt.Sprintf("%s.%s", configFileName, configFileType))
}

// InitFile configures viper to work with a config file and ensure it exists,
// then loads it
func (c *Config) InitFile() error {
	// configure viper options
	c.viperFile.SetConfigName(configFileName)
	c.viperFile.SetConfigType(configFileType)
	c.viperFile.AddConfigPath(c.ConfigDirectory())
	configDir := c.ConfigDirectory()
	configFile := c.File()

	// create parent directory/directories and ensure file exists
	info, err := os.Stat(configDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0777); err != nil {
			return fmt.Errorf("unable to create CLI config directory %s: %w", configDir, err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf("config directory path exists but is not a directory: %s", configDir)
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err := os.WriteFile(configFile, []byte(configFileBlankData), 0777)
		if err != nil {
			return fmt.Errorf("unable to create config file: %w", err)
		}
	} else {
		// load file
		if err := c.viperFile.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; that's fine for now
			} else {
				// Config file was found but another error was produced
				return err
			}
		}
		fileVersion := c.viperFile.GetInt("config.version")
		if fileVersion > configFileVersion || fileVersion < configMinimumVersion {
			return fmt.Errorf("unsupported config file version: %d", fileVersion)
		}
	}

	// set defaults
	c.viperFile.SetDefault("config.version", configFileVersion)
	c.viperFile.SetDefault("auth", map[string]AuthMetadata{})
	return nil
}

// SaveFile saves the config to a file
func (c *Config) SaveFile() error {
	return c.viperFile.WriteConfig()
}
