package remote

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"taskflow/internal/config"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Simplified Gist backend MVP (plaintext, no merge heuristics beyond remote-wins on conflict).
// Future: add merge + encryption + conflict details.

var gistTokenEnv = "TASKFLOW_GIST_TOKEN"

func init() {
	RemoteCmd.AddCommand(GistInitCmd, GistStatusCmd, GistPullCmd, GistPushCmd, GistSyncCmd)
	GistInitCmd.Flags().Bool("public", false, "Create a public gist (default private)")
	GistSyncCmd.Flags().Bool("force", false, "Force divergence resolution with --mode")
	GistSyncCmd.Flags().String("mode", "", "When forcing: 'push' (local wins) or 'pull' (remote wins)")
}

// Config keys (extendable later).
const gistConfigKey = "remote.gist.id"

var GistSyncCmd = &cobra.Command{
	Use:   "gist-sync",
	Short: "Synchronize local tasks with gist (fast-forward or push)",
	Long: `Performs a stateful sync using stored remote.gist.last_version and remote.gist.last_local_hash.
Logic:
  - First sync (no metadata): pull remote.
  - If remote version unchanged and local hash changed: push.
  - If remote advanced and local unchanged since last sync: pull (fast-forward).
  - Divergence (both changed): requires --force with --mode=push|pull.
`,
	Run: func(cmd *cobra.Command, args []string) {
		id := getGistID()
		if id == "" {
			fmt.Println("No gist configured.")
			return
		}
		force, _ := cmd.Flags().GetBool("force")
		mode, _ := cmd.Flags().GetString("mode")
		if mode != "" && mode != "push" && mode != "pull" {
			fmt.Println("--mode must be 'push' or 'pull'")
			return
		}
		lastVer := config.GetGistLastVersion()
		lastHash := config.GetGistLastLocalHash()

		remoteMain, remoteArch, remoteVer, err := fetchGist(id)
		if err != nil {
			fmt.Printf("Fetch error: %v\n", err)
			return
		}
		localHash, err := hashLocalState()
		if err != nil {
			fmt.Printf("Local hash error: %v\n", err)
			return
		}

		// First sync: no stored version
		if lastVer == "" {
			if err := overwriteLocal(remoteMain, remoteArch); err != nil {
				fmt.Printf("Write error: %v\n", err)
				return
			}
			newHash, _ := hashLocalState()
			_ = config.SetGistSyncMeta(remoteVer, newHash)
			fmt.Println("First sync: pulled remote state.")
			return
		}

		// Remote same version as last sync
		if remoteVer == lastVer {
			if localHash == lastHash {
				fmt.Println("Already up to date (no changes).")
				return
			}
			// Local changed, remote not advanced -> push
			mainPath := config.GetTasksFilePath()
			archPath := config.GetArchiveFilePath()
			m, err := os.ReadFile(mainPath)
			if err != nil {
				fmt.Printf("Read error: %v\n", err)
				return
			}
			a, err := os.ReadFile(archPath)
			if err != nil && !os.IsNotExist(err) {
				fmt.Printf("Read error: %v\n", err)
				return
			}
			if os.IsNotExist(err) {
				a = []byte("tasks: []\n")
			}
			newVer, err := patchGist(id, string(m), string(a))
			if err != nil {
				fmt.Printf("Push error: %v\n", err)
				return
			}
			newHash, _ := hashLocalState()
			_ = config.SetGistSyncMeta(newVer, newHash)
			fmt.Println("Sync: pushed local changes.")
			return
		}

		// Remote advanced vs our stored version
		remoteHash := hashPair(remoteMain, remoteArch)

		if localHash == lastHash { // local unchanged -> fast-forward pull
			if err := overwriteLocal(remoteMain, remoteArch); err != nil {
				fmt.Printf("Write error: %v\n", err)
				return
			}
			newHash, _ := hashLocalState()
			_ = config.SetGistSyncMeta(remoteVer, newHash)
			fmt.Println("Fast-forward: pulled remote changes.")
			return
		}

		// Divergence
		if !force {
			fmt.Println("Divergence detected: both local and remote changed since last sync. Re-run with --force --mode=push or --force --mode=pull.")
			fmt.Printf("Stored local hash: %s\nCurrent local hash: %s\nRemote hash: %s\n", lastHash, localHash, remoteHash)
			return
		}
		if mode == "" {
			fmt.Println("--mode required with --force (push or pull)")
			return
		}
		if mode == "pull" {
			if err := overwriteLocal(remoteMain, remoteArch); err != nil {
				fmt.Printf("Write error: %v\n", err)
				return
			}
			newHash, _ := hashLocalState()
			_ = config.SetGistSyncMeta(remoteVer, newHash)
			fmt.Println("Forced pull applied.")
			return
		}
		// mode push
		mainPath := config.GetTasksFilePath()
		archPath := config.GetArchiveFilePath()
		m, err := os.ReadFile(mainPath)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		a, err := os.ReadFile(archPath)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		if os.IsNotExist(err) {
			a = []byte("tasks: []\n")
		}
		newVer, err := patchGist(id, string(m), string(a))
		if err != nil {
			fmt.Printf("Push error: %v\n", err)
			return
		}
		newHash, _ := hashLocalState()
		_ = config.SetGistSyncMeta(newVer, newHash)
		fmt.Println("Forced push applied.")
	},
}

var GistInitCmd = &cobra.Command{
	Use:   "gist-init",
	Short: "Initialize (or link to) a GitHub Gist for remote storage",
	Run: func(cmd *cobra.Command, args []string) {
		id := getGistID()
		if id != "" {
			fmt.Printf("Gist already configured: %s\n", id)
			return
		}
		token := os.Getenv(gistTokenEnv)
		if token == "" {
			fmt.Printf("Environment variable %s not set\n", gistTokenEnv)
			return
		}
		pub, _ := cmd.Flags().GetBool("public")
		payload := map[string]any{
			"description": "TaskFlow task storage",
			"public":      pub,
			"files": map[string]map[string]string{
				"tasks.yaml":         {"content": "tasks: []\n"},
				"tasks.archive.yaml": {"content": "tasks: []\n"},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "https://api.github.com/gists", bytes.NewReader(body))
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 201 {
			b, _ := io.ReadAll(resp.Body)
			fmt.Printf("Create failed (%d): %s\n", resp.StatusCode, string(b))
			return
		}
		var out struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&out)
		if out.ID == "" {
			fmt.Println("No gist id returned")
			return
		}
		// Persist via viper
		setGistID(out.ID)
		fmt.Printf("Created gist %s and stored in config.\n", out.ID)
	},
}

var GistStatusCmd = &cobra.Command{
	Use:   "gist-status",
	Short: "Show gist sync status",
	Run: func(cmd *cobra.Command, args []string) {
		id := getGistID()
		if id == "" {
			fmt.Println("No gist configured.")
			return
		}
		fmt.Printf("Configured gist: %s\n", id)
		// Simple HEAD-ish fetch
		if _, _, ver, err := fetchGist(id); err == nil {
			fmt.Printf("Remote reachable. Latest version: %s\n", ver)
		} else {
			fmt.Printf("Fetch error: %v\n", err)
		}
	},
}

var GistPullCmd = &cobra.Command{
	Use:   "gist-pull",
	Short: "Pull tasks from configured gist (remote wins)",
	Run: func(cmd *cobra.Command, args []string) {
		id := getGistID()
		if id == "" {
			fmt.Println("No gist configured.")
			return
		}
		mainContent, archiveContent, _, err := fetchGist(id)
		if err != nil {
			fmt.Printf("Fetch error: %v\n", err)
			return
		}
		if err := overwriteLocal(mainContent, archiveContent); err != nil {
			fmt.Printf("Write error: %v\n", err)
			return
		}
		fmt.Println("Pulled remote gist into local storage.")
	},
}

var GistPushCmd = &cobra.Command{
	Use:   "gist-push",
	Short: "Push local tasks to gist (blind overwrite)",
	Run: func(cmd *cobra.Command, args []string) {
		id := getGistID()
		if id == "" {
			fmt.Println("No gist configured.")
			return
		}
		mainPath := config.GetTasksFilePath()
		archivePath := config.GetArchiveFilePath()
		mainData, err := os.ReadFile(mainPath)
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		archiveData, err := os.ReadFile(archivePath)
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Read error: %v\n", err)
			return
		}
		if os.IsNotExist(err) {
			archiveData = []byte("tasks: []\n")
		}
		if _, err := patchGist(id, string(mainData), string(archiveData)); err != nil {
			fmt.Printf("Push error: %v\n", err)
			return
		}
		fmt.Println("Pushed local tasks to gist.")
	},
}

// Helpers

type gistResponse struct {
	Files map[string]struct {
		Content   string `json:"content"`
		Truncated bool   `json:"truncated"`
		RawURL    string `json:"raw_url"`
	} `json:"files"`
	History []struct {
		Version   string    `json:"version"`
		Committed time.Time `json:"committed_at"`
	} `json:"history"`
}

func fetchGist(id string) (main, archive, version string, err error) {
	token := os.Getenv(gistTokenEnv)
	if token == "" {
		return "", "", "", errors.New("missing token env")
	}
	req, _ := http.NewRequest("GET", "https://api.github.com/gists/"+id, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", "", fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var gr gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", "", "", err
	}
	main = extractFile(gr, "tasks.yaml")
	archive = extractFile(gr, "tasks.archive.yaml")
	if archive == "" {
		archive = "tasks: []\n"
	}
	ver := ""
	if len(gr.History) > 0 {
		ver = gr.History[0].Version
	}
	return main, archive, ver, nil
}

func extractFile(gr gistResponse, name string) string {
	f, ok := gr.Files[name]
	if !ok {
		return ""
	}
	// For truncated we would need second fetch (skipped MVP)
	return f.Content
}

func patchGist(id, mainContent, archiveContent string) (string, error) {
	token := os.Getenv(gistTokenEnv)
	if token == "" {
		return "", errors.New("missing token env")
	}
	payload := map[string]any{
		"files": map[string]map[string]string{
			"tasks.yaml":         {"content": mainContent},
			"tasks.archive.yaml": {"content": archiveContent},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PATCH", "https://api.github.com/gists/"+id, bytes.NewReader(body))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("patch failed %d: %s", resp.StatusCode, string(b))
	}
	var gr gistResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", err
	}
	ver := ""
	if len(gr.History) > 0 {
		ver = gr.History[0].Version
	}
	return ver, nil
}

// hashPair forms the same canonical hash as hashLocalState but for remote contents.
func hashPair(mainContent, archiveContent string) string {
	h := sha256.Sum256(append(append([]byte(mainContent), []byte("\n--\n")...), []byte(archiveContent)...))
	return hex.EncodeToString(h[:])
}

func overwriteLocal(mainContent, archiveContent string) error {
	mainPath := config.GetTasksFilePath()
	archPath := config.GetArchiveFilePath()
	// Basic validation: require mainContent to include 'tasks:'
	if !bytes.Contains([]byte(mainContent), []byte("tasks:")) {
		return errors.New("remote main file missing tasks: key")
	}
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(archPath, []byte(archiveContent), 0644); err != nil {
		return err
	}
	return nil
}

// hashLocalState returns SHA256 of concatenated task + archive file contents.
func hashLocalState() (string, error) {
	mainPath := config.GetTasksFilePath()
	archPath := config.GetArchiveFilePath()
	m, err := os.ReadFile(mainPath)
	if err != nil {
		return "", err
	}
	a, err := os.ReadFile(archPath)
	if err != nil {
		if os.IsNotExist(err) {
			a = []byte("tasks: []\n")
		} else {
			return "", err
		}
	}
	// Simple canonical form: main + "\n--\n" + archive
	h := sha256.Sum256(append(append(m, []byte("\n--\n")...), a...))
	return hex.EncodeToString(h[:]), nil
}

// Configuration helpers for gist ID persistence
func getGistID() string {
	return viper.GetString(gistConfigKey)
}

func setGistID(id string) error {
	viper.Set(gistConfigKey, id)
	return viper.WriteConfig()
}
