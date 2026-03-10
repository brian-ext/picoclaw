package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database manages VIN-keyed repair sessions and machine profiles
type Database struct {
	db           *sql.DB
	workspacePath string
}

// Machine represents a vehicle or equipment being repaired
type Machine struct {
	VIN         string    `json:"vin"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Year        int       `json:"year"`
	BuildDate   string    `json:"build_date"`
	ProfilePath string    `json:"profile_path"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Session represents a repair session for a specific machine
type Session struct {
	SessionID           string    `json:"session_id"`
	VIN                 string    `json:"vin"`
	StartedAt           time.Time `json:"started_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	RepairType          string    `json:"repair_type"`
	ConversationSummary string    `json:"conversation_summary"`
	InsightsJSON        string    `json:"insights_json"`
}

// ValidatedHack represents a user-discovered repair technique
type ValidatedHack struct {
	HackID           string    `json:"hack_id"`
	VIN              string    `json:"vin"`
	Description      string    `json:"description"`
	ValidationCount  int       `json:"validation_count"`
	SharedToNetwork  bool      `json:"shared_to_network"`
	CreatedAt        time.Time `json:"created_at"`
}

// Citation represents a manual reference used during repair
type Citation struct {
	CitationID string    `json:"citation_id"`
	SessionID  string    `json:"session_id"`
	Source     string    `json:"source"`
	Page       string    `json:"page"`
	Snippet    string    `json:"snippet"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewDatabase creates a new session database
func NewDatabase(workspacePath string) (*Database, error) {
	dbPath := filepath.Join(workspacePath, "sessions.db")
	
	// Ensure workspace directory exists
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{
		db:           db,
		workspacePath: workspacePath,
	}

	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return d, nil
}

// initSchema creates the database tables
func (d *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS machines (
		vin TEXT PRIMARY KEY,
		make TEXT,
		model TEXT,
		year INTEGER,
		build_date TEXT,
		profile_path TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT PRIMARY KEY,
		vin TEXT,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		repair_type TEXT,
		conversation_summary TEXT,
		insights_json TEXT,
		FOREIGN KEY (vin) REFERENCES machines(vin)
	);

	CREATE TABLE IF NOT EXISTS validated_hacks (
		hack_id TEXT PRIMARY KEY,
		vin TEXT,
		description TEXT,
		validation_count INTEGER DEFAULT 1,
		shared_to_network BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (vin) REFERENCES machines(vin)
	);

	CREATE TABLE IF NOT EXISTS citations (
		citation_id TEXT PRIMARY KEY,
		session_id TEXT,
		source TEXT,
		page TEXT,
		snippet TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (session_id) REFERENCES sessions(session_id)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_vin ON sessions(vin);
	CREATE INDEX IF NOT EXISTS idx_citations_session ON citations(session_id);
	CREATE INDEX IF NOT EXISTS idx_hacks_vin ON validated_hacks(vin);
	`

	_, err := d.db.Exec(schema)
	return err
}

// CreateMachine registers a new machine in the database
func (d *Database) CreateMachine(m *Machine) error {
	// Create machine profile directory
	machineDir := filepath.Join(d.workspacePath, "machines", m.VIN)
	if err := os.MkdirAll(machineDir, 0755); err != nil {
		return fmt.Errorf("failed to create machine directory: %w", err)
	}

	m.ProfilePath = filepath.Join(machineDir, "MACHINE_PROFILE.md")
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	// Create initial machine profile
	if err := d.createMachineProfile(m); err != nil {
		return fmt.Errorf("failed to create machine profile: %w", err)
	}

	// Insert into database
	_, err := d.db.Exec(`
		INSERT INTO machines (vin, make, model, year, build_date, profile_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, m.VIN, m.Make, m.Model, m.Year, m.BuildDate, m.ProfilePath, m.CreatedAt, m.UpdatedAt)

	return err
}

// GetMachine retrieves a machine by VIN
func (d *Database) GetMachine(vin string) (*Machine, error) {
	m := &Machine{}
	err := d.db.QueryRow(`
		SELECT vin, make, model, year, build_date, profile_path, created_at, updated_at
		FROM machines WHERE vin = ?
	`, vin).Scan(&m.VIN, &m.Make, &m.Model, &m.Year, &m.BuildDate, &m.ProfilePath, &m.CreatedAt, &m.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return m, err
}

// CreateSession starts a new repair session
func (d *Database) CreateSession(s *Session) error {
	s.StartedAt = time.Now()
	
	_, err := d.db.Exec(`
		INSERT INTO sessions (session_id, vin, started_at, repair_type)
		VALUES (?, ?, ?, ?)
	`, s.SessionID, s.VIN, s.StartedAt, s.RepairType)

	return err
}

// UpdateSession updates session details
func (d *Database) UpdateSession(s *Session) error {
	_, err := d.db.Exec(`
		UPDATE sessions 
		SET completed_at = ?, conversation_summary = ?, insights_json = ?
		WHERE session_id = ?
	`, s.CompletedAt, s.ConversationSummary, s.InsightsJSON, s.SessionID)

	return err
}

// GetRecentSessions retrieves the N most recent sessions for a VIN
func (d *Database) GetRecentSessions(vin string, limit int) ([]*Session, error) {
	rows, err := d.db.Query(`
		SELECT session_id, vin, started_at, completed_at, repair_type, conversation_summary, insights_json
		FROM sessions
		WHERE vin = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, vin, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		s := &Session{}
		err := rows.Scan(&s.SessionID, &s.VIN, &s.StartedAt, &s.CompletedAt, &s.RepairType, &s.ConversationSummary, &s.InsightsJSON)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, rows.Err()
}

// AddCitation records a manual citation used during repair
func (d *Database) AddCitation(c *Citation) error {
	c.CreatedAt = time.Now()
	
	_, err := d.db.Exec(`
		INSERT INTO citations (citation_id, session_id, source, page, snippet, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, c.CitationID, c.SessionID, c.Source, c.Page, c.Snippet, c.CreatedAt)

	return err
}

// GetSessionCitations retrieves all citations for a session
func (d *Database) GetSessionCitations(sessionID string) ([]*Citation, error) {
	rows, err := d.db.Query(`
		SELECT citation_id, session_id, source, page, snippet, created_at
		FROM citations
		WHERE session_id = ?
		ORDER BY created_at
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var citations []*Citation
	for rows.Next() {
		c := &Citation{}
		err := rows.Scan(&c.CitationID, &c.SessionID, &c.Source, &c.Page, &c.Snippet, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		citations = append(citations, c)
	}

	return citations, rows.Err()
}

// AddValidatedHack records a user-discovered repair technique
func (d *Database) AddValidatedHack(h *ValidatedHack) error {
	h.CreatedAt = time.Now()
	
	_, err := d.db.Exec(`
		INSERT INTO validated_hacks (hack_id, vin, description, validation_count, shared_to_network, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, h.HackID, h.VIN, h.Description, h.ValidationCount, h.SharedToNetwork, h.CreatedAt)

	return err
}

// IncrementHackValidation increments the validation count for a hack
func (d *Database) IncrementHackValidation(hackID string) error {
	_, err := d.db.Exec(`
		UPDATE validated_hacks 
		SET validation_count = validation_count + 1
		WHERE hack_id = ?
	`, hackID)

	return err
}

// GetValidatedHacks retrieves validated hacks for a VIN
func (d *Database) GetValidatedHacks(vin string, minValidations int) ([]*ValidatedHack, error) {
	rows, err := d.db.Query(`
		SELECT hack_id, vin, description, validation_count, shared_to_network, created_at
		FROM validated_hacks
		WHERE vin = ? AND validation_count >= ?
		ORDER BY validation_count DESC
	`, vin, minValidations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hacks []*ValidatedHack
	for rows.Next() {
		h := &ValidatedHack{}
		err := rows.Scan(&h.HackID, &h.VIN, &h.Description, &h.ValidationCount, &h.SharedToNetwork, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		hacks = append(hacks, h)
	}

	return hacks, rows.Err()
}

// createMachineProfile generates the initial MACHINE_PROFILE.md file
func (d *Database) createMachineProfile(m *Machine) error {
	template := fmt.Sprintf(`# Machine Profile

## Identification
- VIN: %s
- Make: %s
- Model: %s
- Year: %d
- Build Date: %s

## Mid-Year Changes
(To be discovered during repairs)

## Confirmed Variations
(To be discovered during repairs)

## Repair History
(Automatically updated from sessions)

## Validated Hacks
(User-discovered techniques validated by multiple repairs)

---
*This profile is automatically updated as repairs are completed.*
`, m.VIN, m.Make, m.Model, m.Year, m.BuildDate)

	return os.WriteFile(m.ProfilePath, []byte(template), 0644)
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// SessionInsights represents structured insights from a repair session
type SessionInsights struct {
	LearnedFacts      []string `json:"learned_facts"`
	MidYearChanges    []string `json:"mid_year_changes"`
	SafetyChecks      []string `json:"safety_checks"`
	ToolsRequired     []string `json:"tools_required"`
	UnexpectedIssues  []string `json:"unexpected_issues"`
	ValidatedHacks    []string `json:"validated_hacks"`
}

// MarshalInsights converts insights to JSON for storage
func MarshalInsights(insights *SessionInsights) (string, error) {
	data, err := json.Marshal(insights)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalInsights converts JSON back to insights struct
func UnmarshalInsights(data string) (*SessionInsights, error) {
	insights := &SessionInsights{}
	err := json.Unmarshal([]byte(data), insights)
	return insights, err
}
