package policy

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aid/agentic/internal/bundle"
	"github.com/aid/agentic/internal/graph"
	"github.com/aid/agentic/internal/token"
)

// Result represents the result of policy evaluation
type Result struct {
	Passed     bool
	Violations []Violation
}

// Violation describes a policy violation
type Violation struct {
	Policy   string
	Severity string // "error" or "warning"
	Message  string
}

// ContractHashes stores the contract hashes for all nodes
type ContractHashes struct {
	Hashes map[string]string `json:"hashes"`
}

// contractHashesPath is the default path for storing contract hashes
const contractHashesPath = ".agentic/contracts.json"

// exportedSymbolPattern matches exported Go symbols (uppercase after keyword)
var exportedSymbolPattern = regexp.MustCompile(`^(func|type|var|const)\s+(\(?[A-Z][^\s(]*)`)

// Evaluate checks all policies for a node and response
func Evaluate(node *graph.Node, b *bundle.Bundle, diff string) *Result {
	result := &Result{Passed: true}

	// Check token budget
	if node.Meta != nil && node.Meta.Budgets.TokenCap > 0 {
		tokens := b.EstimateTokens()
		if tokens > node.Meta.Budgets.TokenCap {
			result.Passed = false
			result.Violations = append(result.Violations, Violation{
				Policy:   "token_budget",
				Severity: "error",
				Message:  fmt.Sprintf("Token count %d exceeds budget %d", tokens, node.Meta.Budgets.TokenCap),
			})
		}
	}

	// Check diff scope
	if node.Meta != nil && len(node.Meta.Policies.AllowedPaths) > 0 {
		violations := checkDiffScope(diff, node.Meta.Policies.AllowedPaths, node.Path)
		for _, v := range violations {
			result.Passed = false
			result.Violations = append(result.Violations, v)
		}
	}

	// Check for contract changes
	contractViolations := checkContractChanges(diff, node)
	for _, v := range contractViolations {
		result.Violations = append(result.Violations, v)
		if v.Severity == "error" {
			result.Passed = false
		}
	}

	return result
}

// checkDiffScope verifies that all changed files are in allowed paths
func checkDiffScope(diff string, allowedPaths []string, nodePath string) []Violation {
	var violations []Violation

	// Extract file paths from diff
	changedFiles := ExtractFilePaths(diff)

	for _, file := range changedFiles {
		// Normalize path relative to node
		relPath := file
		if strings.HasPrefix(file, nodePath+"/") {
			relPath = strings.TrimPrefix(file, nodePath+"/")
		}

		allowed := false
		for _, allowedPath := range allowedPaths {
			// Check if file matches allowed path pattern
			if MatchPath(relPath, allowedPath) {
				allowed = true
				break
			}
		}

		if !allowed {
			violations = append(violations, Violation{
				Policy:   "diff_scope",
				Severity: "error",
				Message:  fmt.Sprintf("File %s is outside allowed paths: %v", file, allowedPaths),
			})
		}
	}

	return violations
}

// ExtractFilePaths extracts file paths from a unified diff
func ExtractFilePaths(diff string) []string {
	var paths []string
	seen := make(map[string]bool)

	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+++ ") {
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "b/")
			if path != "/dev/null" && !seen[path] {
				paths = append(paths, path)
				seen[path] = true
			}
		}
	}

	return paths
}

// MatchPath checks if a file path matches an allowed path pattern
func MatchPath(file, pattern string) bool {
	// Handle directory patterns (e.g., "SRC/")
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(file, pattern) || strings.HasPrefix(file, strings.TrimSuffix(pattern, "/"))
	}

	// Try glob matching
	matched, _ := filepath.Match(pattern, file)
	if matched {
		return true
	}

	// Try prefix matching
	return strings.HasPrefix(file, pattern)
}

// checkContractChanges detects changes to public contracts
func checkContractChanges(diff string, node *graph.Node) []Violation {
	var violations []Violation

	if node.Meta == nil || len(node.Meta.PublicContract) == 0 {
		return violations
	}

	changedFiles := ExtractFilePaths(diff)

	for _, file := range changedFiles {
		for _, contract := range node.Meta.PublicContract {
			if MatchPath(file, contract) {
				violations = append(violations, Violation{
					Policy:   "contract_change",
					Severity: "warning",
					Message:  fmt.Sprintf("Public contract file modified: %s. This may require updating dependents.", file),
				})
			}
		}
	}

	return violations
}

// ShouldSplit determines if a node should be split based on token count
func ShouldSplit(node *graph.Node, b *bundle.Bundle) bool {
	budget := token.GetBudget("default")

	// If node has a specific budget, use that
	if node.Meta != nil && node.Meta.Budgets.TokenCap > 0 {
		budget.Available = node.Meta.Budgets.TokenCap
	}

	tokens := b.EstimateTokens()
	return tokens > budget.Available
}

// SuggestSplit suggests how to split a node that exceeds budget
func SuggestSplit(node *graph.Node, b *bundle.Bundle) []string {
	var suggestions []string

	// Group files by directory
	dirs := make(map[string]int) // dir -> token count
	for path, content := range b.Files {
		dir := filepath.Dir(path)
		if dir == "." {
			dir = "root"
		}
		dirs[dir] += token.EstimateString(content)
	}

	// Suggest splitting large directories into separate nodes
	for dir, tokens := range dirs {
		if tokens > 10000 { // Significant size
			suggestions = append(suggestions, fmt.Sprintf("Split %s/ into separate node (~%d tokens)", dir, tokens))
		}
	}

	// If no directory-based splits, suggest file-based splits
	if len(suggestions) == 0 {
		fileCount := len(b.Files)
		if fileCount > 10 {
			half := fileCount / 2
			suggestions = append(suggestions, fmt.Sprintf("Split into two nodes with ~%d files each", half))
		}
	}

	return suggestions
}

// HashContracts scans Go files in nodePath for exported symbols and returns
// a sha256 hash of the sorted signatures. Exported symbols are identifiers
// starting with uppercase letters after 'func ', 'type ', 'var ', 'const '.
func HashContracts(nodePath string) (string, error) {
	var signatures []string

	err := filepath.Walk(nodePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSignatures, err := extractExportedSignatures(path)
		if err != nil {
			return fmt.Errorf("extracting signatures from %s: %w", path, err)
		}

		signatures = append(signatures, fileSignatures...)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walking path %s: %w", nodePath, err)
	}

	// Sort signatures for deterministic hashing
	sort.Strings(signatures)

	// Compute sha256 hash
	combined := strings.Join(signatures, "\n")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

// extractExportedSignatures extracts exported symbol signatures from a Go file
func extractExportedSignatures(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var signatures []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for exported symbols
		matches := exportedSymbolPattern.FindStringSubmatch(line)
		if len(matches) >= 3 {
			keyword := matches[1]
			symbol := matches[2]

			// Clean up method receiver notation
			symbol = strings.TrimPrefix(symbol, "(")

			// Build signature
			signature := fmt.Sprintf("%s %s", keyword, symbol)

			// For func, include the full line to capture parameters
			if keyword == "func" {
				signature = extractFuncSignature(line)
			}

			signatures = append(signatures, signature)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return signatures, nil
}

// extractFuncSignature extracts a clean function signature from a func declaration line
func extractFuncSignature(line string) string {
	// Remove leading/trailing whitespace
	line = strings.TrimSpace(line)

	// Remove function body opening brace if present
	if idx := strings.Index(line, "{"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}

	return line
}

// LoadContractHashes loads contract hashes from .agentic/contracts.json
func LoadContractHashes() (*ContractHashes, error) {
	data, err := os.ReadFile(contractHashesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ContractHashes{Hashes: make(map[string]string)}, nil
		}
		return nil, fmt.Errorf("reading contract hashes: %w", err)
	}

	var hashes ContractHashes
	if err := json.Unmarshal(data, &hashes); err != nil {
		return nil, fmt.Errorf("parsing contract hashes: %w", err)
	}

	if hashes.Hashes == nil {
		hashes.Hashes = make(map[string]string)
	}

	return &hashes, nil
}

// SaveContractHashes saves contract hashes to .agentic/contracts.json
func SaveContractHashes(hashes *ContractHashes) error {
	// Ensure directory exists
	dir := filepath.Dir(contractHashesPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling contract hashes: %w", err)
	}

	if err := os.WriteFile(contractHashesPath, data, 0644); err != nil {
		return fmt.Errorf("writing contract hashes: %w", err)
	}

	return nil
}

// HasContractChanged compares the new hash against the stored hash for a node.
// Returns true if the contract has changed (hashes differ or no previous hash exists).
func HasContractChanged(nodeID, newHash string) bool {
	hashes, err := LoadContractHashes()
	if err != nil {
		// If we can't load hashes, assume changed to be safe
		return true
	}

	storedHash, exists := hashes.Hashes[nodeID]
	if !exists {
		// No previous hash means this is new, consider it changed
		return true
	}

	return storedHash != newHash
}