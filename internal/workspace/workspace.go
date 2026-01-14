package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/aid/agentic/internal/bundle"
	"github.com/aid/agentic/internal/graph"
	"github.com/aid/agentic/internal/policy"
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

// LastApplied stores original file contents before the most recent apply
type LastApplied struct {
	NodeID    string       `json:"node_id"`
	Files     []FileChange `json:"files"`     // original content before apply
	Timestamp time.Time    `json:"timestamp"`
}

// Workspace represents the current working state
type Workspace struct {
	mu            sync.Mutex               `json:"-"` // mutex for thread-safe access
	CurrentNode   string                   `json:"current_node,omitempty"`
	StagedChanges map[string]StagedChanges `json:"staged_changes"` // nodeID -> changes
	DirtyNodes    map[string]string        `json:"dirty_nodes"`    // nodeID -> reason
	Checkpoints   []Checkpoint             `json:"checkpoints"`
	LastApplied   *LastApplied             `json:"last_applied,omitempty"`
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
		DirtyNodes:    make(map[string]string),
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

	// Initialize DirtyNodes map if nil (for existing state files)
	if ws.DirtyNodes == nil {
		ws.DirtyNodes = make(map[string]string)
	}

	ws.path = StateFile
	return ws, nil
}

// Save persists the workspace state
func (ws *Workspace) Save() error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.LastModified = time.Now()
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ws.path, data, 0644)
}

// MarkDirty marks a node as dirty with a reason
func (ws *Workspace) MarkDirty(nodeID, reason string) {
	if ws.DirtyNodes == nil {
		ws.DirtyNodes = make(map[string]string)
	}
	ws.DirtyNodes[nodeID] = reason
}

// ClearDirty removes a node from the dirty set
func (ws *Workspace) ClearDirty(nodeID string) {
	if ws.DirtyNodes != nil {
		delete(ws.DirtyNodes, nodeID)
	}
}

// StageFiles stages file changes for a node
func (ws *Workspace) StageFiles(nodeID string, files []FileChange, message string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

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

	fmt.Printf("Dirty nodes: %d\n", len(ws.DirtyNodes))
	for nodeID, reason := range ws.DirtyNodes {
		fmt.Printf("  - %s: %s\n", nodeID, reason)
	}

	fmt.Printf("Checkpoints: %d\n", len(ws.Checkpoints))
	if len(ws.Checkpoints) > 0 {
		latest := ws.Checkpoints[len(ws.Checkpoints)-1]
		fmt.Printf("  Latest: %s (%s)\n", latest.ID, latest.Message)
	}

	if ws.LastApplied != nil {
		fmt.Printf("Undo available: %s (%d file(s), applied %s)\n",
			ws.LastApplied.NodeID,
			len(ws.LastApplied.Files),
			ws.LastApplied.Timestamp.Format(time.RFC3339))
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

// Validate checks if Go source code is syntactically valid using gofmt
func Validate(path, content string) error {
	if !strings.HasSuffix(path, ".go") {
		return nil
	}

	// Use gofmt to check syntax (works without full project context)
	cmd := exec.Command("gofmt", "-e")
	cmd.Stdin = strings.NewReader(content)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax error in %s: %s", path, strings.TrimSpace(string(output)))
	}

	return nil
}

// buildFilePaths creates a diff-like string from file paths for policy evaluation
func buildFilePaths(files []FileChange) string {
	var sb strings.Builder
	for _, f := range files {
		// Format as unified diff header so policy.ExtractFilePaths can parse it
		sb.WriteString(fmt.Sprintf("--- a/%s\n", f.Path))
		sb.WriteString(fmt.Sprintf("+++ b/%s\n", f.Path))
	}
	return sb.String()
}

// ApplyChanges writes all staged files to disk
func (ws *Workspace) ApplyChanges() error {
	if len(ws.StagedChanges) == 0 {
		return fmt.Errorf("no staged changes to apply")
	}

	// Load the graph to get node metadata
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return fmt.Errorf("failed to load graph: %w", err)
	}

	// Evaluate policies for each node with staged changes
	var allViolations []policy.Violation
	var allWarnings []policy.Violation

	for nodeID, changes := range ws.StagedChanges {
		node := g.GetNode(nodeID)
		if node == nil {
			return fmt.Errorf("unknown node: %s", nodeID)
		}

		// Build bundle for the node
		b, err := bundle.Build(node)
		if err != nil {
			return fmt.Errorf("failed to build bundle for %s: %w", nodeID, err)
		}

		// Build file paths string for policy evaluation
		filePaths := buildFilePaths(changes.Files)

		// Evaluate policies
		result := policy.Evaluate(node, b, filePaths)

		// Collect violations and warnings
		for _, v := range result.Violations {
			if v.Severity == "error" {
				allViolations = append(allViolations, v)
			} else if v.Severity == "warning" {
				allWarnings = append(allWarnings, v)
			}
		}
	}

	// If any violations (errors), return error with all violations
	if len(allViolations) > 0 {
		var sb strings.Builder
		sb.WriteString("policy violations:\n")
		for _, v := range allViolations {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", v.Severity, v.Policy, v.Message))
		}
		return fmt.Errorf("%s", sb.String())
	}

	// Print warnings but continue
	for _, w := range allWarnings {
		fmt.Printf("Warning [%s]: %s\n", w.Policy, w.Message)
	}

	// Create checkpoint before applying
	if err := ws.CreateCheckpoint(); err != nil {
		fmt.Printf("Warning: could not create checkpoint: %v\n", err)
	}

	// Collect all files to be modified and read their current content for undo
	var originalFiles []FileChange
	var lastNodeID string
	for nodeID, changes := range ws.StagedChanges {
		lastNodeID = nodeID
		for _, f := range changes.Files {
			// Read current content (empty string if file doesn't exist)
			content, err := os.ReadFile(f.Path)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to read %s for undo backup: %w", f.Path, err)
			}
			originalFiles = append(originalFiles, FileChange{
				Path:    f.Path,
				Content: string(content),
			})
		}
	}

	// Store original content in LastApplied for undo
	ws.LastApplied = &LastApplied{
		NodeID:    lastNodeID,
		Files:     originalFiles,
		Timestamp: time.Now(),
	}

	for nodeID, changes := range ws.StagedChanges {
		fmt.Printf("Applying changes for node: %s\n", nodeID)
		for _, f := range changes.Files {
			if err := Validate(f.Path, f.Content); err != nil {
				return fmt.Errorf("validation failed for %s: %w", f.Path, err)
			}
			if err := WriteFile(f.Path, f.Content); err != nil {
				return fmt.Errorf("failed to write %s: %w", f.Path, err)
			}
			fmt.Printf("  Wrote: %s\n", f.Path)
		}
	}

	// Project-level validation: Run go build to catch type errors, import issues,
	// and other problems that require full project context. Unlike per-file gofmt
	// validation, this checks cross-file dependencies and package-level correctness.
	// If build fails, we print errors but do NOT rollback automatically - the user
	// can review the errors and decide whether to rollback or fix manually.
	cmd := exec.Command("go", "build", "./...")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Build validation failed:\n%s\n", output)
		fmt.Printf("Changes have been written. Use 'undo' or 'rollback' to revert if needed.\n")
	}

	// Clear staged changes
	ws.StagedChanges = make(map[string]StagedChanges)
	return ws.Save()
}

// Undo reverts the last applied changes by restoring original file contents
func (ws *Workspace) Undo() error {
	if ws.LastApplied == nil {
		return fmt.Errorf("no changes to undo")
	}

	fmt.Printf("Undoing changes for node: %s\n", ws.LastApplied.NodeID)

	for _, f := range ws.LastApplied.Files {
		if err := WriteFile(f.Path, f.Content); err != nil {
			return fmt.Errorf("failed to restore %s: %w", f.Path, err)
		}
		fmt.Printf("  Restored: %s\n", f.Path)
	}

	// Clear LastApplied after successful undo
	ws.LastApplied = nil

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