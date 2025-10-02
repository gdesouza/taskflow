package config

import (
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesDefaults(t *testing.T) {
	viper.Reset()
	tempHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	if err := Init(); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	storagePath := GetStoragePath()
	if storagePath == "" {
		t.Fatalf("expected storage path to be set")
	}
	if !filepath.HasPrefix(storagePath, filepath.Join(tempHome, ".config", AppName)) {
		t.Fatalf("expected path under temp home, got %s", storagePath)
	}
	calPath := GetCalendarStoragePath()
	if calPath == "" {
		t.Fatalf("expected calendar storage path to be set")
	}
	if !filepath.HasPrefix(calPath, filepath.Join(tempHome, ".config", AppName)) {
		t.Fatalf("expected calendar path under temp home, got %s", calPath)
	}

	if _, err := os.Stat(filepath.Dir(storagePath)); err != nil {
		t.Fatalf("expected config dir to exist: %v", err)
	}
}
