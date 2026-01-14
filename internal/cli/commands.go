package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/aid/agentic/internal/brain"
	"github.com/aid/agentic/internal/bundle"
	"github.com/aid/agentic/internal/graph"
	"github.com/aid/agentic/internal/token"
	"github.com/aid/agentic/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	Version = "0.1.0"
	cfgFile string
	verbose bool
)

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

// verboseLog prints a message if verbose mode is enabled
func verboseLog(format string, args ...interface{}) {
	if verbose {
		fmt.Printf("[verbose] "+format+"\n", args...)
	}
}

// verboseBundle prints bundle details if verbose mode is enabled
func verboseBundle(b *bundle.Bundle) {
	if !verbose {
		return
	}
	fmt.Println("\n--- Bundle Details ---")
	fmt.Printf("Node: %s\n", b.NodeID)
	fmt.Printf("Hash: %s\n", b.Hash)
	fmt.Printf("Files: %d\n", len(b.Files))
	fmt.Printf("Size: %d bytes\n", b.TotalSize)
	fmt.Printf("Tokens: ~%d\n", b.EstimateTokens())

	var paths []string
	for p := range b.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		content := b.Files[p]
		fmt.Printf("  %s (%d bytes, ~%d tokens)\n", p, len(content), token.EstimateString(content))
	}
	fmt.Println("----------------------")
}

// verbosePrompt prints the LLM prompt if verbose mode is enabled
func verbosePrompt(prompt string) {
	if !verbose {
		return
	}
	fmt.Println("\n--- LLM Prompt ---")
	if len(prompt) > 3000 {
		fmt.Printf("%s\n...[truncated, %d total chars, ~%d tokens]\n", prompt[:3000], len(prompt), token.EstimateString(prompt))
	} else {
		fmt.Println(prompt)
	}
	fmt.Println("------------------")
}

var rootCmd = &cobra.Command{
	Use:   "agentic",
	Short: "Orchestrate graph-of-graphs with AI brains",
	Long: `Agentic is a CLI tool that orchestrates a recursive graph-of-graphs
architecture with pluggable AI brain adapters.

Run without arguments to enter interactive mode.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInteractive()
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize agentic in current directory",
	Long:  `Creates .agentic/ directory and validates existing GRAPH.manifest if present.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		discover, _ := cmd.Flags().GetBool("discover")
		return runInit(discover)
	},
}

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Display the dependency graph",
	Long:  `Parses GRAPH.manifest and displays the node dependency tree.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGraph()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current status",
	Long:  `Displays staged diffs, current context, and pending operations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

var runTaskCmd = &cobra.Command{
	Use:   "run [request]",
	Short: "Run a task with the AI brain",
	Long:  `Sends a request to the AI brain with the current node context.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		node, _ := cmd.Flags().GetString("node")
		parallel, _ := cmd.Flags().GetBool("parallel")
		return runTask(args[0], node, parallel)
	},
}

var planCmd = &cobra.Command{
	Use:   "plan [request]",
	Short: "Generate execution plan without making changes",
	Long:  `Analyzes the request and shows which nodes would be affected.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPlan(args[0])
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show pending diffs",
	Long:  `Displays all staged changes waiting for approval.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiff()
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply staged changes",
	Long:  `Applies all staged diffs to the working directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		skipChecks, _ := cmd.Flags().GetBool("skip-checks")
		return runApply(yes, skipChecks)
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback last applied changes",
	Long:  `Reverts the last applied changeset using git.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRollback()
	},
}

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo last applied changes",
	Long:  `Reverts the last applied changes without using git.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUndo()
	},
}

var enterCmd = &cobra.Command{
	Use:   "enter [node]",
	Short: "Enter a node context",
	Long:  `Sets the working context to a specific node for focused operations.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEnter(args[0])
	},
}

var splitCmd = &cobra.Command{
	Use:   "split [node]",
	Short: "Split a node that exceeds token budget",
	Long:  `Converts a leaf node into a composite with smaller children.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSplit(args[0])
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .agentic/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "show detailed output including token counts and prompts")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(runTaskCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(undoCmd)
	rootCmd.AddCommand(enterCmd)
	rootCmd.AddCommand(splitCmd)

	initCmd.Flags().Bool("discover", false, "auto-discover packages and generate GRAPH.manifest")
	runTaskCmd.Flags().StringP("node", "n", "", "target node for the task")
	runTaskCmd.Flags().BoolP("parallel", "P", false, "process independent nodes in parallel")
	applyCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
	applyCmd.Flags().Bool("skip-checks", false, "skip running checks after apply")
}

func runInit(discover bool) error {
	fmt.Println("Initializing agentic...")

	// Create .agentic directory
	if err := os.MkdirAll(".agentic", 0755); err != nil {
		return fmt.Errorf("failed to create .agentic directory: %w", err)
	}

	if discover {
		fmt.Println("Discovering packages...")
		if err := discoverManifest(); err != nil {
			return fmt.Errorf("discovery failed: %w", err)
		}
		fmt.Println("Generated GRAPH.manifest")
	}

	// Check for existing GRAPH.manifest
	if _, err := os.Stat("GRAPH.manifest"); err == nil {
		fmt.Println("Found existing GRAPH.manifest, validating...")
		g, err := graph.Load("GRAPH.manifest")
		if err != nil {
			return fmt.Errorf("invalid GRAPH.manifest: %w", err)
		}
		fmt.Printf("Graph loaded: %d nodes\n", len(g.Nodes))
	} else if !discover {
		fmt.Println("No GRAPH.manifest found. Create one to define your project structure.")
	}

	fmt.Println("Initialization complete.")
	return nil
}

// discoveredNode holds information about a discovered package
type discoveredNode struct {
	id         string
	path       string
	deps       []string
	tokenCount int
}

// discoverManifest scans internal/ and nodes/ directories for Go packages
// and generates a GRAPH.manifest file
func discoverManifest() error {
	nodes := make(map[string]*discoveredNode)
	var order []string

	// Scan internal/ directory
	internalNodes, err := scanDirectory("internal")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to scan internal/: %w", err)
	}
	for _, n := range internalNodes {
		nodes[n.id] = n
		order = append(order, n.id)
	}

	// Scan nodes/ directory
	nodesNodes, err := scanDirectory("nodes")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to scan nodes/: %w", err)
	}
	for _, n := range nodesNodes {
		nodes[n.id] = n
		order = append(order, n.id)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no packages found in internal/ or nodes/")
	}

	// Filter dependencies to only include discovered nodes
	for _, n := range nodes {
		var validDeps []string
		for _, dep := range n.deps {
			if _, exists := nodes[dep]; exists {
				validDeps = append(validDeps, dep)
			}
		}
		n.deps = validDeps
	}

	// Sort order for deterministic output
	sort.Strings(order)

	// Topologically sort nodes so dependencies come first
	order, err = topologicalSortDiscovered(nodes, order)
	if err != nil {
		return fmt.Errorf("failed to sort nodes: %w", err)
	}

	// Write GRAPH.manifest
	return writeManifest(nodes, order)
}

// scanDirectory scans a directory for Go packages
func scanDirectory(dir string) ([]*discoveredNode, error) {
	var nodes []*discoveredNode

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgPath := filepath.Join(dir, entry.Name())

		// Check if this is a Go package (has .go files) or a node (has SRC/ dir)
		var goFiles []string
		var isNode bool

		srcDir := filepath.Join(pkgPath, "SRC")
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			// This is a node with SRC/ directory
			isNode = true
			goFiles, _ = filepath.Glob(filepath.Join(srcDir, "*.go"))
		} else {
			// This is a regular package
			goFiles, _ = filepath.Glob(filepath.Join(pkgPath, "*.go"))
		}

		if len(goFiles) == 0 {
			continue
		}

		// Calculate token budget based on file sizes
		tokenCount := estimateTokenBudget(goFiles)

		// Discover dependencies
		var deps []string
		if isNode {
			// Use graph.DiscoverDeps for nodes with SRC/
			deps, err = graph.DiscoverDeps(pkgPath)
			if err != nil {
				// Fall back to simple scanning
				deps = scanImports(goFiles)
			}
		} else {
			deps = scanImports(goFiles)
		}

		node := &discoveredNode{
			id:         entry.Name(),
			path:       pkgPath,
			deps:       deps,
			tokenCount: tokenCount,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// estimateTokenBudget calculates token count based on file sizes (4 chars per token)
func estimateTokenBudget(files []string) int {
	var totalSize int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}
	// 4 characters per token, round up
	return int((totalSize + 3) / 4)
}

// scanImports scans Go files for internal package imports
func scanImports(files []string) []string {
	importRegex := regexp.MustCompile(`"github\.com/aid/agentic/internal/([^"/]+)`)
	nodeImportRegex := regexp.MustCompile(`"github\.com/aid/agentic/nodes/([^"/]+)`)

	seen := make(map[string]bool)
	var deps []string

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Find internal/ imports
		matches := importRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			pkgName := match[1]
			if !seen[pkgName] {
				seen[pkgName] = true
				deps = append(deps, pkgName)
			}
		}

		// Find nodes/ imports
		matches = nodeImportRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			pkgName := match[1]
			if !seen[pkgName] {
				seen[pkgName] = true
				deps = append(deps, pkgName)
			}
		}
	}

	return deps
}

// topologicalSortDiscovered sorts discovered nodes so dependencies come first
func topologicalSortDiscovered(nodes map[string]*discoveredNode, initialOrder []string) ([]string, error) {
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	var sorted []string

	var visit func(id string) error
	visit = func(id string) error {
		if temp[id] {
			return fmt.Errorf("cycle detected at node: %s", id)
		}
		if visited[id] {
			return nil
		}

		node, exists := nodes[id]
		if !exists {
			return nil
		}

		temp[id] = true

		for _, dep := range node.deps {
			if err := visit(dep); err != nil {
				return err
			}
		}

		temp[id] = false
		visited[id] = true
		sorted = append(sorted, id)
		return nil
	}

	for _, id := range initialOrder {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// writeManifest writes the GRAPH.manifest file
func writeManifest(nodes map[string]*discoveredNode, order []string) error {
	file, err := os.Create("GRAPH.manifest")
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, "# Agentic Graph Manifest - auto-generated by discover")
	fmt.Fprintln(file, "# Format: TYPE:ID path=PATH deps=[DEPS] toks=TOKEN_CAP ver=VERSION")
	fmt.Fprintln(file, "")

	for _, id := range order {
		node := nodes[id]

		deps := "[]"
		if len(node.deps) > 0 {
			deps = "[" + strings.Join(node.deps, ",") + "]"
		}

		// Determine padding for alignment
		line := fmt.Sprintf("L:%-10s path=%-20s deps=%-40s toks=%-6d ver=1",
			id, node.path, deps, node.tokenCount)
		fmt.Fprintln(file, line)
	}

	return nil
}

func runGraph() error {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return err
	}

	fmt.Println("Dependency Graph:")
	fmt.Println("=================")
	g.Print()
	return nil
}

func runStatus() error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}
	ws.PrintStatus()
	return nil
}

func runTask(request, node string, parallel bool) error {
	fmt.Printf("Running task: %s\n", request)
	if node != "" {
		fmt.Printf("Target node: %s\n", node)
	}

	// Load graph
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return err
	}

	// Find target node(s)
	var targetNodes []*graph.Node
	if node != "" {
		n := g.GetNode(node)
		if n == nil {
			return fmt.Errorf("node not found: %s", node)
		}
		targetNodes = []*graph.Node{n}
	} else {
		// Use all leaf nodes if no specific node
		targetNodes = g.GetLeafNodes()
	}

	// Use parallel mode if requested and multiple nodes
	if parallel && len(targetNodes) > 1 {
		fmt.Printf("\nRunning in parallel mode on %d nodes...\n", len(targetNodes))
		ws, err := workspace.Load()
		if err != nil {
			return err
		}
		return runTasksParallel(g, targetNodes, request, ws)
	}

	// Build bundle and call brain for each node
	for _, n := range targetNodes {
		fmt.Printf("\nProcessing node: %s\n", n.ID)

		b, err := bundle.Build(n)
		if err != nil {
			return fmt.Errorf("failed to build bundle for %s: %w", n.ID, err)
		}

		// Show bundle details in verbose mode
		verboseBundle(b)

		// Check token budget
		tokens := b.EstimateTokens()
		if n.Meta != nil && n.Meta.Budgets.TokenCap > 0 && tokens > n.Meta.Budgets.TokenCap {
			return fmt.Errorf("node %s exceeds token budget: %d > %d", n.ID, tokens, n.Meta.Budgets.TokenCap)
		}

		// Show prompt in verbose mode
		prompt := brain.BuildPrompt(request, b)
		verbosePrompt(prompt)
		verboseLog("Calling Claude with ~%d tokens...", token.EstimateString(prompt))

		// Call brain
		response, err := brain.Call(request, b)
		if err != nil {
			return fmt.Errorf("brain call failed for %s: %w", n.ID, err)
		}

		verboseLog("Response received: %d files", len(response.Files))

		// Convert brain files to workspace files
		var files []workspace.FileChange
		for _, f := range response.Files {
			files = append(files, workspace.FileChange{
				Path:    f.Path,
				Content: f.Content,
			})
		}

		if len(files) == 0 {
			fmt.Printf("No file changes for node: %s\n", n.ID)
			if response.Message != "" {
				fmt.Printf("Message: %s\n", response.Message)
			}
			continue
		}

		// Stage the files
		ws, err := workspace.Load()
		if err != nil {
			return err
		}
		ws.StageFiles(n.ID, files, response.Message)
		if err := ws.Save(); err != nil {
			return err
		}

		fmt.Printf("Changes staged for node: %s (%d files)\n", n.ID, len(files))
	}

	fmt.Println("\nUse 'agentic diff' to review changes, 'agentic apply' to apply them.")
	return nil
}

// groupByDependencyLevel groups nodes by their dependency level
// Level 0 = nodes with no dependencies, Level 1 = nodes that depend on level 0, etc.
func groupByDependencyLevel(nodes []*graph.Node) [][]*graph.Node {
	nodeSet := make(map[string]bool)
	for _, n := range nodes {
		nodeSet[n.ID] = true
	}

	levels := make([][]*graph.Node, 0)
	processed := make(map[string]bool)

	for len(processed) < len(nodes) {
		var level []*graph.Node
		for _, n := range nodes {
			if processed[n.ID] {
				continue
			}
			// Check all deps are either processed or not in our node set
			ready := true
			for _, depID := range n.Dependencies {
				if nodeSet[depID] && !processed[depID] {
					ready = false
					break
				}
			}
			if ready {
				level = append(level, n)
			}
		}
		if len(level) == 0 {
			break // Prevent infinite loop on cycles
		}
		for _, n := range level {
			processed[n.ID] = true
		}
		levels = append(levels, level)
	}
	return levels
}

// runTasksParallel processes nodes in parallel, respecting dependency order
func runTasksParallel(g *graph.Graph, nodes []*graph.Node, request string, ws *workspace.Workspace) error {
	levels := groupByDependencyLevel(nodes)

	for levelNum, level := range levels {
		verboseLog("Processing level %d: %d nodes", levelNum, len(level))

		var wg sync.WaitGroup
		errChan := make(chan error, len(level))

		for _, node := range level {
			wg.Add(1)
			go func(n *graph.Node) {
				defer wg.Done()
				if err := runSingleNodeTask(n, request, ws); err != nil {
					errChan <- fmt.Errorf("%s: %w", n.ID, err)
				}
			}(node)
		}

		wg.Wait()
		close(errChan)

		var errs []error
		for err := range errChan {
			errs = append(errs, err)
		}
		if len(errs) > 0 {
			return fmt.Errorf("parallel execution failed: %v", errs)
		}

		fmt.Printf("Level %d complete: %d nodes processed\n", levelNum, len(level))
	}

	fmt.Println("\nUse 'agentic diff' to review changes, 'agentic apply' to apply them.")
	return nil
}

// runSingleNodeTask processes a single node (used by parallel execution)
func runSingleNodeTask(n *graph.Node, request string, ws *workspace.Workspace) error {
	b, err := bundle.Build(n)
	if err != nil {
		return fmt.Errorf("failed to build bundle: %w", err)
	}

	// Check token budget
	tokens := b.EstimateTokens()
	if n.Meta != nil && n.Meta.Budgets.TokenCap > 0 && tokens > n.Meta.Budgets.TokenCap {
		return fmt.Errorf("exceeds token budget: %d > %d", tokens, n.Meta.Budgets.TokenCap)
	}

	response, err := brain.Call(request, b)
	if err != nil {
		return fmt.Errorf("brain call failed: %w", err)
	}

	if len(response.Files) == 0 {
		verboseLog("Node %s: no file changes", n.ID)
		return nil
	}

	var files []workspace.FileChange
	for _, f := range response.Files {
		files = append(files, workspace.FileChange{
			Path:    f.Path,
			Content: f.Content,
		})
	}

	ws.StageFiles(n.ID, files, response.Message)
	if err := ws.Save(); err != nil {
		return err
	}

	fmt.Printf("Changes staged for node: %s (%d files)\n", n.ID, len(files))
	return nil
}

func runPlan(request string) error {
	fmt.Printf("Planning: %s\n", request)

	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return err
	}

	// For now, just show affected nodes based on simple heuristics
	fmt.Println("\nExecution plan:")
	sorted, err := g.TopologicalSort()
	if err != nil {
		return err
	}

	for i, n := range sorted {
		nodeType := "leaf"
		if n.Type == graph.NodeTypeComposite {
			nodeType = "composite"
		}
		fmt.Printf("  %d. [%s] %s\n", i+1, nodeType, n.ID)
	}

	return nil
}

func runDiff() error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	ws.PrintDiff()
	return nil
}

func runApply(skipConfirm bool, skipChecks bool) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	if len(ws.StagedChanges) == 0 {
		fmt.Println("No staged changes to apply.")
		return nil
	}

	// Save node IDs before apply clears them
	var affectedNodeIDs []string
	for nodeID := range ws.StagedChanges {
		affectedNodeIDs = append(affectedNodeIDs, nodeID)
	}

	// Count total files
	totalFiles := 0
	for _, changes := range ws.StagedChanges {
		totalFiles += len(changes.Files)
	}

	if !skipConfirm {
		fmt.Printf("Apply %d file(s) across %d node(s)? [y/N] ", totalFiles, len(ws.StagedChanges))
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := ws.ApplyChanges(); err != nil {
		return err
	}

	fmt.Println("Changes applied successfully.")

	// Run checks for affected nodes unless skipped
	if !skipChecks {
		g, err := graph.Load("GRAPH.manifest")
		if err != nil {
			return fmt.Errorf("failed to load graph for checks: %w", err)
		}

		for _, nodeID := range affectedNodeIDs {
			node := g.GetNode(nodeID)
			if node == nil {
				continue
			}
			if err := workspace.RunChecks(node); err != nil {
				return fmt.Errorf("checks failed for node %s: %w", nodeID, err)
			}
		}

		fmt.Println("All checks passed!")
	}

	return nil
}

func runRollback() error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	if err := ws.Rollback(); err != nil {
		return err
	}

	fmt.Println("Rollback complete.")
	return nil
}

func runUndo() error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	if err := ws.Undo(); err != nil {
		return err
	}

	fmt.Println("Undo complete.")
	return nil
}

func runEnter(nodeID string) error {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return err
	}

	n := g.GetNode(nodeID)
	if n == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	ws.CurrentNode = nodeID
	if err := ws.Save(); err != nil {
		return err
	}

	fmt.Printf("Entered node: %s\n", nodeID)
	if n.Type == graph.NodeTypeComposite {
		fmt.Printf("This is a composite node. Loading subgraph from: %s\n", n.Path)
	}

	return nil
}

func runSplit(nodeID string) error {
	fmt.Printf("Split protocol for node: %s\n", nodeID)

	// Load graph
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		return err
	}

	// Get the node
	node := g.GetNode(nodeID)
	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// Build bundle
	b, err := bundle.Build(node)
	if err != nil {
		return fmt.Errorf("failed to build bundle: %w", err)
	}

	// Check token budget
	totalTokens := b.EstimateTokens()
	tokenCap := 0
	if node.Meta != nil && node.Meta.Budgets.TokenCap > 0 {
		tokenCap = node.Meta.Budgets.TokenCap
	}

	fmt.Printf("\nBundle tokens: %d\n", totalTokens)
	if tokenCap > 0 {
		fmt.Printf("Token budget:  %d\n", tokenCap)
	}

	// If over budget, list files with token counts
	if tokenCap > 0 && totalTokens > tokenCap {
		fmt.Printf("\nNode exceeds token budget by %d tokens.\n", totalTokens-tokenCap)
		fmt.Println("\nFiles by token count:")

		// Build list of files with token counts
		type fileTokens struct {
			path   string
			tokens int
		}
		var files []fileTokens
		for path, content := range b.Files {
			tokens := token.EstimateString(content)
			files = append(files, fileTokens{path: path, tokens: tokens})
		}

		// Sort by token count descending
		sort.Slice(files, func(i, j int) bool {
			return files[i].tokens > files[j].tokens
		})

		for _, f := range files {
			fmt.Printf("  %6d  %s\n", f.tokens, f.path)
		}

		fmt.Println("\nConsider splitting this node into smaller sub-nodes.")
	} else {
		fmt.Println("\nNode is within token budget. No split required.")
	}

	return nil
}

func runInteractive() {
	fmt.Println("Agentic Interactive Mode")
	fmt.Println("========================")
	fmt.Println("Commands: graph, enter <node>, request <text>, plan, run, diff, apply, rollback, undo, status, quit")
	fmt.Println()

	StartREPL()
}