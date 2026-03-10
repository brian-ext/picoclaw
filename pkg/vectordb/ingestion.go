package vectordb

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ManualIngester handles PDF manual ingestion into ChromaDB
type ManualIngester struct {
	db            *ChromaDB
	chunkSize     int // Characters per chunk
	chunkOverlap  int // Overlap between chunks
}

// NewManualIngester creates a new manual ingestion handler
func NewManualIngester(db *ChromaDB) *ManualIngester {
	return &ManualIngester{
		db:           db,
		chunkSize:    1000, // ~250 tokens
		chunkOverlap: 200,  // ~50 tokens overlap
	}
}

// IngestManual processes a PDF manual and adds it to the vector database
func (m *ManualIngester) IngestManual(ctx context.Context, pdfPath string, metadata ManualMetadata) error {
	// Validate file exists
	if _, err := os.Stat(pdfPath); err != nil {
		return fmt.Errorf("PDF file not found: %w", err)
	}

	// Extract text from PDF
	text, pageTexts, err := m.extractPDFText(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to extract PDF text: %w", err)
	}

	// Set source path in metadata
	metadata.SourcePath = pdfPath

	// Create chunks
	chunks := m.createChunks(text, pageTexts, metadata)

	// Add to ChromaDB
	if err := m.db.AddManualChunks(ctx, chunks); err != nil {
		return fmt.Errorf("failed to add chunks to database: %w", err)
	}

	return nil
}

// extractPDFText extracts text from a PDF file
func (m *ManualIngester) extractPDFText(pdfPath string) (string, map[int]string, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	var fullText strings.Builder
	pageTexts := make(map[int]string)

	totalPages := r.NumPage()
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			// Skip pages with extraction errors
			continue
		}

		pageTexts[pageNum] = text
		fullText.WriteString(text)
		fullText.WriteString("\n\n")
	}

	return fullText.String(), pageTexts, nil
}

// createChunks splits text into overlapping chunks
func (m *ManualIngester) createChunks(fullText string, pageTexts map[int]string, metadata ManualMetadata) []ManualChunk {
	var chunks []ManualChunk
	chunkID := 0

	// Process each page separately to maintain page context
	for pageNum := 1; pageNum <= len(pageTexts); pageNum++ {
		pageText, ok := pageTexts[pageNum]
		if !ok || len(pageText) == 0 {
			continue
		}

		pageChunks := m.chunkText(pageText, metadata, pageNum)
		for i, chunk := range pageChunks {
			chunk.ID = m.generateChunkID(metadata.SourcePath, pageNum, i)
			chunk.Metadata.ChunkIndex = i
			chunk.Metadata.TotalChunks = len(pageChunks)
			chunks = append(chunks, chunk)
			chunkID++
		}
	}

	return chunks
}

// chunkText splits text into chunks with overlap
func (m *ManualIngester) chunkText(text string, metadata ManualMetadata, pageNum int) []ManualChunk {
	var chunks []ManualChunk
	
	// Clean text
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return chunks
	}

	// Simple chunking by character count with overlap
	start := 0
	for start < len(text) {
		end := start + m.chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at sentence boundary
		if end < len(text) {
			// Look for sentence endings near the chunk boundary
			for i := end; i > start+m.chunkSize-100 && i > start; i-- {
				if text[i] == '.' || text[i] == '\n' {
					end = i + 1
					break
				}
			}
		}

		chunkText := strings.TrimSpace(text[start:end])
		if len(chunkText) > 0 {
			chunk := ManualChunk{
				Text:     chunkText,
				Metadata: metadata,
			}
			chunk.Metadata.PageNumber = fmt.Sprintf("%d", pageNum)
			chunks = append(chunks, chunk)
		}

		// Move start forward with overlap
		start = end - m.chunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

// generateChunkID creates a unique ID for a chunk
func (m *ManualIngester) generateChunkID(sourcePath string, pageNum, chunkIndex int) string {
	// Use hash of source path + page + chunk for deterministic IDs
	data := fmt.Sprintf("%s:%d:%d", sourcePath, pageNum, chunkIndex)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash
}

// IngestDirectory processes all PDFs in a directory
func (m *ManualIngester) IngestDirectory(ctx context.Context, dirPath string, defaultMetadata ManualMetadata) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.ToLower(filepath.Ext(path)) != ".pdf" {
			return nil
		}

		// Use filename as manual title if not set
		metadata := defaultMetadata
		if metadata.ManualTitle == "" {
			metadata.ManualTitle = strings.TrimSuffix(filepath.Base(path), ".pdf")
		}

		fmt.Printf("Ingesting: %s\n", path)
		if err := m.IngestManual(ctx, path, metadata); err != nil {
			fmt.Printf("  Error: %v\n", err)
			return nil // Continue with other files
		}
		fmt.Printf("  ✓ Complete\n")

		return nil
	})
}

// ExtractPDFMetadata attempts to extract metadata from PDF properties
func ExtractPDFMetadata(pdfPath string) (ManualMetadata, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return ManualMetadata{}, err
	}
	defer f.Close()

	metadata := ManualMetadata{
		SourcePath:  pdfPath,
		ManualTitle: filepath.Base(pdfPath),
	}

	// Try to extract title from PDF metadata
	if r.Trailer() != nil {
		if info := r.Trailer().Key("Info"); info != nil {
			if title := info.Key("Title"); title != nil {
				metadata.ManualTitle = title.String()
			}
		}
	}

	return metadata, nil
}

// VerifyPDF checks if a PDF file is readable
func VerifyPDF(pdfPath string) error {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if r.NumPage() == 0 {
		return fmt.Errorf("PDF has no pages")
	}

	return nil
}

// GetPDFInfo returns basic information about a PDF
func GetPDFInfo(pdfPath string) (pages int, size int64, err error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	info, err := os.Stat(pdfPath)
	if err != nil {
		return 0, 0, err
	}

	return r.NumPage(), info.Size(), nil
}

// ReadPDFPage extracts text from a specific page
func ReadPDFPage(pdfPath string, pageNum int) (string, error) {
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if pageNum < 1 || pageNum > r.NumPage() {
		return "", fmt.Errorf("page %d out of range (1-%d)", pageNum, r.NumPage())
	}

	page := r.Page(pageNum)
	if page.V.IsNull() {
		return "", fmt.Errorf("page %d is null", pageNum)
	}

	return page.GetPlainText(nil)
}
