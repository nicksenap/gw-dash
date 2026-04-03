// Package grove reads Grove state and config files.
// This package has zero dependencies on the grove CLI — it reads the files directly.
package grove

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Dir returns the Grove directory, preferring the GROVE_DIR env var.
func Dir() string {
	if d := os.Getenv("GROVE_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".grove")
}

// Workspace is a workspace from state.json.
type Workspace struct {
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Branch    string         `json:"branch"`
	CreatedAt string         `json:"created_at"`
	Repos     []RepoWorktree `json:"repos"`
}

// RepoWorktree is a single repo's worktree within a workspace.
type RepoWorktree struct {
	RepoName     string `json:"repo_name"`
	SourceRepo   string `json:"source_repo"`
	WorktreePath string `json:"worktree_path"`
	Branch       string `json:"branch"`
}

// State holds the loaded Grove state.
type State struct {
	Workspaces []Workspace
}

// LoadState reads state.json from the given Grove directory.
func LoadState(groveDir string) (*State, error) {
	path := filepath.Join(groveDir, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return &State{}, nil
	}

	var workspaces []Workspace
	if err := json.Unmarshal(data, &workspaces); err != nil {
		return nil, err
	}

	return &State{Workspaces: workspaces}, nil
}
