package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/vectordb"
)

// LibrarianWebFetcher handles on-demand fetching from web sources like charm.li
type LibrarianWebFetcher struct {
	pinchtabTool *PinchTabTool
	webSources   *vectordb.WebSourceRegistry
	db           *vectordb.ChromaDB // For caching fetched pages
}

// NewLibrarianWebFetcher creates a new web fetcher
func NewLibrarianWebFetcher(pinchtabURL string, db *vectordb.ChromaDB) *LibrarianWebFetcher {
	return &LibrarianWebFetcher{
		pinchtabTool: NewPinchTabTool(pinchtabURL),
		webSources:   vectordb.NewWebSourceRegistry(),
		db:           db,
	}
}

// FetchFromCharmLi fetches manual content from charm.li using PinchTab
func (f *LibrarianWebFetcher) FetchFromCharmLi(ctx context.Context, make, model string, year int, query string) (*LibrarianWebResult, error) {
	// Get charm.li source
	charmLi := f.webSources.GetSourceByName("charm.li")
	if charmLi == nil {
		return nil, fmt.Errorf("charm.li source not available")
	}

	// Check if year is supported (1982-2013)
	if source, ok := charmLi.(*vectordb.CharmLiSource); ok {
		if !source.IsSupported(year) {
			return nil, fmt.Errorf("charm.li only supports years 1982-2013 (requested: %d)", year)
		}
	}

	// Get manual URL
	manualURL, err := charmLi.GetManualURL(make, model, year)
	if err != nil {
		return nil, fmt.Errorf("failed to construct charm.li URL: %w", err)
	}

	// Use PinchTab to navigate to the manual
	navResult := f.pinchtabTool.Execute(ctx, map[string]any{
		"action": "navigate",
		"url":    manualURL,
	})
	if navResult.Error != "" {
		return nil, fmt.Errorf("navigation failed: %s", navResult.Error)
	}

	// Extract text from the page (800 tokens/page - token efficient!)
	textResult := f.pinchtabTool.Execute(ctx, map[string]any{
		"action": "text",
	})
	if textResult.Error != "" {
		return nil, fmt.Errorf("text extraction failed: %s", textResult.Error)
	}

	// Parse the extracted text to find relevant sections
	pageText := textResult.ForLLM
	relevantSnippet := f.findRelevantSnippet(pageText, query)

	result := &LibrarianWebResult{
		Snippet:     relevantSnippet,
		Source:      fmt.Sprintf("%d %s %s Service Manual (charm.li)", year, make, model),
		URL:         manualURL,
		Make:        make,
		Model:       model,
		Year:        year,
		WebSource:   "charm.li",
		Confidence:  "medium", // Web sources get medium confidence by default
		TokensUsed:  len(pageText) / 4, // Rough estimate
	}

	// Cache this page in ChromaDB for future queries
	if f.db != nil {
		go f.cacheWebPage(context.Background(), result, pageText)
	}

	return result, nil
}

// findRelevantSnippet searches for query-relevant text in the page
func (f *LibrarianWebFetcher) findRelevantSnippet(pageText, query string) string {
	// Simple keyword matching for now
	// In production, use more sophisticated relevance scoring
	
	lines := strings.Split(pageText, "\n")
	queryLower := strings.ToLower(query)
	
	var relevantLines []string
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, queryLower) {
			relevantLines = append(relevantLines, line)
			if len(relevantLines) >= 5 {
				break
			}
		}
	}

	if len(relevantLines) > 0 {
		return strings.Join(relevantLines, "\n")
	}

	// If no exact match, return first 500 chars
	if len(pageText) > 500 {
		return pageText[:500] + "..."
	}
	return pageText
}

// cacheWebPage stores fetched web content in ChromaDB for future use
func (f *LibrarianWebFetcher) cacheWebPage(ctx context.Context, result *LibrarianWebResult, fullText string) {
	// Create chunks from the web page
	chunks := []vectordb.ManualChunk{
		{
			ID:   fmt.Sprintf("web_%s_%s_%d", result.Make, result.Model, result.Year),
			Text: fullText,
			Metadata: vectordb.ManualMetadata{
				Make:        result.Make,
				Model:       result.Model,
				Year:        result.Year,
				ManualType:  "service",
				ManualTitle: result.Source,
				SourcePath:  result.URL,
				PageNumber:  "web",
				Section:     "charm.li",
			},
		},
	}

	// Add to database (ignore errors - caching is best-effort)
	f.db.AddManualChunks(ctx, chunks)
}

// SearchWeb searches web sources for manuals
func (f *LibrarianWebFetcher) SearchWeb(ctx context.Context, make, model string, year int, query string) ([]*LibrarianWebResult, error) {
	var results []*LibrarianWebResult

	// Try charm.li first
	if year >= 1982 && year <= 2013 {
		result, err := f.FetchFromCharmLi(ctx, make, model, year, query)
		if err == nil {
			results = append(results, result)
		}
	}

	// Future: Add more web sources here
	// - Forums (mechanicadvice, justanswer)
	// - TSB databases
	// - OEM websites

	return results, nil
}

// LibrarianWebResult represents a result from web fetching
type LibrarianWebResult struct {
	Snippet    string
	Source     string
	URL        string
	Make       string
	Model      string
	Year       int
	WebSource  string // "charm.li", "forum", etc.
	Confidence string
	TokensUsed int
}

// HybridSearch performs both local ChromaDB search and web fetching
func (f *LibrarianWebFetcher) HybridSearch(ctx context.Context, query string, make, model string, year int, maxResults int) ([]string, error) {
	var allResults []string

	// 1. Search local ChromaDB first (fast, already indexed)
	// This would be called by the main Librarian tool

	// 2. If local results are insufficient, fetch from web
	webResults, err := f.SearchWeb(ctx, make, model, year, query)
	if err != nil {
		return nil, err
	}

	for _, result := range webResults {
		formatted := fmt.Sprintf(`Web Result:
  Snippet: %s
  Source: %s
  URL: %s
  Confidence: %s
  Tokens Used: %d (token-efficient!)
  
⚠️ Web source - verify against local manuals when possible`, 
			result.Snippet, result.Source, result.URL, result.Confidence, result.TokensUsed)
		
		allResults = append(allResults, formatted)
	}

	return allResults, nil
}
