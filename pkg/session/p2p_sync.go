package session

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// P2PSyncManager handles automatic P2P knowledge sharing
type P2PSyncManager struct {
	db              *Database
	garageID        string
	privateKey      ed25519.PrivateKey
	publicKey       ed25519.PublicKey
	autoSyncEnabled bool
	minValidations  int
}

// SyncableHack represents a hack ready for P2P sharing
type SyncableHack struct {
	HackID          string                 `json:"hack_id"`
	Version         int                    `json:"version"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Vehicle         VehicleInfo            `json:"vehicle"`
	Repair          RepairInfo             `json:"repair"`
	Validation      ValidationInfo         `json:"validation"`
	Metadata        map[string]interface{} `json:"metadata"`
	Signature       SignatureInfo          `json:"signature"`
}

// VehicleInfo contains anonymized vehicle information
type VehicleInfo struct {
	Make           string `json:"make"`
	Model          string `json:"model"`
	Year           int    `json:"year"`
	VINPattern     string `json:"vin_pattern"` // Anonymized: "1G4HP54K*"
	BuildDateRange string `json:"build_date_range,omitempty"`
}

// RepairInfo contains the repair technique details
type RepairInfo struct {
	Component     string   `json:"component"`
	Issue         string   `json:"issue"`
	Solution      string   `json:"solution"`
	ToolsRequired []string `json:"tools_required,omitempty"`
	TimeSaved     string   `json:"time_saved,omitempty"`
	Difficulty    string   `json:"difficulty,omitempty"`
}

// ValidationInfo tracks validation across garages
type ValidationInfo struct {
	Count                int      `json:"count"`
	Garages              []string `json:"garages"`
	Confidence           string   `json:"confidence"`
	VerifiedByLibrarian  bool     `json:"verified_by_librarian"`
	ManualCitation       string   `json:"manual_citation,omitempty"`
}

// SignatureInfo contains cryptographic verification
type SignatureInfo struct {
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

// NewP2PSyncManager creates a new P2P sync manager
func NewP2PSyncManager(db *Database, autoSync bool, minValidations int) (*P2PSyncManager, error) {
	manager := &P2PSyncManager{
		db:              db,
		autoSyncEnabled: autoSync,
		minValidations:  minValidations,
	}

	// Load or create garage identity
	if err := manager.ensureGarageIdentity(); err != nil {
		return nil, fmt.Errorf("failed to ensure garage identity: %w", err)
	}

	return manager, nil
}

// ensureGarageIdentity loads or creates the garage's cryptographic identity
func (m *P2PSyncManager) ensureGarageIdentity() error {
	// Try to load existing identity
	var garageID, privateKeyB64, publicKeyB64 string
	err := m.db.db.QueryRow(`
		SELECT garage_id, private_key, public_key 
		FROM garage_identity 
		WHERE id = 1
	`).Scan(&garageID, &privateKeyB64, &publicKeyB64)

	if err == nil {
		// Identity exists, decode keys
		m.garageID = garageID
		
		privKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
		if err != nil {
			return fmt.Errorf("failed to decode private key: %w", err)
		}
		m.privateKey = ed25519.PrivateKey(privKeyBytes)
		
		pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
		if err != nil {
			return fmt.Errorf("failed to decode public key: %w", err)
		}
		m.publicKey = ed25519.PublicKey(pubKeyBytes)
		
		return nil
	}

	// Create new identity
	m.garageID = uuid.New().String()
	
	// Generate Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}
	
	m.privateKey = privKey
	m.publicKey = pubKey
	
	// Store in database
	_, err = m.db.db.Exec(`
		INSERT INTO garage_identity (id, garage_id, private_key, public_key)
		VALUES (1, ?, ?, ?)
	`, m.garageID, 
		base64.StdEncoding.EncodeToString(privKey),
		base64.StdEncoding.EncodeToString(pubKey))
	
	if err != nil {
		return fmt.Errorf("failed to store garage identity: %w", err)
	}

	return nil
}

// ExportValidatedHacks exports hacks ready for P2P sharing
func (m *P2PSyncManager) ExportValidatedHacks() ([]*SyncableHack, error) {
	// Query validated hacks that meet minimum validation threshold
	rows, err := m.db.db.Query(`
		SELECT h.hack_id, h.vin, h.description, h.validation_count, h.created_at,
		       ma.make, ma.model, ma.year, ma.build_date
		FROM validated_hacks h
		JOIN machines ma ON h.vin = ma.vin
		WHERE h.validation_count >= ? 
		  AND h.shared_to_network = 1
		  AND h.hack_id NOT IN (SELECT hack_id FROM synced_hacks WHERE source_garage_id = ?)
	`, m.minValidations, m.garageID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to query validated hacks: %w", err)
	}
	defer rows.Close()

	var hacks []*SyncableHack
	for rows.Next() {
		var hackID, vin, description, make, model, buildDate string
		var validationCount, year int
		var createdAt time.Time
		
		err := rows.Scan(&hackID, &vin, &description, &validationCount, &createdAt,
			&make, &model, &year, &buildDate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hack: %w", err)
		}

		// Create syncable hack with anonymized VIN
		hack := &SyncableHack{
			HackID:    hackID,
			Version:   1,
			CreatedAt: createdAt,
			UpdatedAt: time.Now(),
			Vehicle: VehicleInfo{
				Make:       make,
				Model:      model,
				Year:       year,
				VINPattern: anonymizeVIN(vin),
				BuildDateRange: buildDate,
			},
			Repair: RepairInfo{
				Solution: description,
			},
			Validation: ValidationInfo{
				Count:      validationCount,
				Garages:    []string{m.garageID},
				Confidence: getConfidenceLevel(validationCount),
			},
			Metadata: map[string]interface{}{
				"language": "en",
				"region":   "US",
			},
		}

		// Sign the hack
		if err := m.signHack(hack); err != nil {
			return nil, fmt.Errorf("failed to sign hack %s: %w", hackID, err)
		}

		hacks = append(hacks, hack)
	}

	return hacks, rows.Err()
}

// signHack creates a cryptographic signature for a hack
func (m *P2PSyncManager) signHack(hack *SyncableHack) error {
	// Serialize hack data (excluding signature field)
	hackData := struct {
		HackID     string         `json:"hack_id"`
		Version    int            `json:"version"`
		Vehicle    VehicleInfo    `json:"vehicle"`
		Repair     RepairInfo     `json:"repair"`
		Validation ValidationInfo `json:"validation"`
	}{
		HackID:     hack.HackID,
		Version:    hack.Version,
		Vehicle:    hack.Vehicle,
		Repair:     hack.Repair,
		Validation: hack.Validation,
	}

	dataBytes, err := json.Marshal(hackData)
	if err != nil {
		return fmt.Errorf("failed to marshal hack data: %w", err)
	}

	// Sign with private key
	signature := ed25519.Sign(m.privateKey, dataBytes)

	// Store signature info
	hack.Signature = SignatureInfo{
		Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(m.publicKey),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}

	return nil
}

// ImportHacks imports and verifies hacks from other garages
func (m *P2PSyncManager) ImportHacks(hacks []*SyncableHack) (int, int, error) {
	imported := 0
	rejected := 0

	for _, hack := range hacks {
		// Verify signature
		if !m.verifyHackSignature(hack) {
			rejected++
			continue
		}

		// Check if already imported
		var exists int
		err := m.db.db.QueryRow(`
			SELECT COUNT(*) FROM synced_hacks WHERE hack_id = ?
		`, hack.HackID).Scan(&exists)
		
		if err != nil || exists > 0 {
			continue
		}

		// Store in synced_hacks table
		hackDataJSON, _ := json.Marshal(hack)
		_, err = m.db.db.Exec(`
			INSERT INTO synced_hacks (hack_id, source_garage_id, signature, verified, hack_data_json)
			VALUES (?, ?, ?, 1, ?)
		`, hack.HackID, extractGarageID(hack.Signature.PublicKey), hack.Signature.Signature, string(hackDataJSON))

		if err != nil {
			rejected++
			continue
		}

		// Update peer garage reputation
		m.updatePeerReputation(extractGarageID(hack.Signature.PublicKey), true)

		imported++
	}

	return imported, rejected, nil
}

// verifyHackSignature verifies the cryptographic signature of a hack
func (m *P2PSyncManager) verifyHackSignature(hack *SyncableHack) bool {
	// Decode public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(hack.Signature.PublicKey)
	if err != nil {
		return false
	}
	pubKey := ed25519.PublicKey(pubKeyBytes)

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(hack.Signature.Signature)
	if err != nil {
		return false
	}

	// Reconstruct signed data
	hackData := struct {
		HackID     string         `json:"hack_id"`
		Version    int            `json:"version"`
		Vehicle    VehicleInfo    `json:"vehicle"`
		Repair     RepairInfo     `json:"repair"`
		Validation ValidationInfo `json:"validation"`
	}{
		HackID:     hack.HackID,
		Version:    hack.Version,
		Vehicle:    hack.Vehicle,
		Repair:     hack.Repair,
		Validation: hack.Validation,
	}

	dataBytes, err := json.Marshal(hackData)
	if err != nil {
		return false
	}

	// Verify signature
	return ed25519.Verify(pubKey, dataBytes, sigBytes)
}

// updatePeerReputation updates the reputation score for a peer garage
func (m *P2PSyncManager) updatePeerReputation(garageID string, verified bool) {
	if verified {
		m.db.db.Exec(`
			INSERT INTO peer_garages (garage_id, reputation_score, total_contributions, verified_contributions, last_seen)
			VALUES (?, 1, 1, 1, ?)
			ON CONFLICT(garage_id) DO UPDATE SET
				reputation_score = reputation_score + 1,
				total_contributions = total_contributions + 1,
				verified_contributions = verified_contributions + 1,
				last_seen = ?
		`, garageID, time.Now(), time.Now())
	} else {
		m.db.db.Exec(`
			INSERT INTO peer_garages (garage_id, reputation_score, total_contributions, rejected_contributions, last_seen)
			VALUES (?, -1, 1, 1, ?)
			ON CONFLICT(garage_id) DO UPDATE SET
				reputation_score = reputation_score - 1,
				total_contributions = total_contributions + 1,
				rejected_contributions = rejected_contributions + 1,
				last_seen = ?
		`, garageID, time.Now(), time.Now())
	}
}

// LogSync records a sync operation
func (m *P2PSyncManager) LogSync(syncType string, exported, imported int, errors string) error {
	syncID := uuid.New().String()
	_, err := m.db.db.Exec(`
		INSERT INTO sync_log (sync_id, sync_type, completed_at, hacks_exported, hacks_imported, errors)
		VALUES (?, ?, ?, ?, ?, ?)
	`, syncID, syncType, time.Now(), exported, imported, errors)
	
	return err
}

// GetSyncedHacks retrieves hacks imported from other garages
func (m *P2PSyncManager) GetSyncedHacks(minReputation int) ([]*SyncableHack, error) {
	rows, err := m.db.db.Query(`
		SELECT sh.hack_data_json
		FROM synced_hacks sh
		JOIN peer_garages pg ON sh.source_garage_id = pg.garage_id
		WHERE sh.verified = 1 AND pg.reputation_score >= ?
		ORDER BY sh.synced_at DESC
	`, minReputation)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hacks []*SyncableHack
	for rows.Next() {
		var hackJSON string
		if err := rows.Scan(&hackJSON); err != nil {
			continue
		}

		hack := &SyncableHack{}
		if err := json.Unmarshal([]byte(hackJSON), hack); err != nil {
			continue
		}

		hacks = append(hacks, hack)
	}

	return hacks, rows.Err()
}

// Helper functions

// anonymizeVIN replaces the last 8 digits with wildcards for privacy
func anonymizeVIN(vin string) string {
	if len(vin) < 9 {
		return vin
	}
	return vin[:9] + "*"
}

// getConfidenceLevel returns confidence based on validation count
func getConfidenceLevel(count int) string {
	if count >= 10 {
		return "very_high"
	} else if count >= 5 {
		return "high"
	} else if count >= 3 {
		return "medium"
	}
	return "low"
}

// extractGarageID creates a deterministic ID from public key
func extractGarageID(publicKeyB64 string) string {
	// In production, this would hash the public key
	// For now, use first 8 chars as identifier
	if len(publicKeyB64) < 8 {
		return publicKeyB64
	}
	return publicKeyB64[:8]
}
