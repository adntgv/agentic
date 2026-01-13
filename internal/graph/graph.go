package graph

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// NodeType represents whether a node is a leaf or composite
type NodeType string

const (
	NodeTypeLeaf      NodeType = "L"
	NodeTypeComposite NodeType = "C"
)

// Node represents a single node in the graph
type Node struct {
	ID           string
	Type         NodeType
	Path         string
	Dependencies []string
	Tokens       int
	Version      int
	ContractHash string // for leaf nodes
	BundleHash   string // for leaf nodes
	ManifestHash string // for composite nodes

	// Resolved references
	Meta       *NodeMeta
	Children   []*Node // resolved dependency references
	Dependents []*Node // reverse dependencies
}

// NodeMeta represents NODE.meta.yaml contents
type NodeMeta struct {
	ID         string   `yaml:"id"`
	Type       string   `yaml:"type"`
	Purpose    string   `yaml:"purpose"`
	Invariants []string `yaml:"invariants"`
	NonGoals   []string `yaml:"non_goals,omitempty"`
	Budgets    struct {
		TokenCap int `yaml:"token_cap"`
	} `yaml:"budgets"`
	Policies struct {
		AllowedPaths []string `yaml:"allowed_paths"`
		Checks       []string `yaml:"checks"`
	} `yaml:"policies"`
	PublicContract []string `yaml:"public_contract,omitempty"`
}

// Graph represents the full dependency graph
type Graph struct {
	Nodes    map[string]*Node
	Order    []string // original order from manifest
	RootPath string
}

// Load loads a GRAPH.manifest file
func Load(path string) (*Graph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	g := &Graph{
		Nodes:    make(map[string]*Node),
		RootPath: filepath.Dir(path),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		node, err := parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		g.Nodes[node.ID] = node
		g.Order = append(g.Order, node.ID)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading manifest: %w", err)
	}

	// Resolve dependencies
	if err := g.resolveDependencies(); err != nil {
		return nil, err
	}

	// Load node metadata
	if err := g.loadNodeMeta(); err != nil {
		return nil, err
	}

	// Validate graph
	if err := g.validate(); err != nil {
		return nil, err
	}

	return g, nil
}

// parseLine parses a single line from GRAPH.manifest
// Format: L:id path=... deps=[...] toks=... ver=...
func parseLine(line string) (*Node, error) {
	node := &Node{}

	// Parse type and ID
	typeIDRegex := regexp.MustCompile(`^([LC]):(\S+)`)
	match := typeIDRegex.FindStringSubmatch(line)
	if match == nil {
		return nil, fmt.Errorf("invalid format: expected L:id or C:id")
	}

	node.Type = NodeType(match[1])
	node.ID = match[2]

	// Parse key=value pairs
	kvRegex := regexp.MustCompile(`(\w+)=(\[[^\]]*\]|\S+)`)
	matches := kvRegex.FindAllStringSubmatch(line, -1)

	for _, m := range matches {
		key := m[1]
		value := m[2]

		switch key {
		case "path":
			node.Path = value
		case "deps":
			node.Dependencies = parseList(value)
		case "toks":
			if n, err := strconv.Atoi(value); err == nil {
				node.Tokens = n
			}
		case "ver":
			if n, err := strconv.Atoi(value); err == nil {
				node.Version = n
			}
		case "contract":
			node.ContractHash = value
		case "bundle":
			node.BundleHash = value
		case "manifest":
			node.ManifestHash = value
		}
	}

	if node.Path == "" {
		return nil, fmt.Errorf("missing path for node %s", node.ID)
	}

	return node, nil
}

// parseList parses a bracket-enclosed list: [a,b,c] or []
func parseList(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// resolveDependencies resolves string references to actual node pointers
func (g *Graph) resolveDependencies() error {
	for _, node := range g.Nodes {
		for _, depID := range node.Dependencies {
			dep, ok := g.Nodes[depID]
			if !ok {
				return fmt.Errorf("node %s: unknown dependency %s", node.ID, depID)
			}
			node.Children = append(node.Children, dep)
			dep.Dependents = append(dep.Dependents, node)
		}
	}
	return nil
}

// loadNodeMeta loads NODE.meta.yaml for each node
func (g *Graph) loadNodeMeta() error {
	for _, node := range g.Nodes {
		metaPath := filepath.Join(g.RootPath, node.Path, "NODE.meta.yaml")
		if _, err := os.Stat(metaPath); os.IsNotExist(err) {
			continue // metadata is optional
		}

		data, err := os.ReadFile(metaPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", metaPath, err)
		}

		var meta NodeMeta
		if err := yaml.Unmarshal(data, &meta); err != nil {
			return fmt.Errorf("failed to parse %s: %w", metaPath, err)
		}

		node.Meta = &meta
	}
	return nil
}

// validate checks for cycles and other issues
func (g *Graph) validate() error {
	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(nodeID string) bool
	hasCycle = func(nodeID string) bool {
		visited[nodeID] = true
		recStack[nodeID] = true

		node := g.Nodes[nodeID]
		for _, dep := range node.Children {
			if !visited[dep.ID] {
				if hasCycle(dep.ID) {
					return true
				}
			} else if recStack[dep.ID] {
				return true
			}
		}

		recStack[nodeID] = false
		return false
	}

	for id := range g.Nodes {
		if !visited[id] {
			if hasCycle(id) {
				return fmt.Errorf("cycle detected involving node: %s", id)
			}
		}
	}

	return nil
}

// TopologicalSort returns nodes in dependency order (dependencies first)
func (g *Graph) TopologicalSort() ([]*Node, error) {
	var sorted []*Node
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(node *Node) error
	visit = func(node *Node) error {
		if temp[node.ID] {
			return fmt.Errorf("cycle detected at node: %s", node.ID)
		}
		if visited[node.ID] {
			return nil
		}

		temp[node.ID] = true

		for _, dep := range node.Children {
			if err := visit(dep); err != nil {
				return err
			}
		}

		temp[node.ID] = false
		visited[node.ID] = true
		sorted = append(sorted, node)

		return nil
	}

	// Process nodes in original order for determinism
	for _, id := range g.Order {
		node := g.Nodes[id]
		if err := visit(node); err != nil {
			return nil, err
		}
	}

	return sorted, nil
}

// GetNode returns a node by ID
func (g *Graph) GetNode(id string) *Node {
	return g.Nodes[id]
}

// GetLeafNodes returns all leaf nodes
func (g *Graph) GetLeafNodes() []*Node {
	var leaves []*Node
	for _, id := range g.Order {
		node := g.Nodes[id]
		if node.Type == NodeTypeLeaf {
			leaves = append(leaves, node)
		}
	}
	return leaves
}

// GetReverseDeps returns all nodes that depend on the given node
func (g *Graph) GetReverseDeps(nodeID string) []*Node {
	node := g.Nodes[nodeID]
	if node == nil {
		return nil
	}

	visited := make(map[string]bool)
	var result []*Node

	var collect func(n *Node)
	collect = func(n *Node) {
		for _, dep := range n.Dependents {
			if !visited[dep.ID] {
				visited[dep.ID] = true
				result = append(result, dep)
				collect(dep)
			}
		}
	}

	collect(node)
	return result
}

// Print displays the graph structure
func (g *Graph) Print() {
	// Find roots (nodes with no dependents)
	roots := make([]*Node, 0)
	for _, id := range g.Order {
		node := g.Nodes[id]
		if len(node.Dependents) == 0 {
			roots = append(roots, node)
		}
	}

	if len(roots) == 0 {
		// If no roots, just print all nodes
		for _, id := range g.Order {
			g.printNode(g.Nodes[id], "", true)
		}
		return
	}

	for i, root := range roots {
		g.printNode(root, "", i == len(roots)-1)
	}
}

func (g *Graph) printNode(node *Node, prefix string, isLast bool) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	nodeType := "L"
	if node.Type == NodeTypeComposite {
		nodeType = "C"
	}

	fmt.Printf("%s%s[%s] %s (%d toks)\n", prefix, connector, nodeType, node.ID, node.Tokens)

	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	// Print dependencies
	for i, dep := range node.Children {
		g.printNode(dep, newPrefix, i == len(node.Children)-1)
	}
}

// Save writes the graph back to a manifest file
func (g *Graph) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, id := range g.Order {
		node := g.Nodes[id]
		line := formatNode(node)
		fmt.Fprintln(file, line)
	}

	return nil
}

func formatNode(n *Node) string {
	parts := []string{fmt.Sprintf("%s:%s", n.Type, n.ID)}

	parts = append(parts, fmt.Sprintf("path=%s", n.Path))

	deps := "[]"
	if len(n.Dependencies) > 0 {
		deps = "[" + strings.Join(n.Dependencies, ",") + "]"
	}
	parts = append(parts, fmt.Sprintf("deps=%s", deps))

	parts = append(parts, fmt.Sprintf("toks=%d", n.Tokens))

	if n.Type == NodeTypeLeaf {
		if n.ContractHash != "" {
			parts = append(parts, fmt.Sprintf("contract=%s", n.ContractHash))
		}
		if n.BundleHash != "" {
			parts = append(parts, fmt.Sprintf("bundle=%s", n.BundleHash))
		}
	} else {
		if n.ManifestHash != "" {
			parts = append(parts, fmt.Sprintf("manifest=%s", n.ManifestHash))
		}
	}

	parts = append(parts, fmt.Sprintf("ver=%d", n.Version))

	return strings.Join(parts, " ")
}

// DiscoverDeps scans Go files in a node's SRC/ directory and returns
// internal package dependencies (packages under github.com/aid/agentic/internal/*)
func DiscoverDeps(nodePath string) ([]string, error) {
	srcDir := filepath.Join(nodePath, "SRC")

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SRC directory: %w", err)
	}

	// Regex to match import statements and extract the path
	// Handles both single imports and grouped imports
	importRegex := regexp.MustCompile(`"(github\.com/aid/agentic/internal/([^"]+))"`)

	seen := make(map[string]bool)
	var deps []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(srcDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		matches := importRegex.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			// match[2] contains the package path after "github.com/aid/agentic/internal/"
			// Extract just the top-level package name (e.g., "graph" from "graph/subpkg")
			pkgPath := match[2]
			pkgName := strings.Split(pkgPath, "/")[0]

			if !seen[pkgName] {
				seen[pkgName] = true
				deps = append(deps, pkgName)
			}
		}
	}

	return deps, nil
}