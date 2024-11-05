package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	*sql.DB
}

func NewDB(dsn string) (*DB, error) {
	// Add parseTime=true parameter safely
	if !strings.Contains(dsn, "?") {
		dsn += "?parseTime=true"
	} else if !strings.Contains(dsn, "parseTime=true") {
		dsn += "&parseTime=true"
	}

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
    // First, try dropping the user_preferences table if it exists
    _, err := db.Exec(`DROP TABLE IF EXISTS user_preferences`)
    if err != nil {
        return fmt.Errorf("error dropping user_preferences table: %w", err)
    }

    // Create vehicles table
    _, err = db.Exec(`
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

    // Create preferences table with client_id
    _, err = db.Exec(`
        CREATE TABLE user_preferences (
            id INT AUTO_INCREMENT PRIMARY KEY,
            device_id VARCHAR(255) NOT NULL,
            client_id VARCHAR(255) NOT NULL DEFAULT 'default',
            display_name VARCHAR(255),
            is_hidden BOOLEAN DEFAULT false,
            sort_order INT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
            UNIQUE KEY unique_device_client (device_id, client_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    `)
    return err
}


func (db *DB) GetAllPreferencesForClient(clientID string) ([]models.UserPreference, error) {
    query := `
        SELECT id, device_id, client_id, display_name, is_hidden, sort_order, created_at, updated_at
        FROM user_preferences
        WHERE client_id = ?
        ORDER BY sort_order ASC
    `
    fmt.Printf("Executing query: %s with clientID: %s\n", query, clientID)
    
    rows, err := db.Query(query, clientID)
    if err != nil {
        return nil, fmt.Errorf("error querying preferences: %w", err)
    }
    defer rows.Close()

    var preferences []models.UserPreference
    for rows.Next() {
        var pref models.UserPreference
        var createdAt, updatedAt sql.NullTime
        err := rows.Scan(
            &pref.ID,
            &pref.DeviceID,
            &pref.ClientID,
            &pref.DisplayName,
            &pref.IsHidden,
            &pref.SortOrder,
            &createdAt,
            &updatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("error scanning preference row: %w", err)
        }

        if createdAt.Valid {
            pref.CreatedAt = createdAt.Time
        }
        if updatedAt.Valid {
            pref.UpdatedAt = updatedAt.Time
        }
        preferences = append(preferences, pref)
    }

    // If no preferences found, return empty slice instead of nil
    if preferences == nil {
        preferences = []models.UserPreference{}
    }

    return preferences, nil
}

// GetPreferenceByDeviceAndClientID retrieves a specific preference
func (db *DB) GetPreferenceByDeviceAndClientID(deviceID, clientID string) (*models.UserPreference, error) {
    var pref models.UserPreference
    var createdAt, updatedAt sql.NullTime

    err := db.QueryRow(`
        SELECT id, device_id, client_id, display_name, is_hidden, sort_order, created_at, updated_at
        FROM user_preferences
        WHERE device_id = ? AND client_id = ?
    `, deviceID, clientID).Scan(
        &pref.ID,
        &pref.DeviceID,
        &pref.ClientID,
        &pref.DisplayName,
        &pref.IsHidden,
        &pref.SortOrder,
        &createdAt,
        &updatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("error getting preference: %w", err)
    }

    if createdAt.Valid {
        pref.CreatedAt = createdAt.Time
    }
    if updatedAt.Valid {
        pref.UpdatedAt = updatedAt.Time
    }

    return &pref, nil
}

// CreatePreference creates a new preference
func (db *DB) CreatePreference(pref *models.PreferenceCreate) (*models.UserPreference, error) {
    _, err := db.Exec(`
        INSERT INTO user_preferences (device_id, client_id, display_name, is_hidden, sort_order)
        VALUES (?, ?, ?, ?, ?)
    `, pref.DeviceID, pref.ClientID, pref.DisplayName, pref.IsHidden, pref.SortOrder)
    if err != nil {
        return nil, fmt.Errorf("error creating preference: %w", err)
    }

    // Return the newly created preference
    return db.GetPreferenceByDeviceAndClientID(pref.DeviceID, pref.ClientID)
}

// UpdatePreferenceByDeviceAndClientID updates an existing preference
func (db *DB) UpdatePreferenceByDeviceAndClientID(deviceID, clientID string, updates *models.PreferenceUpdate) (*models.UserPreference, error) {
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

    query += " WHERE device_id = ? AND client_id = ?"
    args = append(args, deviceID, clientID)

    _, err := db.Exec(query, args...)
    if err != nil {
        return nil, fmt.Errorf("error updating preference: %w", err)
    }

    return db.GetPreferenceByDeviceAndClientID(deviceID, clientID)
}

// DeletePreference deletes a preference
func (db *DB) DeletePreference(deviceID, clientID string) error {
    result, err := db.Exec("DELETE FROM user_preferences WHERE device_id = ? AND client_id = ?", deviceID, clientID)
    if err != nil {
        return fmt.Errorf("error deleting preference: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("error getting rows affected: %w", err)
    }

    if rows == 0 {
        return fmt.Errorf("no preference found with device ID: %s and client ID: %s", deviceID, clientID)
    }

    return nil
}
