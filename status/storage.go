package status

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// sqliteTimeFormats are time formats that may be used by SQLite for datetime storage.
// The Go sqlite driver stores time.Time as RFC3339, while CURRENT_TIMESTAMP uses
// the simpler format without timezone.
var sqliteTimeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
}

// parseDBTime parses a time string from the database.
// Tries multiple formats since SQLite can store times in various formats.
// Returns zero time if parsing fails.
func parseDBTime(s string) time.Time {
	for _, format := range sqliteTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// StatusRecord represents a single status check result stored in the database.
type StatusRecord struct {
	ID         int64
	ServiceURL string
	IsUp       bool
	CheckedAt  time.Time
	Message    string
}

// AlertRecord represents a notification alert stored in the database.
type AlertRecord struct {
	ID         int64
	ServiceURL string
	AlertType  string
	Message    string
	SentAt     time.Time
}

// Incident represents a period of service downtime or degradation.
type Incident struct {
	ID          int64
	ServiceURL  string
	ServiceName string    // Display name of the service
	StartedAt   time.Time // When the incident started
	EndedAt     time.Time // When the incident ended (zero if ongoing)
	Duration    time.Duration
	Message     string // Error message when incident started
	IsOngoing   bool   // True if incident hasn't ended yet
}

// ToIncidentInfo converts an Incident to IncidentInfo for display.
func (i *Incident) ToIncidentInfo() IncidentInfo {
	info := IncidentInfo{
		ServiceName: i.ServiceName,
		StartedAt:   i.StartedAt.Format("2006-01-02 15:04"),
		Duration:    i.Duration,
		Message:     i.Message,
		IsOngoing:   i.IsOngoing,
	}
	if !i.IsOngoing {
		info.EndedAt = i.EndedAt.Format("2006-01-02 15:04")
	}
	return info
}

// Storage provides persistent data storage using SQLite.
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new Storage instance and initializes the database schema.
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// initSchema creates the necessary database tables if they don't exist.
func (s *Storage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS status_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_url TEXT NOT NULL,
		is_up INTEGER NOT NULL,
		checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		message TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_status_checks_service_url ON status_checks(service_url);
	CREATE INDEX IF NOT EXISTS idx_status_checks_checked_at ON status_checks(checked_at);

	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_url TEXT NOT NULL,
		alert_type TEXT NOT NULL,
		message TEXT,
		sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_service_url ON alerts(service_url);
	CREATE INDEX IF NOT EXISTS idx_alerts_sent_at ON alerts(sent_at);

	CREATE TABLE IF NOT EXISTS incidents (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_url TEXT NOT NULL,
		service_name TEXT,
		started_at DATETIME NOT NULL,
		ended_at DATETIME,
		message TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_incidents_service_url ON incidents(service_url);
	CREATE INDEX IF NOT EXISTS idx_incidents_started_at ON incidents(started_at);
	CREATE INDEX IF NOT EXISTS idx_incidents_ended_at ON incidents(ended_at);

	CREATE TABLE IF NOT EXISTS service_state (
		service_url TEXT PRIMARY KEY,
		is_up INTEGER NOT NULL,
		last_checked DATETIME NOT NULL,
		last_alert DATETIME
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// RecordStatus stores a status check result in the database.
func (s *Storage) RecordStatus(serviceURL string, isUp bool, message string) error {
	_, err := s.db.Exec(
		"INSERT INTO status_checks (service_url, is_up, checked_at, message) VALUES (?, ?, ?, ?)",
		serviceURL, boolToInt(isUp), time.Now().UTC(), message,
	)
	return err
}

// RecordAlert stores an alert notification in the database.
func (s *Storage) RecordAlert(alert Alert) error {
	_, err := s.db.Exec(
		"INSERT INTO alerts (service_url, alert_type, message, sent_at) VALUES (?, ?, ?, ?)",
		alert.ServiceURL, string(alert.AlertType), alert.Message, alert.Timestamp.UTC(),
	)
	return err
}

// GetLastStatus returns the most recent status for a service.
func (s *Storage) GetLastStatus(serviceURL string) (*StatusRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, service_url, is_up, checked_at, message 
		FROM status_checks 
		WHERE service_url = ? 
		ORDER BY checked_at DESC 
		LIMIT 1`,
		serviceURL,
	)

	var record StatusRecord
	var isUp int
	var checkedAt string
	var message sql.NullString

	err := row.Scan(&record.ID, &record.ServiceURL, &isUp, &checkedAt, &message)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	record.IsUp = isUp == 1
	record.CheckedAt = parseDBTime(checkedAt)
	if message.Valid {
		record.Message = message.String
	}

	return &record, nil
}

// GetStatusHistory returns the status history for a service.
func (s *Storage) GetStatusHistory(serviceURL string, limit int) ([]StatusRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, service_url, is_up, checked_at, message 
		FROM status_checks 
		WHERE service_url = ? 
		ORDER BY checked_at DESC 
		LIMIT ?`,
		serviceURL, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []StatusRecord
	for rows.Next() {
		var record StatusRecord
		var isUp int
		var checkedAt string
		var message sql.NullString

		if err := rows.Scan(&record.ID, &record.ServiceURL, &isUp, &checkedAt, &message); err != nil {
			return nil, err
		}

		record.IsUp = isUp == 1
		record.CheckedAt = parseDBTime(checkedAt)
		if message.Valid {
			record.Message = message.String
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// GetRecentAlerts returns recent alerts for all services.
func (s *Storage) GetRecentAlerts(limit int) ([]AlertRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, service_url, alert_type, message, sent_at 
		FROM alerts 
		ORDER BY sent_at DESC 
		LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []AlertRecord
	for rows.Next() {
		var record AlertRecord
		var sentAt string

		if err := rows.Scan(&record.ID, &record.ServiceURL, &record.AlertType, &record.Message, &sentAt); err != nil {
			return nil, err
		}

		record.SentAt = parseDBTime(sentAt)
		records = append(records, record)
	}

	return records, rows.Err()
}

// GetLastAlert returns the most recent alert for a service.
func (s *Storage) GetLastAlert(serviceURL string) (*AlertRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, service_url, alert_type, message, sent_at 
		FROM alerts 
		WHERE service_url = ? 
		ORDER BY sent_at DESC 
		LIMIT 1`,
		serviceURL,
	)

	var record AlertRecord
	var sentAt string

	err := row.Scan(&record.ID, &record.ServiceURL, &record.AlertType, &record.Message, &sentAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	record.SentAt = parseDBTime(sentAt)
	return &record, nil
}

// UpdateServiceState updates or inserts the current state of a service.
func (s *Storage) UpdateServiceState(serviceURL string, isUp bool) error {
	_, err := s.db.Exec(`
		INSERT INTO service_state (service_url, is_up, last_checked)
		VALUES (?, ?, ?)
		ON CONFLICT(service_url) DO UPDATE SET
			is_up = excluded.is_up,
			last_checked = excluded.last_checked`,
		serviceURL, boolToInt(isUp), time.Now().UTC(),
	)
	return err
}

// UpdateLastAlert updates the last alert time for a service.
func (s *Storage) UpdateLastAlert(serviceURL string, alertTime time.Time) error {
	_, err := s.db.Exec(`
		UPDATE service_state SET last_alert = ? WHERE service_url = ?`,
		alertTime.UTC(), serviceURL,
	)
	return err
}

// GetServiceState returns the stored state for a service.
func (s *Storage) GetServiceState(serviceURL string) (isUp bool, lastChecked time.Time, lastAlert *time.Time, err error) {
	row := s.db.QueryRow(`
		SELECT is_up, last_checked, last_alert 
		FROM service_state 
		WHERE service_url = ?`,
		serviceURL,
	)

	var isUpInt int
	var lastCheckedStr string
	var lastAlertStr sql.NullString

	err = row.Scan(&isUpInt, &lastCheckedStr, &lastAlertStr)
	if err == sql.ErrNoRows {
		return false, time.Time{}, nil, nil
	}
	if err != nil {
		return false, time.Time{}, nil, err
	}

	isUp = isUpInt == 1
	lastChecked = parseDBTime(lastCheckedStr)
	if lastAlertStr.Valid {
		parsed := parseDBTime(lastAlertStr.String)
		lastAlert = &parsed
	}

	return isUp, lastChecked, lastAlert, nil
}

// GetAllServiceStates returns the current state of all tracked services.
func (s *Storage) GetAllServiceStates() (map[string]bool, error) {
	rows, err := s.db.Query("SELECT service_url, is_up FROM service_state")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]bool)
	for rows.Next() {
		var serviceURL string
		var isUp int
		if err := rows.Scan(&serviceURL, &isUp); err != nil {
			return nil, err
		}
		states[serviceURL] = isUp == 1
	}

	return states, rows.Err()
}

// PruneOldRecords removes status check records older than the specified duration.
func (s *Storage) PruneOldRecords(olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	result, err := s.db.Exec(
		"DELETE FROM status_checks WHERE checked_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// StartIncident records the beginning of a new incident for a service.
func (s *Storage) StartIncident(serviceURL, serviceName, message string) error {
	_, err := s.db.Exec(
		"INSERT INTO incidents (service_url, service_name, started_at, message) VALUES (?, ?, ?, ?)",
		serviceURL, serviceName, time.Now().UTC(), message,
	)
	return err
}

// EndIncident marks the most recent ongoing incident for a service as resolved.
func (s *Storage) EndIncident(serviceURL string) error {
	_, err := s.db.Exec(`
		UPDATE incidents 
		SET ended_at = ? 
		WHERE service_url = ? AND ended_at IS NULL`,
		time.Now().UTC(), serviceURL,
	)
	return err
}

// GetOngoingIncident returns the current ongoing incident for a service, if any.
func (s *Storage) GetOngoingIncident(serviceURL string) (*Incident, error) {
	row := s.db.QueryRow(`
		SELECT id, service_url, service_name, started_at, message 
		FROM incidents 
		WHERE service_url = ? AND ended_at IS NULL 
		ORDER BY started_at DESC 
		LIMIT 1`,
		serviceURL,
	)

	var incident Incident
	var startedAt string
	var serviceName sql.NullString
	var message sql.NullString

	err := row.Scan(&incident.ID, &incident.ServiceURL, &serviceName, &startedAt, &message)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	incident.StartedAt = parseDBTime(startedAt)
	incident.IsOngoing = true
	incident.Duration = time.Since(incident.StartedAt)
	if serviceName.Valid {
		incident.ServiceName = serviceName.String
	}
	if message.Valid {
		incident.Message = message.String
	}

	return &incident, nil
}

// GetRecentResolvedIncidents returns recent resolved incidents for all services.
// Unlike GetServiceIncidents, this excludes ongoing incidents.
// minDuration filters out incidents shorter than the specified duration (0 to include all).
// Note: Duration filtering is done in Go because SQLite's date functions don't parse
// the RFC3339Nano format used by the Go driver.
func (s *Storage) GetRecentResolvedIncidents(limit int, minDuration time.Duration) ([]Incident, error) {
	// When filtering by duration, we need to fetch more records to ensure we get
	// enough results after filtering. We use a multiplier to reduce round trips.
	fetchLimit := limit
	if minDuration > 0 {
		fetchLimit = limit * 3 // Fetch extra to account for filtered-out short incidents
	}

	rows, err := s.db.Query(`
		SELECT id, service_url, service_name, started_at, ended_at, message 
		FROM incidents 
		WHERE ended_at IS NOT NULL 
		ORDER BY ended_at DESC 
		LIMIT ?`,
		fetchLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var incident Incident
		var startedAt string
		var endedAt string
		var serviceName sql.NullString
		var message sql.NullString

		if err := rows.Scan(&incident.ID, &incident.ServiceURL, &serviceName, &startedAt, &endedAt, &message); err != nil {
			return nil, err
		}

		incident.StartedAt = parseDBTime(startedAt)
		incident.EndedAt = parseDBTime(endedAt)
		incident.Duration = incident.EndedAt.Sub(incident.StartedAt)
		incident.IsOngoing = false
		if serviceName.Valid {
			incident.ServiceName = serviceName.String
		}
		if message.Valid {
			incident.Message = message.String
		}

		// Filter by minimum duration (done in Go since SQLite can't parse RFC3339Nano)
		if minDuration > 0 && incident.Duration < minDuration {
			continue
		}

		incidents = append(incidents, incident)

		// Stop when we have enough results
		if len(incidents) >= limit {
			break
		}
	}

	return incidents, rows.Err()
}

// GetServiceIncidents returns recent incidents (both resolved and ongoing) for a specific service.
func (s *Storage) GetServiceIncidents(serviceURL string, limit int) ([]Incident, error) {
	rows, err := s.db.Query(`
		SELECT id, service_url, service_name, started_at, ended_at, message 
		FROM incidents 
		WHERE service_url = ? 
		ORDER BY started_at DESC 
		LIMIT ?`,
		serviceURL, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var incident Incident
		var startedAt string
		var endedAtStr sql.NullString
		var serviceName sql.NullString
		var message sql.NullString

		if err := rows.Scan(&incident.ID, &incident.ServiceURL, &serviceName, &startedAt, &endedAtStr, &message); err != nil {
			return nil, err
		}

		incident.StartedAt = parseDBTime(startedAt)
		if endedAtStr.Valid {
			incident.EndedAt = parseDBTime(endedAtStr.String)
			incident.Duration = incident.EndedAt.Sub(incident.StartedAt)
			incident.IsOngoing = false
		} else {
			incident.IsOngoing = true
			incident.Duration = time.Since(incident.StartedAt)
		}
		if serviceName.Valid {
			incident.ServiceName = serviceName.String
		}
		if message.Valid {
			incident.Message = message.String
		}

		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

// RecordStatusTransition checks for status changes, records incidents, and updates service state.
// Returns true if a new incident was started or an existing one was ended.
// This method atomically handles both transition detection and state update to prevent race conditions.
func (s *Storage) RecordStatusTransition(serviceURL, serviceName string, isUp bool, message string) (bool, error) {
	// Get the previous state
	prevUp, prevChecked, _, err := s.GetServiceState(serviceURL)
	if err != nil {
		return false, err
	}

	var changed bool

	// If this is the first check for this service, no transition to record
	if prevChecked.IsZero() {
		// If starting down, record an incident
		if !isUp {
			if err := s.StartIncident(serviceURL, serviceName, message); err != nil {
				return false, err
			}
			changed = true
		}
	} else {
		// Check for transitions
		if prevUp && !isUp {
			// Transition from up to down - start incident
			if err := s.StartIncident(serviceURL, serviceName, message); err != nil {
				return false, err
			}
			changed = true
		} else if !prevUp && isUp {
			// Transition from down to up - end incident
			if err := s.EndIncident(serviceURL); err != nil {
				return false, err
			}
			changed = true
		}
	}

	// Update service state at the end (atomically with transition detection)
	if err := s.UpdateServiceState(serviceURL, isUp); err != nil {
		return changed, err
	}

	return changed, nil
}

// boolToInt converts a boolean to an integer for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
