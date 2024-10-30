package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	*sql.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) CreateTableIfNotExists() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicles (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			latitude DOUBLE,
			longitude DOUBLE
		)
	`)
	return err
}
