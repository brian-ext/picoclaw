package p2p

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/spf13/cobra"
)

// NewExportCommand creates the export command for P2P sync
func NewExportCommand() *cobra.Command {
	var (
		outputPath     string
		minValidations int
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export validated hacks for P2P sharing",
		Long: `Export validated repair hacks that meet the minimum validation threshold.
Exported hacks are cryptographically signed and ready for sharing with other garages.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExport(outputPath, minValidations)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "hacks_export.json", "Output file path")
	cmd.Flags().IntVarP(&minValidations, "min-validations", "m", 3, "Minimum validation count")

	return cmd
}

func runExport(outputPath string, minValidations int) error {
	// Get workspace path
	workspacePath := os.Getenv("PICOCLAW_WORKSPACE")
	if workspacePath == "" {
		home, _ := os.UserHomeDir()
		workspacePath = filepath.Join(home, ".picoclaw", "workspace")
	}

	// Open database
	db, err := session.NewDatabase(workspacePath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create sync manager
	syncManager, err := session.NewP2PSyncManager(db, false, minValidations)
	if err != nil {
		return fmt.Errorf("failed to create sync manager: %w", err)
	}

	// Export hacks
	hacks, err := syncManager.ExportValidatedHacks()
	if err != nil {
		return fmt.Errorf("failed to export hacks: %w", err)
	}

	if len(hacks) == 0 {
		fmt.Println("No validated hacks found meeting criteria")
		return nil
	}

	// Write to file
	data, err := json.MarshalIndent(hacks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hacks: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("✅ Exported %d validated hacks to %s\n", len(hacks), outputPath)
	fmt.Printf("📊 Minimum validation count: %d\n", minValidations)
	
	// Log sync
	syncManager.LogSync("export", len(hacks), 0, "")

	return nil
}
