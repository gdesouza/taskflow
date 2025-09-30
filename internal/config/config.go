package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	AppName = "taskflow"
)

// Init initializes the configuration system.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".config", AppName)
	viper.AddConfigPath(configPath)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set default values
	viper.SetDefault("storage.path", filepath.Join(configPath, "tasks.yaml"))
	viper.SetDefault("calendar.storage.path", filepath.Join(configPath, "calendar.yaml"))

	// Create config file if it doesn't exist
	if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
		return err
	}

	// Attempt to read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; create it with defaults
			if err := viper.SafeWriteConfig(); err != nil {
				return err
			}
		} else {
			// Another error occurred
			return err
		}
	}

	return nil
}

// GetStoragePath returns the path to the tasks YAML file.
func GetStoragePath() string {
	return viper.GetString("storage.path")
}

// GetCalendarStoragePath returns the path to the calendar events YAML file.
func GetCalendarStoragePath() string {
	return viper.GetString("calendar.storage.path")
}