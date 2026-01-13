package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aid/agentic/internal/graph"
)

// FileChange represents a file to be written
type FileChange struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// StagedChanges represents pending changes for a node
type StagedChanges struct {
	NodeID  string       `json:"node_id"`
	Files   []FileChange `json:"files"`
	Message string       `json:"message,omitempty"`
}

// Workspace represents the current working state
type Workspace struct {
	CurrentNode   string                   `json:"current_node,omitempty"`
	StagedChanges map[string]StagedChanges `json:"staged_changes"` // nodeID -> changes
	Checkpoints   []Checkpoint             `json:"checkpoints"`
	LastModified  time.Time                `json:"last_modified"`
	path          string
}

// Checkpoint represents a git commit we can rollback to
type Checkpoint struct {
	ID        string    `json:"id"`
	CommitSHA string    `json:"commit_sha"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// StateFile is the path to workspace state
const StateFile = ".agentic/state.json"

// Load loads or creates the workspace state
func Load() (*Workspace, error) {
	ws := &Workspace{
		StagedChanges: make(map[string]StagedChanges),
		path:          StateFile,
	}

	if err := os.MkdirAll(".agentic", 0755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(StateFile)
	if err == nil {
		if err := json.Unmarshal(data, ws); err != nil {
			return nil, fmt.Errorf("failed to parse workspace state: %w", err)
		}
	}

	ws.path = StateFile
	return ws, nil
}

// Save persists the workspace state
func (ws *Workspace) Save() error {
	ws.LastModified = time.Now()
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ws.path, data, 0644)
}

// StageFiles stages file changes for a node
func (ws *Workspace) StageFiles(nodeID string, files []FileChange, message string) {
	if ws.StagedChanges == nil {
		ws.StagedChanges = make(map[string]StagedChanges)
	}
	ws.StagedChanges[nodeID] = StagedChanges{
		NodeID:  nodeID,
		Files:   files,
		Message: message,
	}
}

// PrintStatus displays the current workspace status
func (ws *Workspace) PrintStatus() {
	fmt.Println("Workspace Status")
	fmt.Println("================")

	if ws.CurrentNode != "" {
		fmt.Printf("Current node: %s\n", ws.CurrentNode)
	} else {
		fmt.Println("Current node: (root)")
	}

	fmt.Printf("Staged changes: %d node(s)\n", len(ws.StagedChanges))
	for nodeID, changes := range ws.StagedChanges {
		fmt.Printf("  - %s: %d file(s)\n", nodeID, len(changes.Files))
	}

	fmt.Printf("Checkpoints: %d\n", len(ws.Checkpoints))
	if len(ws.Checkpoints) > 0 {
		latest := ws.Checkpoints[len(ws.Checkpoints)-1]
		fmt.Printf("  Latest: %s (%s)\n", latest.ID, latest.Message)
	}

	if !ws.LastModified.IsZero() {
		fmt.Printf("Last modified: %s\n", ws.LastModified.Format(time.RFC3339))
	}
}

// PrintDiff shows the staged changes
func (ws *Workspace) PrintDiff() {
	if len(ws.StagedChanges) == 0 {
		fmt.Println("No staged changes.")
		return
	}

	fmt.Println("Staged changes:")
	fmt.Println("===============")
	for nodeID, changes := range ws.StagedChanges {
		fmt.Printf("\n--- Node: %s ---\n", nodeID)
		if changes.Message != "" {
			fmt.Printf("Message: %s\n", changes.Message)
		}
		for _, f := range changes.Files {
			fmt.Printf("\nFile: %s\n", f.Path)
			// Show first 20 lines or so
			lines := strings.Split(f.Content, "\n")
			if len(lines) > 20 {
				for _, line := range lines[:20] {
					fmt.Println(line)
				}
				fmt.Printf("... (%d more lines)\n", len(lines)-20)
			} else {
				fmt.Println(f.Content)
			}
		}
	}
}

// ApplyChanges writes all staged files to disk
func (ws *Workspace) ApplyChanges() error {
	if len(ws.StagedChanges) == 0 {
		return fmt.Errorf("no staged changes to apply")
	}

	// Create checkpoint before applying
	if err := ws.CreateCheckpoint(); err != nil {
		fmt.Printf("Warning: could not create checkpoint: %v\n", err)
	}

	for nodeID, changes := range ws.StagedChanges {
		fmt.Printf("Applying changes for node: %s\n", nodeID)
		for _, f := range changes.Files {
			if err := WriteFile(f.Path, f.Content); err != nil {
				return fmt.Errorf("failed to write %s: %w", f.Path, err)
			}
			fmt.Printf("  Wrote: %s\n", f.Path)
		}
	}

	// Clear staged changes
	ws.StagedChanges = make(map[string]StagedChanges)
	return ws.Save()
}

// WriteFile writes content to a file, creating directories as needed
func WriteFile(path, content string) error {
	dir := strings.TrimSuffix(path, "/"+path[strings.LastIndex(path, "/")+1:])
	if dir != "" && dir != path {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateCheckpoint creates a git commit as a checkpoint
func (ws *Workspace) CreateCheckpoint() error {
	if !IsGitRepo() {
		return nil
	}

	sha, err := GitCurrentSHA()
	if err != nil {
		return err
	}

	checkpoint := Checkpoint{
		ID:        fmt.Sprintf("cp-%d", len(ws.Checkpoints)+1),
		CommitSHA: sha,
		Message:   fmt.Sprintf("Before applying changes at %s", time.Now().Format(time.RFC3339)),
		Timestamp: time.Now(),
	}

	ws.Checkpoints = append(ws.Checkpoints, checkpoint)

	// Limit checkpoints to last 10
	if len(ws.Checkpoints) > 10 {
		ws.Checkpoints = ws.Checkpoints[len(ws.Checkpoints)-10:]
	}

	return nil
}

// Rollback reverts to the last checkpoint
func (ws *Workspace) Rollback() error {
	if len(ws.Checkpoints) == 0 {
		return fmt.Errorf("no checkpoints available")
	}

	if !IsGitRepo() {
		return fmt.Errorf("not in a git repository")
	}

	latest := ws.Checkpoints[len(ws.Checkpoints)-1]

	cmd := exec.Command("git", "reset", "--hard", latest.CommitSHA)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git reset failed: %w\n%s", err, output)
	}

	ws.Checkpoints = ws.Checkpoints[:len(ws.Checkpoints)-1]

	fmt.Printf("Rolled back to checkpoint: %s\n", latest.ID)
	return nil
}

// IsGitRepo checks if current directory is a git repository
func IsGitRepo() bool {
	_, err := os.Stat(".git")
	return err == nil
}

// GitCurrentSHA returns the current HEAD commit SHA
func GitCurrentSHA() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// RunChecks runs the configured checks for a node
func RunChecks(node *graph.Node) error {
	if node.Meta == nil || len(node.Meta.Policies.Checks) == 0 {
		return nil
	}

	fmt.Printf("Running checks for node %s...\n", node.ID)

	for _, check := range node.Meta.Policies.Checks {
		fmt.Printf("  Running: %s\n", check)
		cmd := exec.Command("sh", "-c", check)
		cmd.Dir = node.Path
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("check failed: %s\n%s", check, output)
		}
		fmt.Printf("  Passed\n")
	}

	return nil
}
