package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aid/agentic/internal/graph"
	"github.com/aid/agentic/internal/token"
)

// Bundle cache for avoiding redundant file reads
var bundleCache = struct {
	sync.RWMutex
	entries map[string]*cacheEntry
}{entries: make(map[string]*cacheEntry)}

type cacheEntry struct {
	bundle  *Bundle
	modTime time.Time
}

// Bundle represents the assembled context for a brain call
type Bundle struct {
	NodeID    string
	NodePath  string            // path to node from repo root
	Files     map[string]string // path -> content (relative to repo root)
	Meta      *graph.NodeMeta
	Contracts map[string]string // dependency contracts
	TotalSize int
	Hash      string
}

// getLatestModTime finds the most recent modification time in a directory
func getLatestModTime(dir string) time.Time {
	var latest time.Time
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
		return nil
	})
	return latest
}

// Build creates a bundle for a node
func Build(node *graph.Node) (*Bundle, error) {
	cacheKey := node.Path

	// Get absolute path to node and cwd for relative path calculation
	cwd, _ := os.Getwd()
	nodePath := node.Path
	if !filepath.IsAbs(nodePath) {
		nodePath = filepath.Join(cwd, nodePath)
	}

	// Check cache first
	modTime := getLatestModTime(nodePath)
	bundleCache.RLock()
	if entry, ok := bundleCache.entries[cacheKey]; ok {
		if !modTime.IsZero() && entry.modTime.Equal(modTime) {
			bundleCache.RUnlock()
			return entry.bundle, nil
		}
	}
	bundleCache.RUnlock()

	b := &Bundle{
		NodeID:    node.ID,
		NodePath:  node.Path,
		Files:     make(map[string]string),
		Meta:      node.Meta,
		Contracts: make(map[string]string),
	}

	// Collect files from node directory
	err := filepath.WalkDir(nodePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Skip common excluded directories
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == "__pycache__" || name == ".agentic" || name == "build" ||
				name == "dist" || name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get path relative to repo root (cwd), not node
		relPath, err := filepath.Rel(cwd, path)
		if err != nil {
			return err
		}

		// Skip binary and generated files
		ext := strings.ToLower(filepath.Ext(path))
		if isBinaryExt(ext) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		b.Files[relPath] = string(content)
		b.TotalSize += len(content)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	// Collect dependency contracts
	for _, dep := range node.Children {
		contractPath := filepath.Join(dep.Path, "CONTRACTS")
		if _, err := os.Stat(contractPath); err == nil {
			contracts, err := collectContracts(contractPath)
			if err != nil {
				return nil, fmt.Errorf("failed to collect contracts from %s: %w", dep.ID, err)
			}
			for k, v := range contracts {
				b.Contracts[dep.ID+"/"+k] = v
			}
		}
	}

	// Calculate bundle hash
	b.Hash = b.calculateHash()

	// Store in cache
	bundleCache.Lock()
	bundleCache.entries[cacheKey] = &cacheEntry{
		bundle:  b,
		modTime: modTime,
	}
	bundleCache.Unlock()

	return b, nil
}

// InvalidateCache removes a specific node from the cache
func InvalidateCache(nodePath string) {
	bundleCache.Lock()
	delete(bundleCache.entries, nodePath)
	bundleCache.Unlock()
}

// ClearCache removes all entries from the cache
func ClearCache() {
	bundleCache.Lock()
	bundleCache.entries = make(map[string]*cacheEntry)
	bundleCache.Unlock()
}

// CacheStats returns the number of cached entries and total size
func CacheStats() (int, int64) {
	bundleCache.RLock()
	defer bundleCache.RUnlock()
	count := len(bundleCache.entries)
	var size int64
	for _, e := range bundleCache.entries {
		size += int64(e.bundle.TotalSize)
	}
	return count, size
}

func isBinaryExt(ext string) bool {
	binary := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".o": true, ".a": true, ".lib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".wasm": true, ".pyc": true, ".class": true,
	}
	return binary[ext]
}

func collectContracts(contractPath string) (map[string]string, error) {
	contracts := make(map[string]string)

	err := filepath.WalkDir(contractPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(contractPath, path)
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		contracts[relPath] = string(content)
		return nil
	})

	return contracts, err
}

func (b *Bundle) calculateHash() string {
	h := sha256.New()

	// Sort file paths for deterministic hashing
	var paths []string
	for p := range b.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte(b.Files[p]))
	}

	return hex.EncodeToString(h.Sum(nil))[:16]
}

// EstimateTokens estimates the token count for this bundle
func (b *Bundle) EstimateTokens() int {
	total := 0

	// Count file contents
	total += token.EstimateMap(b.Files)

	// Count contracts
	total += token.EstimateMap(b.Contracts)

	// Count metadata
	if b.Meta != nil {
		total += token.EstimateString(b.Meta.Purpose)
		total += token.EstimateStrings(b.Meta.Invariants...)
	}

	// Add overhead for formatting (headers, code blocks, etc.)
	total = int(float64(total) * 1.1)

	return total
}

// Format formats the bundle as a string for the brain prompt
func (b *Bundle) Format() string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Node: %s\n\n", b.NodeID))

	// Node metadata (capsule)
	if b.Meta != nil {
		sb.WriteString("## Capsule\n\n")
		sb.WriteString(fmt.Sprintf("**Purpose:** %s\n\n", b.Meta.Purpose))

		if len(b.Meta.Invariants) > 0 {
			sb.WriteString("**Invariants:**\n")
			for _, inv := range b.Meta.Invariants {
				sb.WriteString(fmt.Sprintf("- %s\n", inv))
			}
			sb.WriteString("\n")
		}

		if len(b.Meta.NonGoals) > 0 {
			sb.WriteString("**Non-goals:**\n")
			for _, ng := range b.Meta.NonGoals {
				sb.WriteString(fmt.Sprintf("- %s\n", ng))
			}
			sb.WriteString("\n")
		}

		if len(b.Meta.Policies.AllowedPaths) > 0 {
			sb.WriteString(fmt.Sprintf("**Allowed write paths:** %v\n\n", b.Meta.Policies.AllowedPaths))
		}
	}

	// Dependency contracts
	if len(b.Contracts) > 0 {
		sb.WriteString("## Dependency Contracts\n\n")
		var contractPaths []string
		for p := range b.Contracts {
			contractPaths = append(contractPaths, p)
		}
		sort.Strings(contractPaths)

		for _, p := range contractPaths {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", p, b.Contracts[p]))
		}
	}

	// Source files
	sb.WriteString("## Source Files\n\n")

	var filePaths []string
	for p := range b.Files {
		filePaths = append(filePaths, p)
	}
	sort.Strings(filePaths)

	for _, p := range filePaths {
		ext := filepath.Ext(p)
		lang := extToLang(ext)
		sb.WriteString(fmt.Sprintf("### %s\n```%s\n%s\n```\n\n", p, lang, b.Files[p]))
	}

	return sb.String()
}

func extToLang(ext string) string {
	langs := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "jsx",
		".tsx":   "tsx",
		".rs":    "rust",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".sh":    "bash",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".sql":   "sql",
		".md":    "markdown",
	}
	if lang, ok := langs[ext]; ok {
		return lang
	}
	return ""
}
