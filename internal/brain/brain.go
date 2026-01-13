package brain

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// Call sends a request to Claude Code and returns file outputs
func Call(request string, b *bundle.Bundle) (*Response, error) {
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

	sb.WriteString("You are an AI assistant helping to modify code.\n\n")

	sb.WriteString("## User Request\n\n")
	sb.WriteString(request)
	sb.WriteString("\n\n")

	sb.WriteString("## Current Files\n\n")

	// List files with their content
	var paths []string
	for p := range b.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		content := b.Files[p]
		lang := extToLang(p)
		sb.WriteString(fmt.Sprintf("### %s\n```%s\n%s\n```\n\n", p, lang, content))
	}

	if b.Meta != nil {
		sb.WriteString("## Constraints\n\n")
		sb.WriteString(fmt.Sprintf("**Purpose:** %s\n\n", b.Meta.Purpose))
		if len(b.Meta.Invariants) > 0 {
			sb.WriteString("**Invariants to maintain:**\n")
			for _, inv := range b.Meta.Invariants {
				sb.WriteString(fmt.Sprintf("- %s\n", inv))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Make the requested changes\n")
	sb.WriteString("2. Return the COMPLETE updated file(s)\n")
	sb.WriteString("3. Use this exact format for each file:\n\n")
	sb.WriteString("```\n")
	sb.WriteString("=== FILE: path/to/file.go ===\n")
	sb.WriteString("<complete file content here>\n")
	sb.WriteString("=== END FILE ===\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Only include files that need changes. Return complete file contents, not diffs.\n")

	return sb.String()
}

// ExtractFiles parses the response to extract complete file contents
func ExtractFiles(response string, b *bundle.Bundle) (*Response, error) {
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

			resp.Files = append(resp.Files, FileOutput{
				Path:    path,
				Content: content,
			})
		}
	}

	// If no structured output found, try to find code blocks with file paths
	if len(resp.Files) == 0 {
		resp.Files = extractFromCodeBlocks(response, b)
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
