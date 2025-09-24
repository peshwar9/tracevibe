package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

type DB struct {
	*sql.DB
}

// Interfaces for transaction support
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type Row interface {
	Scan(dest ...interface{}) error
}

type Tx interface {
	Exec(query string, args ...interface{}) (Result, error)
	QueryRow(query string, args ...interface{}) Row
	Commit() error
	Rollback() error
}

// Wrapper for sql.Tx to implement our interface
type txWrapper struct {
	*sql.Tx
}

func (tx *txWrapper) Exec(query string, args ...interface{}) (Result, error) {
	return tx.Tx.Exec(query, args...)
}

func (tx *txWrapper) QueryRow(query string, args ...interface{}) Row {
	return tx.Tx.QueryRow(query, args...)
}

func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) InitSchema() error {
	// Check if tables already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='projects'").Scan(&count)
	if err == nil && count > 0 {
		// Tables already exist, skip initialization
		return nil
	}

	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

func (db *DB) GetProjectByKey(projectKey string) (*Project, error) {
	var p Project
	query := `SELECT id, project_key, name, description, repository_url, version, status, created_at, updated_at
			  FROM projects WHERE project_key = ?`

	err := db.QueryRow(query, projectKey).Scan(
		&p.ID, &p.ProjectKey, &p.Name, &p.Description,
		&p.RepositoryURL, &p.Version, &p.Status, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return &p, err
}

func (db *DB) Begin() (Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &txWrapper{tx}, nil
}

func (db *DB) CreateProject(p *Project) error {
	query := `INSERT INTO projects (project_key, name, description, repository_url, version, status)
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query, p.ProjectKey, p.Name, p.Description, p.RepositoryURL, p.Version, p.Status)
	return err
}

type Project struct {
	ID            string  `json:"id"`
	ProjectKey    string  `json:"project_key"`
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	RepositoryURL *string `json:"repository_url"`
	Version       *string `json:"version"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}