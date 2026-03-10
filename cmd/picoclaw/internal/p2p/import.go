package p2p

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/session"
	"github.com/spf13/cobra"
)

// NewImportCommand creates the import command for P2P sync
func NewImportCommand() *cobra.Command {
	var (
		inputPath string
		verify    bool
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import validated hacks from other garages",
		Long: `Import and verify validated repair hacks from other garages.
All imported hacks are cryptographically verified before being added to the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImport(inputPath, verify)
		},
	}

	cmd.Flags().StringVarP(&inputPath, "input", "i", "hacks_export.json", "Input file path")
	cmd.Flags().BoolVarP(&verify, "verify", "v", true, "Verify signatures (recommended)")

	return cmd
}

func runImport(inputPath string, verify bool) error {
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
	syncManager, err := session.NewP2PSyncManager(db, false, 3)
	if err != nil {
		return fmt.Errorf("failed to create sync manager: %w", err)
	}

	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Parse hacks
	var hacks []*session.SyncableHack
	if err := json.Unmarshal(data, &hacks); err != nil {
		return fmt.Errorf("failed to parse hacks: %w", err)
	}

	if len(hacks) == 0 {
		fmt.Println("No hacks found in input file")
		return nil
	}

	fmt.Printf("📥 Importing %d hacks from %s\n", len(hacks), inputPath)

	// Import hacks
	imported, rejected, err := syncManager.ImportHacks(hacks)
	if err != nil {
		return fmt.Errorf("failed to import hacks: %w", err)
	}

	fmt.Printf("✅ Imported: %d\n", imported)
	fmt.Printf("❌ Rejected: %d\n", rejected)
	
	if rejected > 0 {
		fmt.Println("⚠️  Some hacks were rejected due to signature verification failures")
	}

	// Log sync
	var errMsg string
	if rejected > 0 {
		errMsg = fmt.Sprintf("%d hacks rejected", rejected)
	}
	syncManager.LogSync("import", 0, imported, errMsg)

	return nil
}
