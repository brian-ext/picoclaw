package vectordb

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// WebSource represents an online manual repository
type WebSource interface {
	Name() string
	Search(ctx context.Context, make, model string, year int, query string) ([]WebManualResult, error)
	GetManualURL(make, model string, year int) (string, error)
}

// WebManualResult represents a search result from a web source
type WebManualResult struct {
	Title       string
	URL         string
	Make        string
	Model       string
	Year        int
	Source      string // "charm.li", "forum", etc.
	Description string
}

// CharmLiSource implements WebSource for charm.li
type CharmLiSource struct {
	baseURL string
}

// NewCharmLiSource creates a new charm.li source
func NewCharmLiSource() *CharmLiSource {
	return &CharmLiSource{
		baseURL: "https://charm.li",
	}
}

func (c *CharmLiSource) Name() string {
	return "charm.li"
}

// GetManualURL constructs the URL to a specific manual on charm.li
func (c *CharmLiSource) GetManualURL(make, model string, year int) (string, error) {
	// charm.li structure: https://charm.li/Make/Model/Year/
	// Example: https://charm.li/Honda/Civic/1999/
	
	// Normalize make and model (capitalize first letter, handle spaces)
	make = normalizeForURL(make)
	model = normalizeForURL(model)
	
	manualURL := fmt.Sprintf("%s/%s/%s/%d/", c.baseURL, make, model, year)
	return manualURL, nil
}

// Search finds manuals on charm.li (requires PinchTab to navigate and extract)
func (c *CharmLiSource) Search(ctx context.Context, make, model string, year int, query string) ([]WebManualResult, error) {
	// This will be called by Librarian with PinchTab to navigate charm.li
	manualURL, err := c.GetManualURL(make, model, year)
	if err != nil {
		return nil, err
	}

	result := WebManualResult{
		Title:       fmt.Sprintf("%d %s %s Service Manual", year, make, model),
		URL:         manualURL,
		Make:        make,
		Model:       model,
		Year:        year,
		Source:      "charm.li",
		Description: fmt.Sprintf("Factory service manual for %d %s %s (1982-2013)", year, make, model),
	}

	return []WebManualResult{result}, nil
}

// normalizeForURL formats a string for charm.li URLs
func normalizeForURL(s string) string {
	// Capitalize first letter of each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	result := strings.Join(words, "%20")
	return url.PathEscape(result)
}

// GetMakeURL returns the URL for a specific make on charm.li
func (c *CharmLiSource) GetMakeURL(make string) string {
	return fmt.Sprintf("%s/%s/", c.baseURL, normalizeForURL(make))
}

// GetModelURL returns the URL for a specific model on charm.li
func (c *CharmLiSource) GetModelURL(make, model string) string {
	return fmt.Sprintf("%s/%s/%s/", c.baseURL, normalizeForURL(make), normalizeForURL(model))
}

// SupportedYears returns the year range supported by charm.li
func (c *CharmLiSource) SupportedYears() (int, int) {
	return 1982, 2013
}

// IsSupported checks if a year is supported by charm.li
func (c *CharmLiSource) IsSupported(year int) bool {
	minYear, maxYear := c.SupportedYears()
	return year >= minYear && year <= maxYear
}

// WebSourceRegistry manages multiple web sources
type WebSourceRegistry struct {
	sources []WebSource
}

// NewWebSourceRegistry creates a new registry
func NewWebSourceRegistry() *WebSourceRegistry {
	return &WebSourceRegistry{
		sources: []WebSource{
			NewCharmLiSource(),
			// Future: Add more sources (forums, TSB databases, etc.)
		},
	}
}

// GetSources returns all registered sources
func (r *WebSourceRegistry) GetSources() []WebSource {
	return r.sources
}

// FindManual searches all sources for a manual
func (r *WebSourceRegistry) FindManual(ctx context.Context, make, model string, year int) ([]WebManualResult, error) {
	var allResults []WebManualResult

	for _, source := range r.sources {
		results, err := source.Search(ctx, make, model, year, "")
		if err != nil {
			// Log error but continue with other sources
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// GetSourceByName returns a source by name
func (r *WebSourceRegistry) GetSourceByName(name string) WebSource {
	for _, source := range r.sources {
		if source.Name() == name {
			return source
		}
	}
	return nil
}
