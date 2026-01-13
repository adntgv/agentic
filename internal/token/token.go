package token

import (
	"fmt"
	"strings"
)

// EstimateString estimates tokens for a string
// Conservative: ~4 characters per token for code
func EstimateString(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4 // ~4 chars per token heuristic
}

// EstimateStrings estimates total tokens for multiple strings
func EstimateStrings(strings ...string) int {
	total := 0
	for _, s := range strings {
		total += EstimateString(s)
	}
	return total
}

// EstimateMap estimates tokens for a map of strings
func EstimateMap(m map[string]string) int {
	total := 0
	for _, content := range m {
		total += EstimateString(content)
	}
	return total
}

// CountLines counts the number of newlines in a string
func CountLines(s string) int {
	return strings.Count(s, "\n")
}

// Budget represents token limits for different models
type Budget struct {
	ContextWindow int
	Reserved      int // tokens reserved for output and system
	Available     int // context - reserved
}

// DefaultBudgets for common models
var DefaultBudgets = map[string]Budget{
	"claude-sonnet": {
		ContextWindow: 200000,
		Reserved:      20000,
		Available:     180000,
	},
	"claude-opus": {
		ContextWindow: 200000,
		Reserved:      20000,
		Available:     180000,
	},
	"claude-haiku": {
		ContextWindow: 200000,
		Reserved:      20000,
		Available:     180000,
	},
	"default": {
		ContextWindow: 100000,
		Reserved:      10000,
		Available:     90000,
	},
}

// GetBudget returns the token budget for a model
func GetBudget(model string) Budget {
	if budget, ok := DefaultBudgets[model]; ok {
		return budget
	}
	return DefaultBudgets["default"]
}

// CheckBudget checks if content fits within the budget
func CheckBudget(tokens int, budget Budget) error {
	if tokens > budget.Available {
		return &BudgetExceededError{
			Tokens:    tokens,
			Available: budget.Available,
		}
	}
	return nil
}

// BudgetExceededError indicates content exceeds token budget
type BudgetExceededError struct {
	Tokens    int
	Available int
}

func (e *BudgetExceededError) Error() string {
	return fmt.Sprintf("token budget exceeded: %d > %d", e.Tokens, e.Available)
}

// EstimatePrompt estimates tokens for a prompt with overhead
func EstimatePrompt(request string, contentTokens int) int {
	requestTokens := EstimateString(request)
	overhead := 500 // system prompt overhead
	return requestTokens + contentTokens + overhead
}