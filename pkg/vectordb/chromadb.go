package vectordb

import (
	"context"
	"fmt"
	"path/filepath"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/types"
)

// ChromaDB wraps the ChromaDB client for repair manual storage
type ChromaDB struct {
	client     *chroma.Client
	collection *chroma.Collection
	dbPath     string
}

// ManualChunk represents a chunk of a repair manual
type ManualChunk struct {
	ID       string
	Text     string
	Metadata ManualMetadata
}

// ManualMetadata contains metadata for a manual chunk
type ManualMetadata struct {
	VIN          string `json:"vin,omitempty"`           // Vehicle VIN (if vehicle-specific)
	Make         string `json:"make"`                    // Manufacturer
	Model        string `json:"model"`                   // Model name
	Year         int    `json:"year"`                    // Model year
	ManualType   string `json:"manual_type"`             // "service", "owner", "tsb", "wiring"
	ManualTitle  string `json:"manual_title"`            // Full manual title
	SourcePath   string `json:"source_path"`             // Original PDF path
	PageNumber   string `json:"page_number"`             // Page or section number
	Section      string `json:"section,omitempty"`       // Section/chapter name
	ChunkIndex   int    `json:"chunk_index"`             // Chunk number within page
	TotalChunks  int    `json:"total_chunks,omitempty"`  // Total chunks for this page
}

// SearchResult represents a search result from ChromaDB
type SearchResult struct {
	ID       string
	Text     string
	Metadata ManualMetadata
	Distance float32 // Lower is better (cosine distance)
}

// NewChromaDB creates a new ChromaDB client with persistent storage
func NewChromaDB(workspacePath string) (*ChromaDB, error) {
	dbPath := filepath.Join(workspacePath, "vector_db")
	
	// Create persistent client (stores data locally)
	client, err := chroma.NewPersistentClient(
		chroma.WithPersistDirectory(dbPath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	db := &ChromaDB{
		client: client,
		dbPath: dbPath,
	}

	// Get or create the manuals collection
	if err := db.initCollection(); err != nil {
		return nil, err
	}

	return db, nil
}

// NewChromaDBWithServer creates a ChromaDB client that connects to a server
func NewChromaDBWithServer(serverURL string) (*ChromaDB, error) {
	client, err := chroma.NewClient(
		chroma.WithBasePath(serverURL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w", err)
	}

	db := &ChromaDB{
		client: client,
	}

	if err := db.initCollection(); err != nil {
		return nil, err
	}

	return db, nil
}

// initCollection gets or creates the manuals collection
func (db *ChromaDB) initCollection() error {
	ctx := context.Background()

	// Try to get existing collection
	collection, err := db.client.GetCollection(ctx, "repair_manuals", nil)
	if err != nil {
		// Collection doesn't exist, create it
		collection, err = db.client.CreateCollection(ctx, "repair_manuals", map[string]any{
			"description": "Repair manuals, service bulletins, and technical documentation",
		}, true, nil, types.L2)
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}

	db.collection = collection
	return nil
}

// AddManualChunks adds manual chunks to the vector database
func (db *ChromaDB) AddManualChunks(ctx context.Context, chunks []ManualChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	ids := make([]string, len(chunks))
	texts := make([]string, len(chunks))
	metadatas := make([]map[string]any, len(chunks))

	for i, chunk := range chunks {
		ids[i] = chunk.ID
		texts[i] = chunk.Text
		metadatas[i] = map[string]any{
			"vin":          chunk.Metadata.VIN,
			"make":         chunk.Metadata.Make,
			"model":        chunk.Metadata.Model,
			"year":         chunk.Metadata.Year,
			"manual_type":  chunk.Metadata.ManualType,
			"manual_title": chunk.Metadata.ManualTitle,
			"source_path":  chunk.Metadata.SourcePath,
			"page_number":  chunk.Metadata.PageNumber,
			"section":      chunk.Metadata.Section,
			"chunk_index":  chunk.Metadata.ChunkIndex,
			"total_chunks": chunk.Metadata.TotalChunks,
		}
	}

	_, err := db.collection.Add(ctx,
		chroma.WithIDs(ids...),
		chroma.WithTexts(texts...),
		chroma.WithMetadatas(metadatas...),
	)

	return err
}

// Search performs semantic search on the manual collection
func (db *ChromaDB) Search(ctx context.Context, query string, filters map[string]any, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	// Build where filter from provided filters
	var where map[string]any
	if len(filters) > 0 {
		where = filters
	}

	// Perform query
	results, err := db.collection.Query(ctx,
		[]string{query},
		int32(limit),
		where,
		nil, // whereDocument
		nil, // include
	)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results
	var searchResults []SearchResult
	if len(results.Documents) > 0 && len(results.Documents[0]) > 0 {
		for i := 0; i < len(results.Documents[0]); i++ {
			result := SearchResult{
				ID:   results.Ids[0][i],
				Text: results.Documents[0][i],
			}

			// Extract distance if available
			if len(results.Distances) > 0 && len(results.Distances[0]) > i {
				result.Distance = results.Distances[0][i]
			}

			// Extract metadata
			if len(results.Metadatas) > 0 && len(results.Metadatas[0]) > i {
				meta := results.Metadatas[0][i]
				result.Metadata = ManualMetadata{
					VIN:         getStringFromMeta(meta, "vin"),
					Make:        getStringFromMeta(meta, "make"),
					Model:       getStringFromMeta(meta, "model"),
					Year:        getIntFromMeta(meta, "year"),
					ManualType:  getStringFromMeta(meta, "manual_type"),
					ManualTitle: getStringFromMeta(meta, "manual_title"),
					SourcePath:  getStringFromMeta(meta, "source_path"),
					PageNumber:  getStringFromMeta(meta, "page_number"),
					Section:     getStringFromMeta(meta, "section"),
					ChunkIndex:  getIntFromMeta(meta, "chunk_index"),
					TotalChunks: getIntFromMeta(meta, "total_chunks"),
				}
			}

			searchResults = append(searchResults, result)
		}
	}

	return searchResults, nil
}

// SearchByVIN searches manuals specific to a VIN
func (db *ChromaDB) SearchByVIN(ctx context.Context, query string, vin string, limit int) ([]SearchResult, error) {
	filters := map[string]any{
		"vin": vin,
	}
	return db.Search(ctx, query, filters, limit)
}

// SearchByMakeModel searches manuals for a specific make/model/year
func (db *ChromaDB) SearchByMakeModel(ctx context.Context, query string, make, model string, year int, limit int) ([]SearchResult, error) {
	filters := map[string]any{
		"make":  make,
		"model": model,
		"year":  year,
	}
	return db.Search(ctx, query, filters, limit)
}

// DeleteManual removes all chunks for a specific manual
func (db *ChromaDB) DeleteManual(ctx context.Context, sourcePath string) error {
	where := map[string]any{
		"source_path": sourcePath,
	}

	_, err := db.collection.Delete(ctx, nil, where, nil)
	return err
}

// GetCollectionCount returns the total number of chunks in the database
func (db *ChromaDB) GetCollectionCount(ctx context.Context) (int, error) {
	count, err := db.collection.Count(ctx)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// Close closes the ChromaDB client
func (db *ChromaDB) Close() error {
	// ChromaDB Go client doesn't require explicit close
	return nil
}

// Helper functions to extract metadata values

func getStringFromMeta(meta map[string]any, key string) string {
	if val, ok := meta[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getIntFromMeta(meta map[string]any, key string) int {
	if val, ok := meta[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}
