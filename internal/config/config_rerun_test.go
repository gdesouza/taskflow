package config

import (
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"testing"
)

// Ensures calling Init twice does not error when config already exists.
func TestInitIdempotent(t *testing.T) {
	viper.Reset()
	dir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	// First init (creates config)
	if err := Init(); err != nil {
		t.Fatalf("first init: %v", err)
	}
	cfgDir := filepath.Join(dir, ".config", AppName)
	if _, err := os.Stat(filepath.Join(cfgDir, "config.yaml")); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	// Second init should succeed without attempting SafeWriteConfig (config exists)
	if err := Init(); err != nil {
		t.Fatalf("second init: %v", err)
	}
}
