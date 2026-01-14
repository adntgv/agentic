package brain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/aid/agentic/internal/bundle"
)

// FileOutput represents a complete file to be written
type FileOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Response represents the structured response from the brain
type Response struct {
	Files   []FileOutput `json:"files"`
	Message string       `json:"message,omitempty"`
}

// ClaudeResponse is the raw response structure from Claude CLI
type ClaudeResponse struct {
	Type      string `json:"type"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
	CostUSD   float64 `json:"cost_usd,omitempty"`
}

// BrainAdapter defines the interface for AI adapters
type BrainAdapter interface {
	Call(request string, b *bundle.Bundle) (*Response, error)
}

// ClaudeAdapter implements BrainAdapter for Claude CLI
type ClaudeAdapter struct{}

// GeminiAdapter implements BrainAdapter for Gemini (placeholder)
type GeminiAdapter struct{}

// CodexAdapter implements BrainAdapter for Codex (placeholder)
type CodexAdapter struct{}

// GetAdapter returns a BrainAdapter by name
func GetAdapter(name string) BrainAdapter {
	switch name {
	case "claude":
		return &ClaudeAdapter{}
	case "gemini":
		return &GeminiAdapter{}
	case "codex":
		return &CodexAdapter{}
	default:
		return nil
	}
}

// Call sends a request using the default adapter (Claude)
func Call(request string, b *bundle.Bundle) (*Response, error) {
	return (&ClaudeAdapter{}).Call(request, b)
}

// Call sends a request to Claude Code and returns file outputs
func (a *ClaudeAdapter) Call(request string, b *bundle.Bundle) (*Response, error) {
	prompt := BuildPrompt(request, b)

	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("claude CLI not found. Install Claude Code: https://claude.ai/code")
	}

	args := []string{
		"-p", prompt,
		"--output-format", "json",
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir, _ = os.Getwd()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	fmt.Println("Calling Claude Code...")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude command failed: %w\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()

	// Parse Claude CLI JSON response
	var claudeResp ClaudeResponse
	if err := json.Unmarshal([]byte(output), &claudeResp); err != nil {
		// Try to extract files from raw output
		return ExtractFiles(output, b)
	}

	if claudeResp.IsError {
		return nil, fmt.Errorf("claude error: %s", claudeResp.Result)
	}

	return ExtractFiles(claudeResp.Result, b)
}

// BuildPrompt creates the prompt requesting complete file output
func BuildPrompt(request string, b *bundle.Bundle) string {
	var sb strings.Builder

	sb.WriteString(`You are an AI assistant modifying code files.

CRITICAL OUTPUT RULES - VIOLATIONS WILL CAUSE ERRORS:
1. Output ONLY the file format below - NO markdown fences, NO explanations
2. Return COMPLETE files - never truncate, never use "..." or "// rest unchanged"
3. If a file needs no changes, do not include it
4. Every === FILE: MUST have a matching === END FILE ===

REQUIRED OUTPUT FORMAT:
=== FILE: path/to/file.go ===
package main

// complete file content here - every single line
=== END FILE ===

STRICTLY FORBIDDEN (will cause parse errors):
- ` + "`" + "`" + "`" + ` markdown code fences anywhere in output
- Partial files or "// ... rest of file unchanged"
- Explanatory text outside the === FILE: format
- Starting response with anything other than === FILE:

`)

	sb.WriteString("USER REQUEST:\n")
	sb.WriteString(request)
	sb.WriteString("\n\n")

	sb.WriteString("CURRENT FILES:\n\n")

	// List files with their content
	var paths []string
	for p := range b.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		content := b.Files[p]
		sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n--- END %s ---\n\n", p, content, p))
	}

	if b.Meta != nil {
		sb.WriteString("CONSTRAINTS:\n")
		sb.WriteString(fmt.Sprintf("Purpose: %s\n", b.Meta.Purpose))
		if len(b.Meta.Invariants) > 0 {
			sb.WriteString("Invariants:\n")
			for _, inv := range b.Meta.Invariants {
				sb.WriteString(fmt.Sprintf("- %s\n", inv))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// sanitizeOutput cleans up common LLM output issues
func sanitizeOutput(raw string) string {
	// Strip markdown fences wrapping entire output
	raw = strings.TrimPrefix(raw, "```go\n")
	raw = strings.TrimPrefix(raw, "```\n")
	raw = strings.TrimSuffix(raw, "\n```")
	raw = strings.TrimSuffix(raw, "```")

	// Remove leading text before first === FILE:
	if idx := strings.Index(raw, "=== FILE:"); idx > 0 {
		raw = raw[idx:]
	}

	return strings.TrimSpace(raw)
}

// detectTruncation checks if output was truncated mid-file
func detectTruncation(content string) (bool, string) {
	fileStarts := strings.Count(content, "=== FILE:")
	fileEnds := strings.Count(content, "=== END FILE ===")
	if fileStarts != fileEnds {
		return true, fmt.Sprintf("incomplete output: %d files started, %d ended", fileStarts, fileEnds)
	}
	return false, ""
}

// validateGoSyntax checks Go files for syntax errors
func validateGoSyntax(path, content string) error {
	if !strings.HasSuffix(path, ".go") {
		return nil
	}
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, path, content, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("syntax error in %s: %w", path, err)
	}
	return nil
}

// ExtractFiles parses the response to extract complete file contents
func ExtractFiles(response string, b *bundle.Bundle) (*Response, error) {
	// Sanitize output first
	response = sanitizeOutput(response)

	// Check for truncation
	if truncated, msg := detectTruncation(response); truncated {
		return nil, fmt.Errorf("LLM output truncated: %s", msg)
	}

	resp := &Response{
		Files: []FileOutput{},
	}

	// Pattern: === FILE: path === ... === END FILE ===
	filePattern := regexp.MustCompile(`(?s)=== FILE: (.+?) ===\n(.*?)=== END FILE ===`)
	matches := filePattern.FindAllStringSubmatch(response, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			path := strings.TrimSpace(match[1])
			content := match[2]

			// Remove trailing newline if present
			content = strings.TrimSuffix(content, "\n")

			// Validate Go syntax before accepting
			if err := validateGoSyntax(path, content); err != nil {
				return nil, err
			}

			resp.Files = append(resp.Files, FileOutput{
				Path:    path,
				Content: content,
			})
		}
	}

	// If no structured output found, try to find code blocks with file paths
	if len(resp.Files) == 0 {
		resp.Files = extractFromCodeBlocks(response, b)

		// Validate extracted files too
		for _, f := range resp.Files {
			if err := validateGoSyntax(f.Path, f.Content); err != nil {
				return nil, err
			}
		}
	}

	if len(resp.Files) == 0 {
		resp.Message = response
	}

	return resp, nil
}

// extractFromCodeBlocks tries to extract files from markdown code blocks
func extractFromCodeBlocks(response string, b *bundle.Bundle) []FileOutput {
	var files []FileOutput

	// Look for patterns like "### path/to/file.go" followed by code block
	// or "```go\n// path/to/file.go"
	lines := strings.Split(response, "\n")
	var currentPath string
	var inCodeBlock bool
	var content strings.Builder
	var lang string

	for _, line := range lines {
		// Check for file path header
		if strings.HasPrefix(line, "### ") {
			path := strings.TrimPrefix(line, "### ")
			path = strings.TrimSpace(path)
			if _, exists := b.Files[path]; exists {
				currentPath = path
			}
			continue
		}

		// Check for code block start
		if strings.HasPrefix(line, "```") && !inCodeBlock {
			inCodeBlock = true
			lang = strings.TrimPrefix(line, "```")
			content.Reset()
			continue
		}

		// Check for code block end
		if line == "```" && inCodeBlock {
			inCodeBlock = false
			if currentPath != "" && content.Len() > 0 {
				files = append(files, FileOutput{
					Path:    currentPath,
					Content: strings.TrimSuffix(content.String(), "\n"),
				})
				currentPath = ""
			}
			continue
		}

		// Accumulate content in code block
		if inCodeBlock {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(line)
		}
	}

	_ = lang // unused but might be useful for validation
	return files
}

func extToLang(path string) string {
	if strings.HasSuffix(path, ".go") {
		return "go"
	} else if strings.HasSuffix(path, ".py") {
		return "python"
	} else if strings.HasSuffix(path, ".js") {
		return "javascript"
	} else if strings.HasSuffix(path, ".ts") {
		return "typescript"
	} else if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return "yaml"
	} else if strings.HasSuffix(path, ".json") {
		return "json"
	}
	return ""
}

// Call returns an error indicating Gemini is not yet implemented
func (a *GeminiAdapter) Call(request string, b *bundle.Bundle) (*Response, error) {
	return nil, fmt.Errorf("gemini adapter not implemented yet")
}

// Call returns an error indicating Codex is not yet implemented
func (a *CodexAdapter) Call(request string, b *bundle.Bundle) (*Response, error) {
	return nil, fmt.Errorf("codex adapter not implemented yet")
}

// CheckAvailable checks if the Claude CLI is available
func CheckAvailable() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH")
	}

	cmd := exec.Command("claude", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI found but not working: %w", err)
	}

	return nil
}
