package intelserver

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ja-he/dayplan/internal/model"
	_ "modernc.org/sqlite"
)

// Store handles persistence of events.
type Store struct {
	db *sql.DB
}

// NewStore creates a new store with the given DSN.
// Use ":memory:" for an in-memory database.
func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT,
			retrieved INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX IF NOT EXISTS idx_events_retrieved ON events(retrieved);
	`)
	return err
}

// BeginEvent creates a new incomplete event with the given ID and name, starting now.
func (s *Store) BeginEvent(id, name string) error {
	_, err := s.db.Exec(
		`INSERT INTO events (id, name, start_time) VALUES (?, ?, ?)`,
		id, name, time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

// EndEvent sets the end time for an event.
func (s *Store) EndEvent(id string) error {
	result, err := s.db.Exec(
		`UPDATE events SET end_time = ? WHERE id = ? AND end_time IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), id,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("event not found or already ended: %s", id)
	}
	return nil
}

// RetrieveEvents returns all completed, unretrieved events and marks them as retrieved.
func (s *Store) RetrieveEvents() ([]model.Event, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id, name, start_time, end_time
		FROM events
		WHERE retrieved = 0 AND end_time IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.Event
	var ids []any
	for rows.Next() {
		var e model.Event
		var startStr, endStr string
		if err := rows.Scan(&e.ID, &e.Name, &startStr, &endStr); err != nil {
			return nil, err
		}
		e.Start, _ = time.Parse(time.RFC3339Nano, startStr)
		endTime, _ := time.Parse(time.RFC3339Nano, endStr)
		e.End = endTime
		events = append(events, e)
		ids = append(ids, e.ID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) > 0 {
		// Build placeholders for IN clause
		placeholders := "?"
		for i := 1; i < len(ids); i++ {
			placeholders += ",?"
		}
		_, err = tx.Exec(
			fmt.Sprintf(`UPDATE events SET retrieved = 1 WHERE id IN (%s)`, placeholders),
			ids...,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return events, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
