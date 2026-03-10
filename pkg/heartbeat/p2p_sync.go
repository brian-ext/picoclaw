package heartbeat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/session"
)

// P2PSyncConfig configures automatic P2P sync via heartbeat
type P2PSyncConfig struct {
	Enabled         bool   `json:"enabled"`
	SyncInterval    int    `json:"sync_interval"`    // seconds
	MinValidations  int    `json:"min_validations"`
	ExportPath      string `json:"export_path"`
	ImportPath      string `json:"import_path"`
	AutoExport      bool   `json:"auto_export"`
	AutoImport      bool   `json:"auto_import"`
}

// P2PSyncTask handles automatic P2P knowledge sharing
type P2PSyncTask struct {
	config      P2PSyncConfig
	syncManager *session.P2PSyncManager
	db          *session.Database
	lastSync    time.Time
}

// NewP2PSyncTask creates a new P2P sync task for heartbeat
func NewP2PSyncTask(config P2PSyncConfig, workspacePath string) (*P2PSyncTask, error) {
	if !config.Enabled {
		return nil, nil
	}

	// Open database
	db, err := session.NewDatabase(workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create sync manager
	syncManager, err := session.NewP2PSyncManager(db, true, config.MinValidations)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create sync manager: %w", err)
	}

	// Set default paths if not specified
	if config.ExportPath == "" {
		config.ExportPath = filepath.Join(workspacePath, "p2p_export.json")
	}
	if config.ImportPath == "" {
		config.ImportPath = filepath.Join(workspacePath, "p2p_import.json")
	}

	return &P2PSyncTask{
		config:      config,
		syncManager: syncManager,
		db:          db,
		lastSync:    time.Now(),
	}, nil
}

// Run executes the P2P sync task
func (t *P2PSyncTask) Run(ctx context.Context) error {
	// Check if enough time has passed since last sync
	if time.Since(t.lastSync) < time.Duration(t.config.SyncInterval)*time.Second {
		return nil
	}

	logger.InfoCF("p2p_sync", "Starting automatic P2P sync", nil)

	// Auto-export if enabled
	if t.config.AutoExport {
		if err := t.autoExport(); err != nil {
			logger.ErrorCF("p2p_sync", "Auto-export failed", map[string]any{
				"error": err.Error(),
			})
		}
	}

	// Auto-import if enabled
	if t.config.AutoImport {
		if err := t.autoImport(); err != nil {
			logger.ErrorCF("p2p_sync", "Auto-import failed", map[string]any{
				"error": err.Error(),
			})
		}
	}

	t.lastSync = time.Now()
	return nil
}

// autoExport exports validated hacks automatically
func (t *P2PSyncTask) autoExport() error {
	hacks, err := t.syncManager.ExportValidatedHacks()
	if err != nil {
		return fmt.Errorf("failed to export hacks: %w", err)
	}

	if len(hacks) == 0 {
		logger.DebugCF("p2p_sync", "No new hacks to export", nil)
		return nil
	}

	// Write to export file
	data, err := json.MarshalIndent(hacks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hacks: %w", err)
	}

	if err := os.WriteFile(t.config.ExportPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	logger.InfoCF("p2p_sync", "Auto-exported validated hacks", map[string]any{
		"count": len(hacks),
		"path":  t.config.ExportPath,
	})

	// Log sync
	t.syncManager.LogSync("auto_export", len(hacks), 0, "")

	return nil
}

// autoImport imports hacks from import file
func (t *P2PSyncTask) autoImport() error {
	// Check if import file exists
	if _, err := os.Stat(t.config.ImportPath); os.IsNotExist(err) {
		logger.DebugCF("p2p_sync", "No import file found", map[string]any{
			"path": t.config.ImportPath,
		})
		return nil
	}

	// Read import file
	data, err := os.ReadFile(t.config.ImportPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse hacks
	var hacks []*session.SyncableHack
	if err := json.Unmarshal(data, &hacks); err != nil {
		return fmt.Errorf("failed to parse hacks: %w", err)
	}

	if len(hacks) == 0 {
		logger.DebugCF("p2p_sync", "No hacks in import file", nil)
		return nil
	}

	// Import hacks
	imported, rejected, err := t.syncManager.ImportHacks(hacks)
	if err != nil {
		return fmt.Errorf("failed to import hacks: %w", err)
	}

	logger.InfoCF("p2p_sync", "Auto-imported hacks", map[string]any{
		"imported": imported,
		"rejected": rejected,
		"path":     t.config.ImportPath,
	})

	// Log sync
	var errMsg string
	if rejected > 0 {
		errMsg = fmt.Sprintf("%d hacks rejected", rejected)
	}
	t.syncManager.LogSync("auto_import", 0, imported, errMsg)

	// Archive the import file (rename with timestamp)
	archivePath := t.config.ImportPath + "." + time.Now().Format("20060102_150405") + ".processed"
	os.Rename(t.config.ImportPath, archivePath)

	return nil
}

// Close cleans up resources
func (t *P2PSyncTask) Close() error {
	if t.db != nil {
		return t.db.Close()
	}
	return nil
}
