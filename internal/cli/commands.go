package cli

import (
	"fmt"
	"os"
	"sort"

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
)

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
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
		return runInit()
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
		return runTask(args[0], node)
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
		return runApply(yes)
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
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(runTaskCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(enterCmd)
	rootCmd.AddCommand(splitCmd)

	runTaskCmd.Flags().StringP("node", "n", "", "target node for the task")
	applyCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt")
}

func runInit() error {
	fmt.Println("Initializing agentic...")

	// Create .agentic directory
	if err := os.MkdirAll(".agentic", 0755); err != nil {
		return fmt.Errorf("failed to create .agentic directory: %w", err)
	}

	// Check for existing GRAPH.manifest
	if _, err := os.Stat("GRAPH.manifest"); err == nil {
		fmt.Println("Found existing GRAPH.manifest, validating...")
		g, err := graph.Load("GRAPH.manifest")
		if err != nil {
			return fmt.Errorf("invalid GRAPH.manifest: %w", err)
		}
		fmt.Printf("Graph loaded: %d nodes\n", len(g.Nodes))
	} else {
		fmt.Println("No GRAPH.manifest found. Create one to define your project structure.")
	}

	fmt.Println("Initialization complete.")
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

func runTask(request, node string) error {
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

	// Build bundle and call brain for each node
	for _, n := range targetNodes {
		fmt.Printf("\nProcessing node: %s\n", n.ID)

		b, err := bundle.Build(n)
		if err != nil {
			return fmt.Errorf("failed to build bundle for %s: %w", n.ID, err)
		}

		// Check token budget
		tokens := b.EstimateTokens()
		if n.Meta != nil && n.Meta.Budgets.TokenCap > 0 && tokens > n.Meta.Budgets.TokenCap {
			return fmt.Errorf("node %s exceeds token budget: %d > %d", n.ID, tokens, n.Meta.Budgets.TokenCap)
		}

		// Call brain
		response, err := brain.Call(request, b)
		if err != nil {
			return fmt.Errorf("brain call failed for %s: %w", n.ID, err)
		}

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

func runApply(skipConfirm bool) error {
	ws, err := workspace.Load()
	if err != nil {
		return err
	}

	if len(ws.StagedChanges) == 0 {
		fmt.Println("No staged changes to apply.")
		return nil
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
	fmt.Println("Commands: graph, enter <node>, request <text>, plan, run, diff, apply, rollback, status, quit")
	fmt.Println()

	StartREPL()
}