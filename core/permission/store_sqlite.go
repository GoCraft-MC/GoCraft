package permission

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"
)

// SQLiteStore returns a Store backed by a SQLite database file at dsn.
// table is the table name used to store the document; defaults to "permissions".
func SQLiteStore(dsn, table string) (Store, error) {
	if table == "" {
		table = "permissions"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open %q: %w", dsn, err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + table +
		` (id INTEGER PRIMARY KEY, document TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite create table: %w", err)
	}
	return &sqliteStore{db: db, table: table}, nil
}

type sqliteStore struct {
	db    *sql.DB
	table string
}

func (s *sqliteStore) Load() (Document, error) {
	var raw string
	err := s.db.QueryRow(`SELECT document FROM ` + s.table + ` WHERE id = 1`).Scan(&raw)
	if err == sql.ErrNoRows {
		doc := DefaultDocument()
		return doc, s.Save(doc)
	}
	if err != nil {
		return Document{}, fmt.Errorf("sqlite load: %w", err)
	}
	var doc Document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return Document{}, fmt.Errorf("sqlite decode: %w", err)
	}
	return doc, nil
}

func (s *sqliteStore) Save(doc Document) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO `+s.table+` (id, document) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET document = excluded.document`,
		string(data),
	)
	return err
}

func (s *sqliteStore) Close() error { return s.db.Close() }
