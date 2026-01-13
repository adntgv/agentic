package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aid/agentic/internal/bundle"
	"github.com/aid/agentic/internal/graph"
	"github.com/aid/agentic/internal/workspace"
)

// StartREPL starts the interactive REPL
func StartREPL() {
	reader := bufio.NewReader(os.Stdin)
	var currentNode string

	for {
		// Build prompt
		prompt := "agentic"
		if currentNode != "" {
			prompt = fmt.Sprintf("agentic:%s", currentNode)
		}
		fmt.Printf("%s> ", prompt)

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// Parse command
		parts := strings.Fields(input)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "quit", "exit", "q":
			fmt.Println("Goodbye!")
			return

		case "help", "?":
			printREPLHelp()

		case "graph":
			if err := runGraph(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "status":
			if err := runStatus(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "enter":
			if len(args) == 0 {
				fmt.Println("Usage: enter <node>")
				continue
			}
			if err := runEnter(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				currentNode = args[0]
			}

		case "leave":
			currentNode = ""
			fmt.Println("Returned to root")

		case "request", "req":
			if len(args) == 0 {
				fmt.Println("Usage: request <text>")
				continue
			}
			request := strings.Join(args, " ")
			if err := runTask(request, currentNode); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "plan":
			if len(args) == 0 {
				fmt.Println("Usage: plan <text>")
				continue
			}
			request := strings.Join(args, " ")
			if err := runPlan(request); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "run":
			if len(args) == 0 {
				fmt.Println("Usage: run <text>")
				continue
			}
			request := strings.Join(args, " ")
			if err := runTask(request, currentNode); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "diff":
			if err := runDiff(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "apply":
			skipConfirm := len(args) > 0 && (args[0] == "-y" || args[0] == "--yes")
			if err := runApply(skipConfirm); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "rollback":
			if err := runRollback(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "split":
			if len(args) == 0 {
				if currentNode != "" {
					args = []string{currentNode}
				} else {
					fmt.Println("Usage: split <node>")
					continue
				}
			}
			if err := runSplit(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "nodes":
			listNodes()

		case "info":
			if len(args) == 0 {
				if currentNode != "" {
					showNodeInfo(currentNode)
				} else {
					fmt.Println("Usage: info <node>")
				}
				continue
			}
			showNodeInfo(args[0])

		case "bundle":
			if len(args) == 0 && currentNode == "" {
				fmt.Println("Usage: bundle <node>")
				continue
			}
			nodeID := currentNode
			if len(args) > 0 {
				nodeID = args[0]
			}
			showBundle(nodeID)

		case "check":
			if len(args) == 0 && currentNode == "" {
				fmt.Println("Usage: check <node>")
				continue
			}
			nodeID := currentNode
			if len(args) > 0 {
				nodeID = args[0]
			}
			runNodeChecks(nodeID)

		default:
			fmt.Printf("Unknown command: %s. Type 'help' for available commands.\n", cmd)
		}
	}
}

func printREPLHelp() {
	fmt.Println(`
Available commands:
  graph              Show dependency graph
  status             Show current status
  nodes              List all nodes
  info [node]        Show node details
  bundle [node]      Show bundle contents/stats

  enter <node>       Enter a node context
  leave              Return to root context

  request <text>     Send request to brain
  plan <text>        Generate execution plan
  run <text>         Execute request (alias for request)

  diff               Show staged changes
  apply [-y]         Apply staged changes
  rollback           Rollback last applied changes

  check [node]       Run checks for a node
  split [node]       Split node if too large

  help, ?            Show this help
  quit, exit, q      Exit REPL
`)
}

func listNodes() {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		fmt.Printf("Error loading graph: %v\n", err)
		return
	}

	fmt.Println("Nodes:")
	for _, id := range g.Order {
		node := g.Nodes[id]
		typeStr := "leaf"
		if node.Type == graph.NodeTypeComposite {
			typeStr = "composite"
		}
		fmt.Printf("  [%s] %s - %s\n", typeStr, node.ID, node.Path)
	}
}

func showNodeInfo(nodeID string) {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		fmt.Printf("Error loading graph: %v\n", err)
		return
	}

	node := g.GetNode(nodeID)
	if node == nil {
		fmt.Printf("Node not found: %s\n", nodeID)
		return
	}

	fmt.Printf("\nNode: %s\n", node.ID)
	fmt.Printf("Type: %s\n", node.Type)
	fmt.Printf("Path: %s\n", node.Path)
	fmt.Printf("Tokens: %d\n", node.Tokens)
	fmt.Printf("Version: %d\n", node.Version)

	if len(node.Dependencies) > 0 {
		fmt.Printf("Dependencies: %v\n", node.Dependencies)
	}

	if len(node.Dependents) > 0 {
		var depIDs []string
		for _, d := range node.Dependents {
			depIDs = append(depIDs, d.ID)
		}
		fmt.Printf("Dependents: %v\n", depIDs)
	}

	if node.Meta != nil {
		fmt.Printf("\nPurpose: %s\n", node.Meta.Purpose)

		if len(node.Meta.Invariants) > 0 {
			fmt.Println("Invariants:")
			for _, inv := range node.Meta.Invariants {
				fmt.Printf("  - %s\n", inv)
			}
		}

		if node.Meta.Budgets.TokenCap > 0 {
			fmt.Printf("Token Budget: %d\n", node.Meta.Budgets.TokenCap)
		}

		if len(node.Meta.Policies.AllowedPaths) > 0 {
			fmt.Printf("Allowed Paths: %v\n", node.Meta.Policies.AllowedPaths)
		}

		if len(node.Meta.Policies.Checks) > 0 {
			fmt.Printf("Checks: %v\n", node.Meta.Policies.Checks)
		}
	}
}

func showBundle(nodeID string) {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		fmt.Printf("Error loading graph: %v\n", err)
		return
	}

	node := g.GetNode(nodeID)
	if node == nil {
		fmt.Printf("Node not found: %s\n", nodeID)
		return
	}

	b, err := bundle.Build(node)
	if err != nil {
		fmt.Printf("Error building bundle: %v\n", err)
		return
	}

	tokens := b.EstimateTokens()

	fmt.Printf("\nBundle for node: %s\n", nodeID)
	fmt.Printf("Hash: %s\n", b.Hash)
	fmt.Printf("Files: %d\n", len(b.Files))
	fmt.Printf("Contracts: %d\n", len(b.Contracts))
	fmt.Printf("Total Size: %d bytes\n", b.TotalSize)
	fmt.Printf("Estimated Tokens: %d\n", tokens)

	fmt.Println("\nFiles:")
	for path := range b.Files {
		fmt.Printf("  %s\n", path)
	}

	if len(b.Contracts) > 0 {
		fmt.Println("\nContracts:")
		for path := range b.Contracts {
			fmt.Printf("  %s\n", path)
		}
	}
}

func runNodeChecks(nodeID string) {
	g, err := graph.Load("GRAPH.manifest")
	if err != nil {
		fmt.Printf("Error loading graph: %v\n", err)
		return
	}

	node := g.GetNode(nodeID)
	if node == nil {
		fmt.Printf("Node not found: %s\n", nodeID)
		return
	}

	if err := workspace.RunChecks(node); err != nil {
		fmt.Printf("Checks failed: %v\n", err)
	} else {
		fmt.Println("All checks passed!")
	}
}
