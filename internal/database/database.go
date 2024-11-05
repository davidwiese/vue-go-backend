package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
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
	// Create vehicles table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicles (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL,
			latitude DOUBLE,
			longitude DOUBLE
		)
	`)
	if err != nil {
		return err
	}

	// Create preferences table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_preferences (
			id INT AUTO_INCREMENT PRIMARY KEY,
			device_id VARCHAR(255) NOT NULL,
			display_name VARCHAR(255),
			is_hidden BOOLEAN DEFAULT false,
			sort_order INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_device (device_id)
		)
	`)
	return err
}

// CreatePreference creates a new preference record
func (db *DB) CreatePreference(pref *models.PreferenceCreate) (*models.UserPreference, error) {
	result, err := db.Exec(`
		INSERT INTO user_preferences (device_id, display_name, is_hidden, sort_order)
		VALUES (?, ?, ?, ?)
	`, pref.DeviceID, pref.DisplayName, pref.IsHidden, pref.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("error creating preference: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("error getting last insert ID: %w", err)
	}

	return db.GetPreference(int(id))
}

// GetPreference retrieves a preference by ID
func (db *DB) GetPreference(id int) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := db.QueryRow(`
		SELECT id, device_id, display_name, is_hidden, sort_order, created_at, updated_at
		FROM user_preferences
		WHERE id = ?
	`, id).Scan(
		&pref.ID,
		&pref.DeviceID,
		&pref.DisplayName,
		&pref.IsHidden,
		&pref.SortOrder,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting preference: %w", err)
	}
	return &pref, nil
}

// GetPreferenceByDeviceID retrieves a preference by device ID
func (db *DB) GetPreferenceByDeviceID(deviceID string) (*models.UserPreference, error) {
	var pref models.UserPreference
	err := db.QueryRow(`
		SELECT id, device_id, display_name, is_hidden, sort_order, created_at, updated_at
		FROM user_preferences
		WHERE device_id = ?
	`, deviceID).Scan(
		&pref.ID,
		&pref.DeviceID,
		&pref.DisplayName,
		&pref.IsHidden,
		&pref.SortOrder,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting preference by device ID: %w", err)
	}
	return &pref, nil
}

// GetAllPreferences retrieves all preferences
func (db *DB) GetAllPreferences() ([]models.UserPreference, error) {
	rows, err := db.Query(`
		SELECT id, device_id, display_name, is_hidden, sort_order, created_at, updated_at
		FROM user_preferences
		ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("error querying preferences: %w", err)
	}
	defer rows.Close()

	var preferences []models.UserPreference
	for rows.Next() {
		var pref models.UserPreference
		err := rows.Scan(
			&pref.ID,
			&pref.DeviceID,
			&pref.DisplayName,
			&pref.IsHidden,
			&pref.SortOrder,
			&pref.CreatedAt,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning preference row: %w", err)
		}
		preferences = append(preferences, pref)
	}
	return preferences, nil
}

// UpdatePreference updates an existing preference
func (db *DB) UpdatePreference(deviceID string, updates *models.PreferenceUpdate) (*models.UserPreference, error) {
	// Build dynamic update query based on provided fields
	query := "UPDATE user_preferences SET updated_at = ?"
	args := []interface{}{time.Now()}

	if updates.DisplayName != nil {
		query += ", display_name = ?"
		args = append(args, *updates.DisplayName)
	}
	if updates.IsHidden != nil {
		query += ", is_hidden = ?"
		args = append(args, *updates.IsHidden)
	}
	if updates.SortOrder != nil {
		query += ", sort_order = ?"
		args = append(args, *updates.SortOrder)
	}

	query += " WHERE device_id = ?"
	args = append(args, deviceID)

	_, err := db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error updating preference: %w", err)
	}

	return db.GetPreferenceByDeviceID(deviceID)
}

// DeletePreference deletes a preference by device ID
func (db *DB) DeletePreference(deviceID string) error {
	result, err := db.Exec("DELETE FROM user_preferences WHERE device_id = ?", deviceID)
	if err != nil {
		return fmt.Errorf("error deleting preference: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("no preference found with device ID: %s", deviceID)
	}

	return nil
}