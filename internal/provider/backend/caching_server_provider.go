package backend

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"

	"github.com/ja-he/dayplan/internal/model"
	"github.com/ja-he/dayplan/internal/provider"
)

// CachingServerClientConfig holds configuration for creating a CachingServerClientDataProvider.
type CachingServerClientConfig struct {
	DBPath    string
	ServerURL string
}

// SyncStatus represents the current sync state.
type SyncStatus struct {
	Online         bool
	Syncing        bool
	PendingChanges int
	ConflictCount  int
	LastSyncTime   time.Time
	LastError      error
}

// Conflict represents a conflict between local and server versions.
type Conflict struct {
	RecordType    string // "event" | "category"
	RecordID      string
	LocalVersion  any // *model.Event or *model.Category
	ServerVersion any
	DetectedAt    time.Time
}

// ConflictResolution specifies how to resolve a conflict.
type ConflictResolution struct {
	UseLocal  bool
	UseServer bool
	Merged    any // if neither UseLocal nor UseServer
}

// SyncProvider extends EventProvider with sync capabilities.
type SyncProvider interface {
	provider.EventProvider

	// SyncStatus returns current sync state (online, pending count, conflicts).
	SyncStatus() SyncStatus

	// TriggerSync initiates sync. Non-blocking.
	// Returns channel that closes when sync completes (or fails).
	TriggerSync() <-chan error

	// Conflicts returns unresolved conflicts.
	Conflicts() ([]Conflict, error)

	// ResolveConflict resolves a conflict.
	ResolveConflict(recordType, recordID string, resolution ConflictResolution) error

	// WatchStatus returns a channel that emits on status changes.
	WatchStatus() <-chan SyncStatus

	// Login authenticates with the server.
	Login(username, password string) error

	// Logout invalidates the current token.
	Logout() error
}

// pendingChange represents a change not yet pushed to the server.
type pendingChange struct {
	RecordType string // "event" | "category"
	RecordID   string
	Operation  string // "create" | "update" | "delete"
	ChangedAt  time.Time
}

// serverEvent represents an event as returned by the server.
type serverEvent struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	Deleted         bool   `json:"deleted"`
	UpdatedAt       string `json:"updated_at"`
	ClientUpdatedAt string `json:"client_updated_at,omitempty"` // Sent when pushing to indicate base version
}

// serverCategory represents a category as returned by the server.
type serverCategory struct {
	Name      string `json:"name"`
	Color     string `json:"color"`
	Goal      string `json:"goal"` // JSON
	Priority  int    `json:"priority"`
	Deleted   bool   `json:"deleted"`
	UpdatedAt string `json:"updated_at"`
}

// syncPushRequest is the request body for POST /api/v1/sync.
type syncPushRequest struct {
	Events     []serverEvent    `json:"events"`
	Categories []serverCategory `json:"categories"`
}

// syncPushResponse is the response from POST /api/v1/sync.
type syncPushResponse struct {
	Events     []serverEvent    `json:"events"`
	Categories []serverCategory `json:"categories"`
	Conflicts  []serverConflict `json:"conflicts"`
	ServerTime string           `json:"server_time"`
}

// syncPullResponse is the response from GET /api/v1/sync.
type syncPullResponse struct {
	Events     []serverEvent    `json:"events"`
	Categories []serverCategory `json:"categories"`
	ServerTime string           `json:"server_time"`
}

// serverConflict represents a conflict returned by the server.
type serverConflict struct {
	RecordType    string `json:"record_type"`
	RecordID      string `json:"record_id"`
	ServerVersion any    `json:"server_version"`
}

// loginRequest is the request body for POST /api/v1/auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginResponse is the response from POST /api/v1/auth/login.
type loginResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
}

// CachingServerClientDataProvider implements EventProvider with local SQLite storage
// and background sync to a remote server.
type CachingServerClientDataProvider struct {
	db        *sql.DB
	serverURL string
	token     string

	// Sync coordination
	syncMu          sync.Mutex
	syncNotifyCh    chan struct{}
	syncCtx         context.Context
	syncCancel      context.CancelFunc
	statusMu        sync.RWMutex
	status          SyncStatus
	statusWatchers  []chan SyncStatus
	statusWatcherMu sync.Mutex

	// HTTP client
	httpClient *http.Client

	// Category provider for SumUpTimespanByCategory
	categoryProvider provider.CategoryProvider

	log zerolog.Logger
}

// Ensure CachingServerClientDataProvider implements the required interfaces
var _ provider.EventProvider = (*CachingServerClientDataProvider)(nil)
var _ SyncProvider = (*CachingServerClientDataProvider)(nil)

// NewCachingServerClientDataProvider creates a new CachingServerClientDataProvider.
func NewCachingServerClientDataProvider(
	cfg CachingServerClientConfig,
	categoryProvider provider.CategoryProvider,
) (*CachingServerClientDataProvider, error) {
	// Open local SQLite
	// - WAL mode allows concurrent reads while writing
	// - busy_timeout waits up to 10s for locks before returning SQLITE_BUSY
	// - _txlock=immediate acquires write lock at BEGIN rather than first write (avoids upgrade deadlocks)
	db, err := sql.Open("sqlite", cfg.DBPath+"?_journal_mode=WAL&_busy_timeout=10000&_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Limit to a single connection to eliminate SQLITE_BUSY from connection pool contention.
	// Since this is a local single-user database, concurrent connections provide no benefit
	// and only create lock contention. A single connection serializes all access.
	db.SetMaxOpenConns(1)

	// Run migrations
	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Check server_url in sync_meta matches config
	storedURL, err := getMeta(db, "server_url")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get stored server URL: %w", err)
	}
	if storedURL != "" && storedURL != cfg.ServerURL {
		db.Close()
		return nil, fmt.Errorf("server URL mismatch: stored=%s, config=%s", storedURL, cfg.ServerURL)
	}
	if err := setMeta(db, "server_url", cfg.ServerURL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set server URL: %w", err)
	}

	// Load auth token from sync_meta
	token, _ := getMeta(db, "auth_token")

	ctx, cancel := context.WithCancel(context.Background())

	p := &CachingServerClientDataProvider{
		db:               db,
		serverURL:        cfg.ServerURL,
		token:            token,
		syncNotifyCh:     make(chan struct{}, 1),
		syncCtx:          ctx,
		syncCancel:       cancel,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		categoryProvider: categoryProvider,
		log:              log.Level(zerolog.WarnLevel).With().Str("component", "caching-server-provider").Logger(),
	}

	// Start background goroutines
	go p.runSyncLoop(ctx)
	go p.runSSE(ctx)

	return p, nil
}

// Close shuts down the provider.
func (p *CachingServerClientDataProvider) Close() error {
	p.syncCancel()
	return p.db.Close()
}

// runMigrations creates the necessary tables if they don't exist.
func runMigrations(db *sql.DB) error {
	schema := `
	-- Events (mirrors server)
	CREATE TABLE IF NOT EXISTS events (
		id                TEXT PRIMARY KEY,
		name              TEXT NOT NULL,
		category          TEXT,
		start_time        TEXT NOT NULL,
		end_time          TEXT NOT NULL,
		deleted           INTEGER NOT NULL DEFAULT 0,
		updated_at        TEXT NOT NULL,
		server_updated_at TEXT  -- Last known updated_at from server (for conflict detection)
	);

	-- Categories (mirrors server)
	CREATE TABLE IF NOT EXISTS categories (
		name       TEXT PRIMARY KEY,
		color      TEXT,
		goal       TEXT,
		priority   INTEGER NOT NULL DEFAULT 0,
		deleted    INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL
	);

	-- Pending changes not yet pushed
	CREATE TABLE IF NOT EXISTS pending_changes (
		record_type TEXT NOT NULL,
		record_id   TEXT NOT NULL,
		operation   TEXT NOT NULL,
		changed_at  TEXT NOT NULL,
		PRIMARY KEY (record_type, record_id)
	);

	-- Conflicts awaiting resolution
	CREATE TABLE IF NOT EXISTS conflicts (
		record_type    TEXT NOT NULL,
		record_id      TEXT NOT NULL,
		local_version  TEXT NOT NULL,
		server_version TEXT NOT NULL,
		detected_at    TEXT NOT NULL,
		PRIMARY KEY (record_type, record_id)
	);

	-- Sync metadata
	CREATE TABLE IF NOT EXISTS sync_meta (
		key   TEXT PRIMARY KEY,
		value TEXT
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_events_timerange ON events(start_time, end_time);
	CREATE INDEX IF NOT EXISTS idx_events_deleted ON events(deleted);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: Normalize time strings to RFC3339 without fractional seconds.
	// This fixes a bug where server-synced events had milliseconds (e.g., "2026-01-18T13:04:28.693Z")
	// but queries used RFC3339 format without milliseconds ("2026-01-18T13:04:28Z").
	// Lexicographic comparison differs because '.' < 'Z' in ASCII, causing navigation bugs.
	if err := migrateNormalizeTimeStrings(db); err != nil {
		return fmt.Errorf("failed to normalize time strings: %w", err)
	}

	return nil
}

// migrateNormalizeTimeStrings normalizes all event times to RFC3339 without fractional seconds.
func migrateNormalizeTimeStrings(db *sql.DB) error {
	// Check if migration already ran
	var migrated string
	err := db.QueryRow("SELECT value FROM sync_meta WHERE key = 'migration_normalize_times'").Scan(&migrated)
	if err == nil && migrated == "done" {
		return nil // Already migrated
	}

	log.Info().Msg("Running migration: normalizing event time strings to RFC3339 (removing fractional seconds)")

	// Get all events with potentially non-normalized times
	rows, err := db.Query("SELECT id, start_time, end_time, updated_at, server_updated_at FROM events")
	if err != nil {
		return err
	}
	defer rows.Close()

	type eventTime struct {
		id                                             string
		startTime, endTime, updatedAt, serverUpdatedAt string
	}
	var events []eventTime
	for rows.Next() {
		var e eventTime
		var serverUpdatedAt sql.NullString
		if err := rows.Scan(&e.id, &e.startTime, &e.endTime, &e.updatedAt, &serverUpdatedAt); err != nil {
			return err
		}
		e.serverUpdatedAt = serverUpdatedAt.String
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Update each event with normalized times
	updated := 0
	for _, e := range events {
		newStart := normalizeTimeString(e.startTime)
		newEnd := normalizeTimeString(e.endTime)
		newUpdatedAt := normalizeTimeString(e.updatedAt)
		newServerUpdatedAt := ""
		if e.serverUpdatedAt != "" {
			newServerUpdatedAt = normalizeTimeString(e.serverUpdatedAt)
		}

		// Only update if something changed
		if newStart != e.startTime || newEnd != e.endTime || newUpdatedAt != e.updatedAt || newServerUpdatedAt != e.serverUpdatedAt {
			var serverUpdatedAtVal any = nil
			if newServerUpdatedAt != "" {
				serverUpdatedAtVal = newServerUpdatedAt
			}
			_, err := db.Exec(
				"UPDATE events SET start_time = ?, end_time = ?, updated_at = ?, server_updated_at = ? WHERE id = ?",
				newStart, newEnd, newUpdatedAt, serverUpdatedAtVal, e.id,
			)
			if err != nil {
				return fmt.Errorf("failed to update event %s: %w", e.id, err)
			}
			updated++
		}
	}

	log.Info().Int("events_updated", updated).Int("events_total", len(events)).Msg("Migration complete: normalized event time strings")

	// Mark migration as done
	_, err = db.Exec("INSERT INTO sync_meta (key, value) VALUES ('migration_normalize_times', 'done') ON CONFLICT(key) DO UPDATE SET value = excluded.value")
	return err
}

// getMeta retrieves a value from sync_meta.
func getMeta(db *sql.DB, key string) (string, error) {
	var value sql.NullString
	err := db.QueryRow("SELECT value FROM sync_meta WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value.String, nil
}

// setMeta stores a value in sync_meta.
func setMeta(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		"INSERT INTO sync_meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	return err
}

// isSQLiteBusy checks if an error is a SQLite busy error.
func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "SQLITE_BUSY") ||
		strings.Contains(errStr, "database is locked") ||
		strings.Contains(errStr, "database table is locked")
}

// beginTxWithRetry attempts to begin a transaction with retry on SQLITE_BUSY.
// This provides defense-in-depth against transient busy conditions.
func (p *CachingServerClientDataProvider) beginTxWithRetry() (*sql.Tx, error) {
	const maxRetries = 5
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		tx, err := p.db.Begin()
		if err == nil {
			return tx, nil
		}

		if !isSQLiteBusy(err) {
			return nil, err
		}

		lastErr = err
		// Exponential backoff: 10ms, 20ms, 40ms, 80ms, 160ms
		backoff := time.Duration(10<<attempt) * time.Millisecond
		p.log.Debug().
			Err(err).
			Int("attempt", attempt+1).
			Dur("backoff", backoff).
			Msg("SQLITE_BUSY on begin transaction, retrying")
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("failed to begin transaction after %d retries: %w", maxRetries, lastErr)
}

// syncNotify signals the sync goroutine to attempt a sync.
func (p *CachingServerClientDataProvider) syncNotify() {
	select {
	case p.syncNotifyCh <- struct{}{}:
	default:
		// Already notified
	}
}

// updateStatus updates the sync status and notifies watchers.
func (p *CachingServerClientDataProvider) updateStatus() {
	p.statusMu.Lock()

	// Count pending changes
	var pendingCount int
	p.db.QueryRow("SELECT COUNT(*) FROM pending_changes").Scan(&pendingCount)

	// Count conflicts
	var conflictCount int
	p.db.QueryRow("SELECT COUNT(*) FROM conflicts").Scan(&conflictCount)

	// Get last sync time
	lastSyncStr, _ := getMeta(p.db, "last_sync_time")
	var lastSync time.Time
	if lastSyncStr != "" {
		lastSync, _ = time.Parse(time.RFC3339, lastSyncStr)
	}

	p.status.PendingChanges = pendingCount
	p.status.ConflictCount = conflictCount
	p.status.LastSyncTime = lastSync

	status := p.status
	p.statusMu.Unlock()

	// Notify watchers
	p.statusWatcherMu.Lock()
	for _, ch := range p.statusWatchers {
		select {
		case ch <- status:
		default:
		}
	}
	p.statusWatcherMu.Unlock()
}

// =============================================================================
// EventProvider Implementation - Read Operations (All Local)
// =============================================================================

// GetEvent retrieves the event with the specified ID from local SQLite.
func (p *CachingServerClientDataProvider) GetEvent(id model.EventID) (*model.Event, error) {
	row := p.db.QueryRow(
		"SELECT id, name, category, start_time, end_time FROM events WHERE id = ? AND deleted = 0",
		id,
	)

	var e model.Event
	var startStr, endStr string
	err := row.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event with ID '%s' not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting event: %w", err)
	}

	var parseErr error
	e.Start, parseErr = time.Parse(time.RFC3339, startStr)
	if parseErr != nil {
		log.Warn().
			Err(parseErr).
			Str("event_id", string(id)).
			Str("start_str", startStr).
			Msg("GetEvent: failed to parse start time")
	}
	e.End, parseErr = time.Parse(time.RFC3339, endStr)
	if parseErr != nil {
		log.Warn().
			Err(parseErr).
			Str("event_id", string(id)).
			Str("end_str", endStr).
			Msg("GetEvent: failed to parse end time")
	}

	log.Trace().
		Str("event_id", string(e.ID)).
		Str("event_name", e.Name).
		Str("db_start_str", startStr).
		Str("db_end_str", endStr).
		Time("parsed_start", e.Start).
		Time("parsed_end", e.End).
		Msg("GetEvent: retrieved event")

	return &e, nil
}

// GetEventsCoveringTimerange returns all events that cover the given time range.
func (p *CachingServerClientDataProvider) GetEventsCoveringTimerange(start, end time.Time) ([]*model.Event, error) {
	start = start.UTC()
	end = end.UTC()

	if end.Before(start) {
		return nil, fmt.Errorf("end time is before start time")
	}
	if start.Equal(end) {
		return nil, fmt.Errorf("empty time range requested")
	}

	rows, err := p.db.Query(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0 AND start_time < ? AND end_time > ?
		 ORDER BY start_time, end_time DESC`,
		end.Format(time.RFC3339), start.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("error querying events: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var e model.Event
		var startStr, endStr string
		if err := rows.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr); err != nil {
			return nil, fmt.Errorf("error scanning event: %w", err)
		}
		e.Start, _ = time.Parse(time.RFC3339, startStr)
		e.End, _ = time.Parse(time.RFC3339, endStr)
		events = append(events, &e)
	}

	return events, nil
}

// GetEventAfter returns the event that starts after the given time.
func (p *CachingServerClientDataProvider) GetEventAfter(t time.Time) (*model.Event, error) {
	t = t.UTC()

	row := p.db.QueryRow(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0 AND start_time >= ?
		 ORDER BY start_time, end_time DESC
		 LIMIT 1`,
		t.Format(time.RFC3339),
	)

	var e model.Event
	var startStr, endStr string
	err := row.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting event: %w", err)
	}

	e.Start, _ = time.Parse(time.RFC3339, startStr)
	e.End, _ = time.Parse(time.RFC3339, endStr)

	return &e, nil
}

// GetEventBefore returns the event that ends before the given time.
func (p *CachingServerClientDataProvider) GetEventBefore(t time.Time) (*model.Event, error) {
	t = t.UTC()

	row := p.db.QueryRow(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0 AND end_time <= ?
		 ORDER BY end_time DESC, start_time
		 LIMIT 1`,
		t.Format(time.RFC3339),
	)

	var e model.Event
	var startStr, endStr string
	err := row.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting event: %w", err)
	}

	e.Start, _ = time.Parse(time.RFC3339, startStr)
	e.End, _ = time.Parse(time.RFC3339, endStr)

	return &e, nil
}

// GetPrecedingEvent returns the event immediately before the given event in the total ordering.
// Events are ordered by: start_time ASC, end_time DESC (longer events first for same start), id ASC.
// This ensures all events are reachable via prev/next navigation.
func (p *CachingServerClientDataProvider) GetPrecedingEvent(id model.EventID) (*model.Event, error) {
	log.Debug().Str("input_id", string(id)).Msg("GetPrecedingEvent called")

	e, err := p.GetEvent(id)
	if err != nil {
		log.Debug().Err(err).Str("input_id", string(id)).Msg("GetPrecedingEvent: failed to get current event")
		return nil, err
	}

	startStr := e.Start.Format(time.RFC3339)
	endStr := e.End.Format(time.RFC3339)

	log.Debug().
		Str("current_id", string(e.ID)).
		Str("current_name", e.Name).
		Str("current_start", startStr).
		Str("current_end", endStr).
		Time("current_start_time", e.Start).
		Time("current_end_time", e.End).
		Msg("GetPrecedingEvent: current event details")

	// Also get the raw DB value to check for format mismatches
	var dbStartStr, dbEndStr string
	err = p.db.QueryRow("SELECT start_time, end_time FROM events WHERE id = ?", id).Scan(&dbStartStr, &dbEndStr)
	if err == nil {
		log.Trace().
			Str("db_start_str", dbStartStr).
			Str("db_end_str", dbEndStr).
			Str("formatted_start_str", startStr).
			Str("formatted_end_str", endStr).
			Bool("start_matches", dbStartStr == startStr).
			Bool("end_matches", dbEndStr == endStr).
			Msg("GetPrecedingEvent: comparing DB vs formatted time strings")
	}

	// Find the event immediately before this one in total ordering.
	// Total order: start_time ASC, end_time DESC, id ASC
	// "Before" means: start < e.start, OR (start = e.start AND end > e.end),
	//                 OR (start = e.start AND end = e.end AND id < e.id)
	// To get the closest preceding event, order by: start DESC, end ASC, id DESC
	row := p.db.QueryRow(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0 AND (
		   start_time < ?
		   OR (start_time = ? AND end_time > ?)
		   OR (start_time = ? AND end_time = ? AND id < ?)
		 )
		 ORDER BY start_time DESC, end_time ASC, id DESC
		 LIMIT 1`,
		startStr, startStr, endStr, startStr, endStr, id,
	)

	var prev model.Event
	var prevStartStr, prevEndStr string
	err = row.Scan(&prev.ID, &prev.Name, &prev.Category, &prevStartStr, &prevEndStr)
	if err == sql.ErrNoRows {
		log.Debug().Str("current_id", string(id)).Msg("GetPrecedingEvent: no preceding event found")
		return nil, nil
	}
	if err != nil {
		log.Debug().Err(err).Str("current_id", string(id)).Msg("GetPrecedingEvent: error scanning result")
		return nil, fmt.Errorf("error getting preceding event: %w", err)
	}

	var parseErr error
	prev.Start, parseErr = time.Parse(time.RFC3339, prevStartStr)
	if parseErr != nil {
		log.Warn().Err(parseErr).Str("prev_start_str", prevStartStr).Msg("GetPrecedingEvent: failed to parse prev start time")
	}
	prev.End, parseErr = time.Parse(time.RFC3339, prevEndStr)
	if parseErr != nil {
		log.Warn().Err(parseErr).Str("prev_end_str", prevEndStr).Msg("GetPrecedingEvent: failed to parse prev end time")
	}

	log.Debug().
		Str("prev_id", string(prev.ID)).
		Str("prev_name", prev.Name).
		Str("prev_start", prevStartStr).
		Str("prev_end", prevEndStr).
		Bool("same_as_current", prev.ID == id).
		Msg("GetPrecedingEvent: result")

	if prev.ID == id {
		log.Error().
			Str("event_id", string(id)).
			Str("event_name", e.Name).
			Str("db_start", dbStartStr).
			Str("db_end", dbEndStr).
			Str("formatted_start", startStr).
			Str("formatted_end", endStr).
			Msg("BUG: GetPrecedingEvent returning same event as input!")
	}

	return &prev, nil
}

// GetFollowingEvent returns the event immediately after the given event in the total ordering.
// Events are ordered by: start_time ASC, end_time DESC (longer events first for same start), id ASC.
// This ensures all events are reachable via prev/next navigation.
func (p *CachingServerClientDataProvider) GetFollowingEvent(id model.EventID) (*model.Event, error) {
	log.Debug().Str("input_id", string(id)).Msg("GetFollowingEvent called")

	e, err := p.GetEvent(id)
	if err != nil {
		log.Debug().Err(err).Str("input_id", string(id)).Msg("GetFollowingEvent: failed to get current event")
		return nil, err
	}

	startStr := e.Start.Format(time.RFC3339)
	endStr := e.End.Format(time.RFC3339)

	log.Debug().
		Str("current_id", string(e.ID)).
		Str("current_name", e.Name).
		Str("current_start", startStr).
		Str("current_end", endStr).
		Msg("GetFollowingEvent: current event details")

	// Also get the raw DB value to check for format mismatches
	var dbStartStr, dbEndStr string
	err = p.db.QueryRow("SELECT start_time, end_time FROM events WHERE id = ?", id).Scan(&dbStartStr, &dbEndStr)
	if err == nil {
		log.Trace().
			Str("db_start_str", dbStartStr).
			Str("db_end_str", dbEndStr).
			Str("formatted_start_str", startStr).
			Str("formatted_end_str", endStr).
			Bool("start_matches", dbStartStr == startStr).
			Bool("end_matches", dbEndStr == endStr).
			Msg("GetFollowingEvent: comparing DB vs formatted time strings")
	}

	// Find the event immediately after this one in total ordering.
	// Total order: start_time ASC, end_time DESC, id ASC
	// "After" means: start > e.start, OR (start = e.start AND end < e.end),
	//                OR (start = e.start AND end = e.end AND id > e.id)
	// To get the closest following event, order by: start ASC, end DESC, id ASC
	row := p.db.QueryRow(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0 AND (
		   start_time > ?
		   OR (start_time = ? AND end_time < ?)
		   OR (start_time = ? AND end_time = ? AND id > ?)
		 )
		 ORDER BY start_time ASC, end_time DESC, id ASC
		 LIMIT 1`,
		startStr, startStr, endStr, startStr, endStr, id,
	)

	var next model.Event
	var nextStartStr, nextEndStr string
	err = row.Scan(&next.ID, &next.Name, &next.Category, &nextStartStr, &nextEndStr)
	if err == sql.ErrNoRows {
		log.Debug().Str("current_id", string(id)).Msg("GetFollowingEvent: no following event found")
		return nil, nil
	}
	if err != nil {
		log.Debug().Err(err).Str("current_id", string(id)).Msg("GetFollowingEvent: error scanning result")
		return nil, fmt.Errorf("error getting following event: %w", err)
	}

	var parseErr error
	next.Start, parseErr = time.Parse(time.RFC3339, nextStartStr)
	if parseErr != nil {
		log.Warn().Err(parseErr).Str("next_start_str", nextStartStr).Msg("GetFollowingEvent: failed to parse next start time")
	}
	next.End, parseErr = time.Parse(time.RFC3339, nextEndStr)
	if parseErr != nil {
		log.Warn().Err(parseErr).Str("next_end_str", nextEndStr).Msg("GetFollowingEvent: failed to parse next end time")
	}

	log.Debug().
		Str("next_id", string(next.ID)).
		Str("next_name", next.Name).
		Str("next_start", nextStartStr).
		Str("next_end", nextEndStr).
		Bool("same_as_current", next.ID == id).
		Msg("GetFollowingEvent: result")

	if next.ID == id {
		log.Error().
			Str("event_id", string(id)).
			Str("event_name", e.Name).
			Str("db_start", dbStartStr).
			Str("db_end", dbEndStr).
			Str("formatted_start", startStr).
			Str("formatted_end", endStr).
			Msg("BUG: GetFollowingEvent returning same event as input!")
	}

	return &next, nil
}

// SumUpTimespanByCategory returns the total duration of events in the time range, grouped by category.
func (p *CachingServerClientDataProvider) SumUpTimespanByCategory(start, end time.Time) (map[model.CategoryName]time.Duration, error) {
	events, err := p.GetEventsCoveringTimerange(start, end)
	if err != nil {
		return nil, err
	}

	// Build event list and sum up
	eventList := model.EventList{Events: events}
	summary := eventList.SumUpByCategory(func(category model.CategoryName) int {
		if p.categoryProvider == nil {
			return 0
		}
		c := p.categoryProvider.GetCategory(category)
		if c == nil {
			return 0
		}
		return c.Priority
	})

	return summary, nil
}

// =============================================================================
// EventProvider Implementation - Write Operations (Local + Pending + Sync)
// =============================================================================

// AddEvent adds a new event to local storage and marks it pending.
func (p *CachingServerClientDataProvider) AddEvent(e model.Event) (model.EventID, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}

	e.Start = e.Start.UTC()
	e.End = e.End.UTC()

	if !e.Start.Before(e.End) {
		return "", fmt.Errorf("start time is not before end time")
	}

	now := time.Now().UTC()

	// Start transaction with retry for transient busy conditions
	tx, err := p.beginTxWithRetry()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert locally
	_, err = tx.Exec(
		`INSERT INTO events (id, name, category, start_time, end_time, deleted, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, ?)`,
		e.ID, e.Name, e.Category,
		e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert event: %w", err)
	}

	// 2. Mark pending
	_, err = tx.Exec(
		`INSERT OR REPLACE INTO pending_changes (record_type, record_id, operation, changed_at)
		 VALUES ('event', ?, 'create', ?)`,
		e.ID, now.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("failed to mark pending: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	// 3. Trigger sync attempt
	p.syncNotify()
	p.updateStatus()

	return e.ID, nil
}

// RemoveEvent marks an event as deleted locally and marks it pending.
func (p *CachingServerClientDataProvider) RemoveEvent(id model.EventID) error {
	now := time.Now().UTC()

	tx, err := p.beginTxWithRetry()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Tombstone locally
	result, err := tx.Exec(
		"UPDATE events SET deleted = 1, updated_at = ? WHERE id = ?",
		now.Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event with ID '%s' not found", id)
	}

	// 2. Mark pending
	_, err = tx.Exec(
		`INSERT OR REPLACE INTO pending_changes (record_type, record_id, operation, changed_at)
		 VALUES ('event', ?, 'delete', ?)`,
		id, now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to mark pending: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// 3. Trigger sync
	p.syncNotify()
	p.updateStatus()

	return nil
}

// RemoveEvents removes multiple events by their IDs.
func (p *CachingServerClientDataProvider) RemoveEvents(ids []model.EventID) error {
	for _, id := range ids {
		if err := p.RemoveEvent(id); err != nil {
			return fmt.Errorf("error removing event with ID '%s': %w", id, err)
		}
	}
	return nil
}

// updateEventLocal updates an event in local storage and marks it pending.
func (p *CachingServerClientDataProvider) updateEventLocal(e *model.Event) error {
	now := time.Now().UTC()

	tx, err := p.beginTxWithRetry()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		`UPDATE events SET name = ?, category = ?, start_time = ?, end_time = ?, updated_at = ?
		 WHERE id = ? AND deleted = 0`,
		e.Name, e.Category,
		e.Start.Format(time.RFC3339), e.End.Format(time.RFC3339),
		now.Format(time.RFC3339), e.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("event with ID '%s' not found", e.ID)
	}

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO pending_changes (record_type, record_id, operation, changed_at)
		 VALUES ('event', ?, 'update', ?)`,
		e.ID, now.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to mark pending: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	p.syncNotify()
	p.updateStatus()

	return nil
}

// =============================================================================
// EventProvider Implementation - Compound Operations
// =============================================================================

// SplitEvent splits an event at the given time.
func (p *CachingServerClientDataProvider) SplitEvent(id model.EventID, splitTime time.Time) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	splitTime = splitTime.UTC()

	if !(splitTime.After(e.Start) && splitTime.Before(e.End)) {
		return fmt.Errorf("split time is not between start and end time of event")
	}

	// Create second event
	secondEvent := model.Event{
		Name:     e.Name,
		Category: e.Category,
		Start:    splitTime,
		End:      e.End,
	}
	_, err = p.AddEvent(secondEvent)
	if err != nil {
		return fmt.Errorf("failed to add second event: %w", err)
	}

	// Update first event's end
	return p.SetEventEnd(id, splitTime)
}

// SetEventStart sets the start time of an event.
func (p *CachingServerClientDataProvider) SetEventStart(id model.EventID, start time.Time) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	start = start.UTC()

	if !start.Before(e.End) {
		return fmt.Errorf("start time is not before end time")
	}

	e.Start = start
	return p.updateEventLocal(e)
}

// SetEventEnd sets the end time of an event.
func (p *CachingServerClientDataProvider) SetEventEnd(id model.EventID, end time.Time) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	end = end.UTC()

	if !e.Start.Before(end) {
		return fmt.Errorf("start time %s is not before end time %s", e.Start, end)
	}

	e.End = end
	return p.updateEventLocal(e)
}

// SetEventTimes sets both start and end times of an event.
func (p *CachingServerClientDataProvider) SetEventTimes(id model.EventID, start, end time.Time) error {
	start = start.UTC()
	end = end.UTC()

	if !start.Before(end) {
		return fmt.Errorf("start time is not before end time")
	}

	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	e.Start = start
	e.End = end
	return p.updateEventLocal(e)
}

// OffsetEventStart offsets the start time of an event by a duration.
func (p *CachingServerClientDataProvider) OffsetEventStart(id model.EventID, offset time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, err
	}

	newStart := e.Start.Add(offset).UTC()
	if !newStart.Before(e.End) {
		return time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	e.Start = newStart
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, err
	}
	return e.Start, nil
}

// OffsetEventEnd offsets the end time of an event by a duration.
func (p *CachingServerClientDataProvider) OffsetEventEnd(id model.EventID, offset time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, err
	}

	newEnd := e.End.Add(offset).UTC()
	if !e.Start.Before(newEnd) {
		return time.Time{}, fmt.Errorf("resulting end time would not be after start time")
	}

	e.End = newEnd
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, err
	}
	return e.End, nil
}

// OffsetEventTimes offsets both start and end times of an event by a duration.
func (p *CachingServerClientDataProvider) OffsetEventTimes(id model.EventID, offset time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	e.Start = e.Start.Add(offset).UTC()
	e.End = e.End.Add(offset).UTC()

	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return e.Start, e.End, nil
}

// SnapEventStart snaps the start time of an event to the nearest interval.
func (p *CachingServerClientDataProvider) SnapEventStart(id model.EventID, interval time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, err
	}

	newStart := snapToInterval(e.Start, interval).UTC()
	if !newStart.Before(e.End) {
		return time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	e.Start = newStart
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, err
	}
	return e.Start, nil
}

// SnapEventEnd snaps the end time of an event to the nearest interval.
func (p *CachingServerClientDataProvider) SnapEventEnd(id model.EventID, interval time.Duration) (time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, err
	}

	newEnd := snapToInterval(e.End, interval).UTC()
	if !e.Start.Before(newEnd) {
		return time.Time{}, fmt.Errorf("resulting end time would not be after start time")
	}

	e.End = newEnd
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, err
	}
	return e.End, nil
}

// SnapEventTimes snaps both start and end times of an event to the nearest intervals.
func (p *CachingServerClientDataProvider) SnapEventTimes(id model.EventID, interval time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	newStart := snapToInterval(e.Start, interval).UTC()
	newEnd := snapToInterval(e.End, interval).UTC()

	if !newStart.Before(newEnd) {
		return time.Time{}, time.Time{}, fmt.Errorf("resulting start time would not be before end time")
	}

	e.Start = newStart
	e.End = newEnd
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return e.Start, e.End, nil
}

// SnapEventStartPreseveDuration snaps start and adjusts end to preserve duration.
func (p *CachingServerClientDataProvider) SnapEventStartPreseveDuration(id model.EventID, interval time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	newStart := snapToInterval(e.Start, interval).UTC()
	delta := newStart.Sub(e.Start)
	newEnd := e.End.Add(delta).UTC()

	e.Start = newStart
	e.End = newEnd
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return e.Start, e.End, nil
}

// SnapEventEndPreseveDuration snaps end and adjusts start to preserve duration.
func (p *CachingServerClientDataProvider) SnapEventEndPreseveDuration(id model.EventID, interval time.Duration) (time.Time, time.Time, error) {
	e, err := p.GetEvent(id)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	newEnd := snapToInterval(e.End, interval).UTC()
	delta := newEnd.Sub(e.End)
	newStart := e.Start.Add(delta).UTC()

	e.Start = newStart
	e.End = newEnd
	if err := p.updateEventLocal(e); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return e.Start, e.End, nil
}

// SetEventName sets the name of an event.
func (p *CachingServerClientDataProvider) SetEventName(id model.EventID, name string) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	e.Name = name
	return p.updateEventLocal(e)
}

// SetEventCategory sets the category of an event.
func (p *CachingServerClientDataProvider) SetEventCategory(id model.EventID, category model.CategoryName) error {
	e, err := p.GetEvent(id)
	if err != nil {
		return err
	}

	e.Category = category
	return p.updateEventLocal(e)
}

// SetEventAllData sets all data of an event.
func (p *CachingServerClientDataProvider) SetEventAllData(id model.EventID, newData model.Event) error {
	if newData.ID != "" && newData.ID != id {
		return fmt.Errorf("new event data has different ID than specified")
	}

	newData.ID = id
	newData.Start = newData.Start.UTC()
	newData.End = newData.End.UTC()

	if !newData.Start.Before(newData.End) {
		return fmt.Errorf("start time is not before end time")
	}

	return p.updateEventLocal(&newData)
}

// =============================================================================
// DataProviderInfo Implementation
// =============================================================================

// CommitState checkpoints the SQLite WAL.
func (p *CachingServerClientDataProvider) CommitState() error {
	_, err := p.db.Exec("PRAGMA wal_checkpoint(PASSIVE)")
	return err
}

// FullyCommitted returns whether local state is durably persisted.
// This checks local SQLite persistence, not server sync state.
// Use SyncStatus() to check sync state (pending changes, conflicts, etc.).
func (p *CachingServerClientDataProvider) FullyCommitted() (bool, error) {
	// Local SQLite with WAL mode persists data immediately on write.
	// CommitState() checkpoints the WAL, but data is durable even without explicit checkpoint.
	// Return true since all writes are immediately persisted to the local database.
	return true, nil
}

// GetStorageLocationInfo returns information about the storage location.
func (p *CachingServerClientDataProvider) GetStorageLocationInfo() (string, error) {
	return fmt.Sprintf("server:%s", p.serverURL), nil
}

// =============================================================================
// SyncProvider Implementation
// =============================================================================

// SyncStatus returns the current sync state.
func (p *CachingServerClientDataProvider) SyncStatus() SyncStatus {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status
}

// TriggerSync initiates a sync.
func (p *CachingServerClientDataProvider) TriggerSync() <-chan error {
	ch := make(chan error, 1)

	go func() {
		err := p.Sync(p.syncCtx)
		ch <- err
		close(ch)
	}()

	return ch
}

// Conflicts returns unresolved conflicts.
func (p *CachingServerClientDataProvider) Conflicts() ([]Conflict, error) {
	rows, err := p.db.Query(
		"SELECT record_type, record_id, local_version, server_version, detected_at FROM conflicts",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conflicts []Conflict
	for rows.Next() {
		var c Conflict
		var localJSON, serverJSON, detectedAtStr string
		if err := rows.Scan(&c.RecordType, &c.RecordID, &localJSON, &serverJSON, &detectedAtStr); err != nil {
			return nil, err
		}
		c.DetectedAt, _ = time.Parse(time.RFC3339, detectedAtStr)

		// Parse versions based on record type
		if c.RecordType == "event" {
			var local, server model.Event
			json.Unmarshal([]byte(localJSON), &local)
			json.Unmarshal([]byte(serverJSON), &server)
			c.LocalVersion = &local
			c.ServerVersion = &server
		}

		conflicts = append(conflicts, c)
	}

	return conflicts, nil
}

// ResolveConflict resolves a conflict.
func (p *CachingServerClientDataProvider) ResolveConflict(recordType, recordID string, resolution ConflictResolution) error {
	// Get the conflict
	var localJSON, serverJSON string
	err := p.db.QueryRow(
		"SELECT local_version, server_version FROM conflicts WHERE record_type = ? AND record_id = ?",
		recordType, recordID,
	).Scan(&localJSON, &serverJSON)
	if err != nil {
		return fmt.Errorf("conflict not found: %w", err)
	}

	tx, err := p.beginTxWithRetry()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	if recordType == "event" {
		var resolved model.Event
		if resolution.UseLocal {
			json.Unmarshal([]byte(localJSON), &resolved)
		} else if resolution.UseServer {
			json.Unmarshal([]byte(serverJSON), &resolved)
		} else if resolution.Merged != nil {
			resolved = *(resolution.Merged.(*model.Event))
		}

		// Write resolved version to local DB
		_, err = tx.Exec(
			`UPDATE events SET name = ?, category = ?, start_time = ?, end_time = ?, updated_at = ? WHERE id = ?`,
			resolved.Name, resolved.Category,
			resolved.Start.Format(time.RFC3339), resolved.End.Format(time.RFC3339),
			now.Format(time.RFC3339), recordID,
		)
		if err != nil {
			return err
		}

		// Mark as pending (our resolution needs to sync)
		_, err = tx.Exec(
			`INSERT OR REPLACE INTO pending_changes (record_type, record_id, operation, changed_at) VALUES ('event', ?, 'update', ?)`,
			recordID, now.Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}

	// Remove from conflicts table
	_, err = tx.Exec("DELETE FROM conflicts WHERE record_type = ? AND record_id = ?", recordType, recordID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	p.syncNotify()
	p.updateStatus()

	return nil
}

// WatchStatus returns a channel that emits on status changes.
func (p *CachingServerClientDataProvider) WatchStatus() <-chan SyncStatus {
	ch := make(chan SyncStatus, 1)
	p.statusWatcherMu.Lock()
	p.statusWatchers = append(p.statusWatchers, ch)
	p.statusWatcherMu.Unlock()
	return ch
}

// Login authenticates with the server.
func (p *CachingServerClientDataProvider) Login(username, password string) error {
	reqBody, _ := json.Marshal(loginRequest{Username: username, Password: password})
	req, err := http.NewRequest("POST", p.serverURL+"/api/v1/auth/login", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed: %s", string(body))
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return err
	}

	p.token = loginResp.Token
	setMeta(p.db, "auth_token", loginResp.Token)
	setMeta(p.db, "user_id", loginResp.UserID)
	setMeta(p.db, "token_expires_at", loginResp.ExpiresAt)

	// Trigger initial sync
	p.syncNotify()

	return nil
}

// Logout invalidates the current token.
func (p *CachingServerClientDataProvider) Logout() error {
	if p.token == "" {
		return nil
	}

	req, err := http.NewRequest("POST", p.serverURL+"/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// Still clear local token even if request fails
		p.token = ""
		setMeta(p.db, "auth_token", "")
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp.Body.Close()

	p.token = ""
	setMeta(p.db, "auth_token", "")

	return nil
}

// =============================================================================
// Sync Cycle Implementation
// =============================================================================

// runSyncLoop runs the background sync loop.
func (p *CachingServerClientDataProvider) runSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Debounce timer for sync notifications
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.syncNotifyCh:
			// Debounce sync notifications
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDuration, func() {
				p.Sync(ctx)
			})
		case <-ticker.C:
			p.Sync(ctx)
		}
	}
}

// Sync performs a full sync cycle (push then pull).
func (p *CachingServerClientDataProvider) Sync(ctx context.Context) error {
	p.syncMu.Lock()
	defer p.syncMu.Unlock()

	if p.token == "" {
		return errors.New("not authenticated")
	}

	p.statusMu.Lock()
	p.status.Syncing = true
	p.statusMu.Unlock()
	defer func() {
		p.statusMu.Lock()
		p.status.Syncing = false
		p.statusMu.Unlock()
		p.updateStatus()
	}()

	// Push first, then pull
	if _, err := p.push(ctx); err != nil {
		p.statusMu.Lock()
		p.status.LastError = err
		p.status.Online = false
		p.statusMu.Unlock()
		return err
	}

	if _, err := p.pull(ctx); err != nil {
		p.statusMu.Lock()
		p.status.LastError = err
		p.status.Online = false
		p.statusMu.Unlock()
		return err
	}

	p.statusMu.Lock()
	p.status.Online = true
	p.status.LastError = nil
	p.statusMu.Unlock()

	return nil
}

// push sends pending changes to the server.
func (p *CachingServerClientDataProvider) push(ctx context.Context) ([]Conflict, error) {
	// 1. Gather pending
	pending, err := p.getPendingChanges()
	if err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return nil, nil
	}

	// 2. Build request (fetch full records for each pending ID)
	var events []serverEvent
	for _, pc := range pending {
		if pc.RecordType == "event" {
			row := p.db.QueryRow(
				"SELECT id, name, category, start_time, end_time, deleted, updated_at, server_updated_at FROM events WHERE id = ?",
				pc.RecordID,
			)
			var se serverEvent
			var deleted int
			var serverUpdatedAt sql.NullString
			if err := row.Scan(&se.ID, &se.Name, &se.Category, &se.StartTime, &se.EndTime, &deleted, &se.UpdatedAt, &serverUpdatedAt); err != nil {
				continue
			}
			se.Deleted = deleted != 0
			// Set client_updated_at to the last known server version (for conflict detection)
			// If this is a new event (never synced), server_updated_at will be null
			if serverUpdatedAt.Valid {
				se.ClientUpdatedAt = serverUpdatedAt.String
			}
			events = append(events, se)
		}
	}

	req := syncPushRequest{Events: events}
	reqBody, _ := json.Marshal(req)

	// 3. POST /api/v1/sync
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.serverURL+"/api/v1/sync", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("push request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("push failed: %s", string(body))
	}

	var pushResp syncPushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResp); err != nil {
		return nil, err
	}

	// 4. Apply server versions to local DB
	for _, e := range pushResp.Events {
		if err := p.upsertEventFromServer(e); err != nil {
			p.log.Error().Err(err).Str("eventID", e.ID).Msg("failed to upsert event during push response")
		}
	}

	// 5. Store conflicts
	var conflicts []Conflict
	for _, sc := range pushResp.Conflicts {
		if err := p.storeConflict(sc.RecordType, sc.RecordID, sc.ServerVersion); err != nil {
			p.log.Error().Err(err).Msg("failed to store conflict")
		}
		conflicts = append(conflicts, Conflict{
			RecordType: sc.RecordType,
			RecordID:   sc.RecordID,
		})
	}

	// 6. Clear pending for non-conflicts
	conflictIDs := make(map[string]bool)
	for _, c := range pushResp.Conflicts {
		conflictIDs[c.RecordType+":"+c.RecordID] = true
	}
	for _, pc := range pending {
		if !conflictIDs[pc.RecordType+":"+pc.RecordID] {
			p.db.Exec("DELETE FROM pending_changes WHERE record_type = ? AND record_id = ?",
				pc.RecordType, pc.RecordID)
		}
	}

	return conflicts, nil
}

// pull fetches changes from the server.
func (p *CachingServerClientDataProvider) pull(ctx context.Context) (int, error) {
	since, _ := getMeta(p.db, "last_sync_time")
	if since == "" {
		since = "1970-01-01T00:00:00Z"
	}

	// GET /api/v1/sync?since=...
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.serverURL+"/api/v1/sync?since="+url.QueryEscape(since), nil)
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.token)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("pull request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("pull failed: %s", string(body))
	}

	var pullResp syncPullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return 0, err
	}

	count := 0
	for _, e := range pullResp.Events {
		if p.hasPendingChange("event", e.ID) {
			// Conflict: server changed something we also changed locally
			if err := p.storeConflict("event", e.ID, e); err != nil {
				p.log.Error().Err(err).Msg("failed to store conflict")
			}
		} else {
			if err := p.upsertEventFromServer(e); err != nil {
				p.log.Error().Err(err).Str("eventID", e.ID).Msg("failed to upsert event during pull")
			}
		}
		count++
	}

	setMeta(p.db, "last_sync_time", pullResp.ServerTime)
	return count, nil
}

// getPendingChanges retrieves all pending changes.
func (p *CachingServerClientDataProvider) getPendingChanges() ([]pendingChange, error) {
	rows, err := p.db.Query("SELECT record_type, record_id, operation, changed_at FROM pending_changes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []pendingChange
	for rows.Next() {
		var pc pendingChange
		var changedAtStr string
		if err := rows.Scan(&pc.RecordType, &pc.RecordID, &pc.Operation, &changedAtStr); err != nil {
			return nil, err
		}
		pc.ChangedAt, _ = time.Parse(time.RFC3339, changedAtStr)
		changes = append(changes, pc)
	}

	return changes, nil
}

// hasPendingChange checks if there's a pending change for the given record.
func (p *CachingServerClientDataProvider) hasPendingChange(recordType, recordID string) bool {
	var count int
	p.db.QueryRow(
		"SELECT COUNT(*) FROM pending_changes WHERE record_type = ? AND record_id = ?",
		recordType, recordID,
	).Scan(&count)
	return count > 0
}

// normalizeTimeString parses a time string and reformats it to RFC3339 without
// fractional seconds. This ensures consistent string comparison in SQL queries.
// Server may send times with milliseconds (e.g., "2026-01-18T13:04:28.693Z"),
// but we need consistent format for lexicographic comparison.
func normalizeTimeString(timeStr string) string {
	// Try parsing with RFC3339Nano first (handles fractional seconds)
	t, err := time.Parse(time.RFC3339Nano, timeStr)
	if err != nil {
		// Fall back to RFC3339
		t, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			// If all parsing fails, return original (will cause issues but at least logs will show it)
			log.Warn().Str("time_str", timeStr).Msg("normalizeTimeString: failed to parse time, returning original")
			return timeStr
		}
	}
	// Format without fractional seconds
	return t.UTC().Format(time.RFC3339)
}

// upsertEventFromServer inserts or updates an event from server data.
func (p *CachingServerClientDataProvider) upsertEventFromServer(e serverEvent) error {
	deleted := 0
	if e.Deleted {
		deleted = 1
	}

	// Normalize time strings to RFC3339 without fractional seconds.
	// Server sends times like "2026-01-18T13:04:28.693Z" but our queries use
	// RFC3339 format "2026-01-18T13:04:28Z". Lexicographic comparison of these
	// differs because '.' < 'Z' in ASCII, causing navigation bugs.
	startTime := normalizeTimeString(e.StartTime)
	endTime := normalizeTimeString(e.EndTime)
	updatedAt := normalizeTimeString(e.UpdatedAt)

	_, err := p.db.Exec(
		`INSERT INTO events (id, name, category, start_time, end_time, deleted, updated_at, server_updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name = excluded.name,
		   category = excluded.category,
		   start_time = excluded.start_time,
		   end_time = excluded.end_time,
		   deleted = excluded.deleted,
		   updated_at = excluded.updated_at,
		   server_updated_at = excluded.server_updated_at`,
		e.ID, e.Name, e.Category, startTime, endTime, deleted, updatedAt, updatedAt,
	)
	if err != nil {
		p.log.Error().Err(err).
			Str("eventID", e.ID).
			Str("name", e.Name).
			Str("startTime", startTime).
			Str("endTime", endTime).
			Str("originalStartTime", e.StartTime).
			Str("originalEndTime", e.EndTime).
			Msg("failed to upsert event from server")
		return fmt.Errorf("failed to upsert event %s: %w", e.ID, err)
	}
	return nil
}

// storeConflict stores a conflict between local and server versions.
func (p *CachingServerClientDataProvider) storeConflict(recordType, recordID string, serverVersion any) error {
	var localJSON []byte
	if recordType == "event" {
		row := p.db.QueryRow(
			"SELECT id, name, category, start_time, end_time FROM events WHERE id = ?",
			recordID,
		)
		var e model.Event
		var startStr, endStr string
		if err := row.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr); err != nil {
			return err
		}
		e.Start, _ = time.Parse(time.RFC3339, startStr)
		e.End, _ = time.Parse(time.RFC3339, endStr)
		localJSON, _ = json.Marshal(e)
	}

	serverJSON, _ := json.Marshal(serverVersion)
	now := time.Now().UTC()

	_, err := p.db.Exec(
		`INSERT OR REPLACE INTO conflicts (record_type, record_id, local_version, server_version, detected_at)
		 VALUES (?, ?, ?, ?, ?)`,
		recordType, recordID, string(localJSON), string(serverJSON), now.Format(time.RFC3339),
	)
	return err
}

// =============================================================================
// SSE Listener Implementation
// =============================================================================

// runSSE maintains an SSE connection for real-time updates.
func (p *CachingServerClientDataProvider) runSSE(ctx context.Context) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if p.token == "" {
			time.Sleep(backoff)
			continue
		}

		err := p.connectSSE(ctx)
		if err != nil {
			p.log.Error().Err(err).Dur("backoff", backoff).Msg("SSE connection failed, will retry")
			time.Sleep(backoff)
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		// Reset backoff on successful connection
		backoff = time.Second
	}
}

// connectSSE establishes an SSE connection and processes events.
func (p *CachingServerClientDataProvider) connectSSE(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.serverURL+"/api/v1/events/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
	}

	p.log.Info().Str("url", p.serverURL+"/api/v1/events/stream").Msg("SSE connection established")

	p.statusMu.Lock()
	p.status.Online = true
	p.statusMu.Unlock()
	p.updateStatus()

	scanner := bufio.NewScanner(resp.Body)
	var eventType, data string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// End of event
			if eventType == "change" && data != "" {
				p.log.Debug().Str("eventType", eventType).Str("data", data).Msg("received SSE event")
				p.handleSSEChange(data)
			}
			eventType = ""
			data = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}

	if err := scanner.Err(); err != nil {
		p.log.Error().Err(err).Msg("SSE scanner error")
		return err
	}
	return nil
}

// handleSSEChange processes an SSE change event.
func (p *CachingServerClientDataProvider) handleSSEChange(data string) {
	var change struct {
		Type   string      `json:"type"`
		Action string      `json:"action"`
		Record serverEvent `json:"record"` // Server sends "record", not "event"
	}
	if err := json.Unmarshal([]byte(data), &change); err != nil {
		p.log.Error().Err(err).Str("data", data).Msg("failed to parse SSE change")
		return
	}

	p.log.Debug().
		Str("type", change.Type).
		Str("action", change.Action).
		Str("recordID", change.Record.ID).
		Msg("processing SSE change")

	if change.Type == "event" {
		if p.hasPendingChange("event", change.Record.ID) {
			// Conflict: server changed something we also changed locally
			p.log.Warn().Str("eventID", change.Record.ID).Msg("SSE event conflicts with pending local change")
			p.storeConflict("event", change.Record.ID, change.Record)
		} else {
			p.log.Info().
				Str("eventID", change.Record.ID).
				Str("name", change.Record.Name).
				Str("start", change.Record.StartTime).
				Str("end", change.Record.EndTime).
				Bool("deleted", change.Record.Deleted).
				Msg("upserting event from SSE")
			if err := p.upsertEventFromServer(change.Record); err != nil {
				p.log.Error().Err(err).Str("eventID", change.Record.ID).Msg("failed to upsert event from SSE")
				return
			}
		}
		p.updateStatus()
	} else {
		p.log.Warn().Str("type", change.Type).Msg("received SSE change with unknown type")
	}
}

// =============================================================================
// Helper to get all events sorted (for GetPrecedingEvent/GetFollowingEvent)
// =============================================================================

func (p *CachingServerClientDataProvider) getAllEventsSorted() ([]*model.Event, error) {
	rows, err := p.db.Query(
		`SELECT id, name, category, start_time, end_time FROM events
		 WHERE deleted = 0
		 ORDER BY start_time, end_time DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var e model.Event
		var startStr, endStr string
		if err := rows.Scan(&e.ID, &e.Name, &e.Category, &startStr, &endStr); err != nil {
			return nil, err
		}
		e.Start, _ = time.Parse(time.RFC3339, startStr)
		e.End, _ = time.Parse(time.RFC3339, endStr)
		events = append(events, &e)
	}

	sort.Sort(model.ByStartConsideringDuration(events))
	return events, nil
}
