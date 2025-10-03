package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	AppName = "taskflow"
)

// small wrapper to allow testing replacement if needed
func yamlMarshal(v any) ([]byte, error) { return yaml.Marshal(v) }

// Init initializes the configuration system and migrates legacy keys.
func Init() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", AppName)
	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// New structured defaults
	viper.SetDefault("storage.dir", configDir)
	viper.SetDefault("storage.tasks_file", "tasks.yaml")
	viper.SetDefault("storage.archive_file", "tasks.archive.yaml")
	viper.SetDefault("calendar.storage.path", filepath.Join(configDir, "calendar.yaml"))

	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return err
	}

	// Read or create
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := viper.SafeWriteConfig(); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Migration from legacy storage.path
	legacyPath := viper.GetString("storage.path")
	if legacyPath != "" { // legacy key present
		dir := filepath.Dir(legacyPath)
		base := filepath.Base(legacyPath)
		if base == "" {
			base = "tasks.yaml"
		}
		archiveName := deriveArchiveName(base)
		viper.Set("storage.dir", dir)
		viper.Set("storage.tasks_file", base)
		viper.Set("storage.archive_file", archiveName)
		// Normalize: rebuild a minimal map and overwrite config file without legacy key
		cfg := map[string]any{}
		if v := viper.GetString("calendar.storage.path"); v != "" {
			cfg["calendar"] = map[string]any{"storage": map[string]any{"path": v}}
		}
		cfg["storage"] = map[string]any{
			"dir":          dir,
			"tasks_file":   base,
			"archive_file": archiveName,
		}
		if v := viper.GetString("remote.gist.id"); v != "" {
			cfg["remote"] = map[string]any{"gist": map[string]any{"id": v}}
		}
		// Write normalized config manually
		path := viper.ConfigFileUsed()
		if path == "" {
			path = filepath.Join(configDir, "config.yaml")
		}
		data, marshalErr := yamlMarshal(cfg)
		if marshalErr == nil {
			// Remove original file first to avoid partial overwrite issues
			_ = os.Remove(path)
			_ = os.WriteFile(path, data, 0644)
			viper.Set("storage.path", "") // clear legacy in memory
			_ = viper.ReadInConfig()
		}
	}

	return nil
}

// deriveArchiveName replicates naming rule used elsewhere (name.ext -> name.archive.ext)
func deriveArchiveName(tasksFile string) string {
	name := tasksFile
	ext := ""
	for i := len(tasksFile) - 1; i >= 0; i-- {
		if tasksFile[i] == '.' {
			name = tasksFile[:i]
			ext = tasksFile[i:]
			break
		}
	}
	if name == tasksFile { // no dot found
		return name + ".archive"
	}
	return name + ".archive" + ext
}

// GetStorageDir returns the directory storing task files.
func GetStorageDir() string {
	return viper.GetString("storage.dir")
}

// GetTasksFilePath returns absolute path to tasks.yaml (or configured tasks file).
func GetTasksFilePath() string {
	dir := GetStorageDir()
	name := viper.GetString("storage.tasks_file")
	if dir == "" || name == "" {
		return viper.GetString("storage.path") // fallback legacy
	}
	return filepath.Join(dir, name)
}

// GetArchiveFilePath returns absolute path to archive file.
func GetArchiveFilePath() string {
	dir := GetStorageDir()
	arch := viper.GetString("storage.archive_file")
	if dir == "" || arch == "" { // fallback derive from legacy path
		legacy := viper.GetString("storage.path")
		if legacy != "" {
			return deriveArchiveName(legacy)
		}
		return ""
	}
	return filepath.Join(dir, arch)
}

// GetStoragePath (deprecated) kept for backward compatibility.
func GetStoragePath() string { //nolint:revive
	if p := viper.GetString("storage.path"); p != "" {
		return p
	}
	return GetTasksFilePath()
}

// GetCalendarStoragePath returns the path to the calendar events YAML file.
func GetCalendarStoragePath() string {
	return viper.GetString("calendar.storage.path")
}

// Remote gist sync metadata helpers
func GetGistLastVersion() string   { return viper.GetString("remote.gist.last_version") }
func GetGistLastLocalHash() string { return viper.GetString("remote.gist.last_local_hash") }
func SetGistSyncMeta(version, localHash string) error {
	if version != "" {
		viper.Set("remote.gist.last_version", version)
	}
	if localHash != "" {
		viper.Set("remote.gist.last_local_hash", localHash)
	}
	return viper.WriteConfig()
}
