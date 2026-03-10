package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/vectordb"
)

// LibrarianTool provides semantic search over repair manuals and technical documentation.
type LibrarianTool struct {
	db            *vectordb.ChromaDB
	workspacePath string
	enabled       bool
}

// NewLibrarianTool creates a new Librarian tool instance
func NewLibrarianTool(workspacePath string) *LibrarianTool {
	return &LibrarianTool{
		workspacePath: workspacePath,
		enabled:       false, // Disabled until DB is initialized
	}
}

// Initialize sets up the ChromaDB connection
func (t *LibrarianTool) Initialize() error {
	db, err := vectordb.NewChromaDB(t.workspacePath)
	if err != nil {
		return fmt.Errorf("failed to initialize ChromaDB: %w", err)
	}
	t.db = db
	t.enabled = true
	return nil
}

func (t *LibrarianTool) Name() string {
	return "librarian"
}

func (t *LibrarianTool) Description() string {
	return `Search repair manuals and technical documentation for verified specifications.
Returns structured citations with source information.

CRITICAL: This tool is the ONLY source for technical specifications. Never provide specs without a Librarian citation.

Use this for:
- Torque specifications
- Wiring diagrams and pinouts
- Fuse ratings and locations
- Fluid capacities
- Part numbers
- Safety procedures
- TSB (Technical Service Bulletin) information`
}

func (t *LibrarianTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query for the manual (e.g., 'oil drain plug torque', 'fuse box diagram', 'alternator wiring')",
			},
			"machine_id": map[string]any{
				"type":        "string",
				"description": "VIN or machine identifier to search machine-specific manuals",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 3)",
				"minimum":     1.0,
				"maximum":     10.0,
			},
		},
		"required": []string{"query"},
	}
}

func (t *LibrarianTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return ErrorResult("query is required")
	}

	// Check if ChromaDB is initialized
	if !t.enabled || t.db == nil {
		return &ToolResult{
			ForLLM:  "⚠️ Librarian database not initialized. No manuals have been ingested yet.",
			ForUser: "🔍 Searching repair manuals... (No manuals available)",
		}
	}

	machineID := ""
	if id, ok := args["machine_id"].(string); ok {
		machineID = id
	}

	maxResults := 3
	if max, ok := args["max_results"].(float64); ok {
		maxResults = int(max)
	}

	// Perform semantic search
	var results []vectordb.SearchResult
	var err error

	if machineID != "" {
		// Try VIN-specific search first
		results, err = t.db.SearchByVIN(ctx, query, machineID, maxResults)
		if err != nil {
			return ErrorResult(fmt.Sprintf("search failed: %v", err))
		}

		// If no VIN-specific results, fall back to general search
		if len(results) == 0 {
			results, err = t.db.Search(ctx, query, nil, maxResults)
			if err != nil {
				return ErrorResult(fmt.Sprintf("search failed: %v", err))
			}
		}
	} else {
		// General search without VIN filter
		results, err = t.db.Search(ctx, query, nil, maxResults)
		if err != nil {
			return ErrorResult(fmt.Sprintf("search failed: %v", err))
		}
	}

	// Format results
	if len(results) == 0 {
		return &ToolResult{
			ForLLM:  fmt.Sprintf("No results found for query: \"%s\"", query),
			ForUser: "🔍 No matching repair manual information found.",
		}
	}

	// Build structured response
	var llmOutput strings.Builder
	llmOutput.WriteString(fmt.Sprintf("Found %d result(s) for: \"%s\"\n\n", len(results), query))

	for i, result := range results {
		confidence := "medium"
		if result.Distance < 0.3 {
			confidence = "high"
		} else if result.Distance > 0.6 {
			confidence = "low"
		}

		llmOutput.WriteString(fmt.Sprintf("Result %d:\n", i+1))
		llmOutput.WriteString(fmt.Sprintf("  Snippet: %s\n", truncateText(result.Text, 200)))
		llmOutput.WriteString(fmt.Sprintf("  Source: %s\n", result.Metadata.ManualTitle))
		llmOutput.WriteString(fmt.Sprintf("  Page: %s\n", result.Metadata.PageNumber))
		if result.Metadata.Section != "" {
			llmOutput.WriteString(fmt.Sprintf("  Section: %s\n", result.Metadata.Section))
		}
		llmOutput.WriteString(fmt.Sprintf("  Path: %s\n", result.Metadata.SourcePath))
		llmOutput.WriteString(fmt.Sprintf("  Confidence: %s\n", confidence))
		llmOutput.WriteString(fmt.Sprintf("  Distance: %.3f\n", result.Distance))
		llmOutput.WriteString("\n")
	}

	llmOutput.WriteString("---\n")
	llmOutput.WriteString("⚠️ HARD-LINK REQUIREMENT: You MUST cite the source when providing this information to the user.\n")

	userOutput := fmt.Sprintf("🔍 Found %d result(s) in repair manuals", len(results))

	return &ToolResult{
		ForLLM:  llmOutput.String(),
		ForUser: userOutput,
	}
}

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// LibrarianResult represents a structured search result from the Librarian
// This defines the contract for what the tool will return when fully implemented
type LibrarianResult struct {
	Snippet      string  `json:"snippet"`       // The relevant text excerpt
	Source       string  `json:"source"`        // Manual name/title
	Page         string  `json:"page"`          // Page number or section
	Section      string  `json:"section"`       // Section/chapter name
	Path         string  `json:"path"`          // File path or URL
	MachineID    string  `json:"machine_id"`    // VIN or machine identifier
	Confidence   string  `json:"confidence"`    // high/medium/low
	Notes        string  `json:"notes"`         // Optional additional context
	RetrievedAt  string  `json:"retrieved_at"`  // Timestamp
}
