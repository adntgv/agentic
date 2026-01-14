# agentic

A CLI tool for orchestrating recursive graph-of-graphs with pluggable AI brain adapters. Break large codebases into focused nodes, let AI work on each piece within defined boundaries.

## Installation

```bash
go install github.com/aid/agentic@latest
```

Or build from source:

```bash
git clone https://github.com/aid/agentic.git
cd agentic
go build -o agentic .
```

Requires [Claude Code](https://claude.ai/code) CLI installed and authenticated.

## Quick Start

```bash
# Initialize a new project
agentic init

# Discover and validate the graph
agentic validate

# Run a task on a specific node
agentic run "add input validation" -n api

# Interactive mode
agentic repl
```

## Core Concepts

### Graph Structure

Agentic organizes code into a **graph-of-graphs**. Each node represents a focused area of code with defined boundaries, dependencies, and constraints.

```
project/
├── GRAPH.manifest      # Root graph definition
├── NODE.meta.yaml      # Root node metadata (optional)
├── src/
│   ├── api/
│   │   ├── GRAPH.manifest    # Subgraph (composite node)
│   │   └── NODE.meta.yaml
│   └── utils/
│       └── NODE.meta.yaml    # Leaf node
```

### GRAPH.manifest

Defines nodes and their relationships in a compact text format:

```
# Comment lines start with #
L: utils       ./src/utils                    # Leaf node
L: models      ./src/models
C: api         ./src/api         utils,models # Composite node with dependencies
C: cli         ./src/cli         api
```

- `L:` - Leaf node (no subgraph)
- `C:` - Composite node (has its own GRAPH.manifest)
- Dependencies listed after path, comma-separated

### NODE.meta.yaml

Optional metadata for each node:

```yaml
purpose: "HTTP API handlers for user management"

invariants:
  - "All endpoints require authentication"
  - "Input validation on all handlers"

non_goals:
  - "Direct database access"

budget:
  max_tokens: 50000

policies:
  allowed_paths:
    - "src/api/**"
  denied_paths:
    - "**/*_test.go"
```

## CLI Commands

### `agentic init`

Initialize a new agentic project. Creates GRAPH.manifest template.

### `agentic validate`

Validate the graph structure:
- Check for missing dependencies
- Detect circular references
- Verify manifest syntax
- Check NODE.meta.yaml schemas

```bash
agentic validate
agentic validate -v  # Verbose output
```

### `agentic run <request>`

Run an AI task on one or more nodes:

```bash
# Single node
agentic run "add error handling" -n api

# All nodes
agentic run "add copyright headers"

# Parallel mode (process independent nodes concurrently)
agentic run "format code" -P

# Verbose mode (show token counts, prompts)
agentic run "refactor" -n utils -v
```

Flags:
- `-n, --node <id>` - Target specific node
- `-v, --verbose` - Show bundle details, token estimates, prompts
- `-P, --parallel` - Process independent nodes in parallel

### `agentic show`

Display graph information:

```bash
agentic show              # Show all nodes
agentic show -n api       # Show specific node details
agentic show --deps       # Show dependency tree
agentic show --dirty      # Show nodes with uncommitted changes
```

### `agentic diff`

Show staged changes before committing:

```bash
agentic diff
agentic diff -n api
```

### `agentic commit`

Apply staged changes to files:

```bash
agentic commit
agentic commit -n api
```

### `agentic undo`

Revert the last committed changes:

```bash
agentic undo
agentic undo -n api
```

### `agentic repl`

Interactive mode for multi-turn conversations:

```bash
agentic repl
> focus api          # Set current node
> add input validation
> show staged
> commit
> exit
```

### `agentic check`

Check external requirements:

```bash
agentic check         # Verify Claude CLI available
```

## Configuration

### Environment Variables

- `AGENTIC_BRAIN` - AI backend to use (default: `claude`)
- `AGENTIC_VERBOSE` - Enable verbose output by default

### Supported Brains

- `claude` - Claude Code CLI (default, fully implemented)
- `gemini` - Google Gemini (placeholder)
- `codex` - OpenAI Codex (placeholder)

## How It Works

1. **Bundle**: Collects all files in the target node directory, excluding binary files and common ignore patterns (.git, node_modules, etc.)

2. **Prompt**: Sends the bundle with your request to the AI, along with node constraints (purpose, invariants, policies)

3. **Parse**: Extracts complete file outputs from the AI response using a structured format

4. **Stage**: Stages changes for review (shows diff)

5. **Commit**: Writes changes to disk

### Output Format

The AI returns files in this format:

```
=== FILE: path/to/file.go ===
package main
// complete file content
=== END FILE ===
```

This avoids diff/patch complexity - always complete file replacement.

## Features

- **Recursive Graphs**: Composite nodes can contain their own GRAPH.manifest
- **Dependency Contracts**: Share interfaces between nodes via CONTRACTS directory
- **Policy Enforcement**: Restrict file paths the AI can modify
- **Token Budgets**: Set limits per node to control costs
- **Bundle Caching**: Skip re-reading unchanged files
- **Parallel Execution**: Process independent nodes concurrently
- **Output Sanitization**: Handles markdown fences, truncation detection, Go syntax validation
- **Dirty Tracking**: Identify nodes with uncommitted changes

## Project Structure

```
internal/
├── brain/       # AI adapter interface and implementations
├── bundle/      # File collection and caching
├── cli/         # Command definitions and REPL
├── graph/       # Graph parsing and validation
├── policy/      # Path restrictions
├── token/       # Token estimation
├── validate/    # Schema and structure validation
└── workspace/   # Change staging and history
```

## Testing

```bash
./tests/run-tests.sh
```

Tests cover graph parsing, validation, bundle building, and CLI commands.

## License

MIT
