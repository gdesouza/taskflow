package remote

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"taskflow/internal/config"
)

type mockGist struct {
	Main    string
	Archive string
	History []string // newest first
}

func setupEnv(t *testing.T) (tmpdir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	// configure viper fresh
	viper.Reset()
	os.Setenv("TASKFLOW_GIST_TOKEN", "dummy")
	// simulate config init with custom location
	cfgDir := filepath.Join(dir, ".config", config.AppName)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte("storage:\n  dir: "+strings.ReplaceAll(cfgDir, "\\", "/")+"\n  tasks_file: tasks.yaml\n  archive_file: tasks.archive.yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	viper.AddConfigPath(cfgDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	return dir, func() {}
}

func TestGistSync_FirstSyncAndPushFlow(t *testing.T) {
	m := &mockGist{Main: "tasks: []\n", Archive: "tasks: []\n", History: []string{"v1"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/gists") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		id := strings.TrimPrefix(r.URL.Path, "/gists/")
		_ = id
		switch r.Method {
		case http.MethodGet:
			resp := map[string]any{
				"files": map[string]map[string]string{
					"tasks.yaml":         {"content": m.Main},
					"tasks.archive.yaml": {"content": m.Archive},
				},
				"history": func() []map[string]any {
					out := []map[string]any{}
					for _, h := range m.History {
						out = append(out, map[string]any{"version": h})
					}
					return out
				}(),
			}
			_ = json.NewEncoder(w).Encode(resp)
		case http.MethodPatch:
			var body struct {
				Files map[string]struct {
					Content string `json:"content"`
				} `json:"files"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if f, ok := body.Files["tasks.yaml"]; ok {
				m.Main = f.Content
			}
			if f, ok := body.Files["tasks.archive.yaml"]; ok {
				m.Archive = f.Content
			}
			m.History = append([]string{fmt.Sprintf("v%d", len(m.History)+1)}, m.History...)
			resp := map[string]any{
				"files": map[string]map[string]string{
					"tasks.yaml":         {"content": m.Main},
					"tasks.archive.yaml": {"content": m.Archive},
				},
				"history": []map[string]any{{"version": m.History[0]}},
			}
			_ = json.NewEncoder(w).Encode(resp)
		default:
			w.WriteHeader(405)
		}
	}))
	defer server.Close()
	origBase := gistAPIBase
	gistAPIBase = server.URL
	defer func() { gistAPIBase = origBase }()

	tmp, _ := setupEnv(t)
	_ = tmp
	// create local tasks with one task after first pull scenario
	if err := os.WriteFile(filepath.Join(config.GetStorageDir(), "tasks.yaml"), []byte("tasks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config.GetStorageDir(), "tasks.archive.yaml"), []byte("tasks: []\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// First sync (no gist id yet) – need to set gist id manually in config
	viper.Set("remote.gist.id", "abc123")
	_ = viper.WriteConfig()

	// Run first sync (pull) by calling command Run manually
	GistSyncCmd.Run(GistSyncCmd, []string{})
	if v := config.GetGistLastVersion(); v == "" {
		t.Fatalf("expected version stored after first sync")
	}
	// Modify local and sync -> should push and update version/hash
	if err := os.WriteFile(filepath.Join(config.GetStorageDir(), "tasks.yaml"), []byte("tasks: [{id: 1, title: 'X'}]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	GistSyncCmd.Run(GistSyncCmd, []string{})
	if v := config.GetGistLastVersion(); v == "" {
		t.Fatalf("expected version after push")
	}
}
