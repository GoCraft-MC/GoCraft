package permission

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgreSQLStore returns a Store backed by a PostgreSQL database.
// dsn is a libpq-style connection string or postgres:// URL.
// table is the table name; defaults to "permissions".
func PostgreSQLStore(dsn, table string) (Store, error) {
	if table == "" {
		table = "permissions"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres open: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ` + table +
		` (id INTEGER PRIMARY KEY, document TEXT NOT NULL)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres create table: %w", err)
	}
	return &postgresStore{db: db, table: table}, nil
}

type postgresStore struct {
	db    *sql.DB
	table string
}

func (s *postgresStore) Load() (Document, error) {
	var raw string
	err := s.db.QueryRow(`SELECT document FROM ` + s.table + ` WHERE id = 1`).Scan(&raw)
	if err == sql.ErrNoRows {
		doc := DefaultDocument()
		return doc, s.Save(doc)
	}
	if err != nil {
		return Document{}, fmt.Errorf("postgres load: %w", err)
	}
	var doc Document
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return Document{}, fmt.Errorf("postgres decode: %w", err)
	}
	return doc, nil
}

func (s *postgresStore) Save(doc Document) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO `+s.table+` (id, document) VALUES (1, $1)
		 ON CONFLICT (id) DO UPDATE SET document = EXCLUDED.document`,
		string(data),
	)
	return err
}

func (s *postgresStore) Close() error { return s.db.Close() }
