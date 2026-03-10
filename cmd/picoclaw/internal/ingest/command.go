package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/vectordb"
	"github.com/spf13/cobra"
)

// Command creates the ingest command for manual ingestion
func Command() *cobra.Command {
	var (
		workspacePath string
		make          string
		model         string
		year          int
		manualType    string
		vin           string
	)

	cmd := &cobra.Command{
		Use:   "ingest [pdf-file-or-directory]",
		Short: "Ingest repair manuals into the vector database",
		Long: `Ingest repair manual PDFs into the ChromaDB vector database for semantic search.

Examples:
  # Ingest a single manual
  picoclaw ingest --make Honda --model Civic --year 2015 --type service manual.pdf

  # Ingest all PDFs in a directory
  picoclaw ingest --make Honda --model Civic --year 2015 --type service ./manuals/

  # Ingest VIN-specific manual
  picoclaw ingest --vin 1G4HP54K9XH123456 --type service manual.pdf`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIngest(args[0], workspacePath, make, model, year, manualType, vin)
		},
	}

	homeDir, _ := os.UserHomeDir()
	defaultWorkspace := filepath.Join(homeDir, ".picoclaw", "workspace")

	cmd.Flags().StringVar(&workspacePath, "workspace", defaultWorkspace, "Workspace directory path")
	cmd.Flags().StringVar(&make, "make", "", "Vehicle/equipment manufacturer (required)")
	cmd.Flags().StringVar(&model, "model", "", "Model name (required)")
	cmd.Flags().IntVar(&year, "year", 0, "Model year (required)")
	cmd.Flags().StringVar(&manualType, "type", "service", "Manual type (service, owner, tsb, wiring)")
	cmd.Flags().StringVar(&vin, "vin", "", "VIN for vehicle-specific manuals (optional)")

	cmd.MarkFlagRequired("make")
	cmd.MarkFlagRequired("model")
	cmd.MarkFlagRequired("year")

	return cmd
}

func runIngest(path, workspacePath, make, model string, year int, manualType, vin string) error {
	ctx := context.Background()

	// Initialize ChromaDB
	fmt.Println("Initializing ChromaDB...")
	db, err := vectordb.NewChromaDB(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to initialize ChromaDB: %w", err)
	}
	defer db.Close()

	// Create ingester
	ingester := vectordb.NewManualIngester(db)

	// Prepare metadata
	metadata := vectordb.ManualMetadata{
		VIN:        vin,
		Make:       make,
		Model:      model,
		Year:       year,
		ManualType: manualType,
	}

	// Check if path is file or directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		// Ingest directory
		fmt.Printf("Ingesting PDFs from directory: %s\n", path)
		if err := ingester.IngestDirectory(ctx, path, metadata); err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}
	} else {
		// Ingest single file
		if filepath.Ext(path) != ".pdf" {
			return fmt.Errorf("file must be a PDF")
		}

		// Verify PDF
		fmt.Printf("Verifying PDF: %s\n", path)
		if err := vectordb.VerifyPDF(path); err != nil {
			return fmt.Errorf("PDF verification failed: %w", err)
		}

		pages, size, _ := vectordb.GetPDFInfo(path)
		fmt.Printf("  Pages: %d, Size: %.2f MB\n", pages, float64(size)/(1024*1024))

		// Set manual title from filename if not set
		if metadata.ManualTitle == "" {
			metadata.ManualTitle = filepath.Base(path)
		}

		fmt.Printf("Ingesting manual...\n")
		if err := ingester.IngestManual(ctx, path, metadata); err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}
		fmt.Println("✓ Ingestion complete")
	}

	// Show collection stats
	count, _ := db.GetCollectionCount(ctx)
	fmt.Printf("\nTotal chunks in database: %d\n", count)

	return nil
}
