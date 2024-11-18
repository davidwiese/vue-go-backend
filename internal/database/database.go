// database.go provides MySQL database operations for storing and managing
// user preferences. It handles database connections and CRUD operations
// CRUD operations for the user_preferences table.

package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	_ "github.com/go-sql-driver/mysql"
)

// DB wraps the sql.DB connection and provides custom database methods
type DB struct {
	*sql.DB
}

// Execer interface allows for transaction support in database operations
type Execer interface {
    Exec(query string, args ...interface{}) (sql.Result, error)
    QueryRow(query string, args ...interface{}) *sql.Row
}

// NewDB creates a new database connection with proper configuration
// Called in main.go during server initialization
func NewDB(dsn string) (*DB, error) {
	// Ensure MySQL parses time values correctly
	if !strings.Contains(dsn, "?") {
		dsn += "?parseTime=true"
	} else if !strings.Contains(dsn, "parseTime=true") {
		dsn += "&parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

    // Verify connection is working
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &DB{db}, nil
}

// CreateTableIfNotExists initializes database schema
// Creates table for user preferences
func (db *DB) CreateTableIfNotExists() error {
    // Create preferences table with client_id for frontend display settings
    // Used by VehiclePreferences.vue to store user customizations
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS user_preferences (
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

// GetAllPreferencesForClient retrieves all preferences for a specific client
// Used by VehicleList.vue during initial load and after updates
func (db *DB) GetAllPreferencesForClient(clientID string) ([]models.UserPreference, error) {
    query := `
        SELECT id, device_id, client_id, display_name, is_hidden, sort_order, created_at, updated_at
        FROM user_preferences
        WHERE client_id = ?
        ORDER BY sort_order ASC
    `
    fmt.Printf("Executing query: %s with clientID: %s\n", query, clientID)
    
    // Execute query and handle results
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
        // Convert nullable timestamps to actual times if valid
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
// Used when updating individual vehicle preferences
func (db *DB) GetPreferenceByDeviceAndClientID(deviceID, clientID string, execer Execer) (*models.UserPreference, error) {
    // Use provided execer (transaction) or default to db connection
    if execer == nil {
        execer = db.DB
    }

    var pref models.UserPreference
    var createdAt, updatedAt sql.NullTime

    // Query single preference
    err := execer.QueryRow(`
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

    // Handle case where preference doesn't exist
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


// CreatePreference creates or updates a preference
// Used by VehiclePreferences.vue when saving individual preferences
func (db *DB) CreatePreference(pref *models.PreferenceCreate, execer Execer) (*models.UserPreference, error) {
    // Use provided execer (transaction) or default to db connection
    if execer == nil {
        execer = db.DB
    }
    
    // Use UPSERT to handle insert or update in one query
    _, err := execer.Exec(`
        INSERT INTO user_preferences 
        (device_id, client_id, display_name, is_hidden, sort_order)
        VALUES (?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE
            display_name = VALUES(display_name),
            is_hidden = VALUES(is_hidden),
            sort_order = VALUES(sort_order)
    `, pref.DeviceID, pref.ClientID, pref.DisplayName, pref.IsHidden, pref.SortOrder)
    if err != nil {
        return nil, fmt.Errorf("error creating/updating preference: %w", err)
    }
    fmt.Printf("Created/Updated preference: device_id=%s, client_id=%s\n", pref.DeviceID, pref.ClientID)

    // Return the updated preference data
    return db.GetPreferenceByDeviceAndClientID(pref.DeviceID, pref.ClientID, execer)

}

// UpdatePreferenceByDeviceAndClientID updates specific fields of an existing preference
// Used by VehiclePreferences.vue for partial updates
func (db *DB) UpdatePreferenceByDeviceAndClientID(deviceID, clientID string, updates *models.PreferenceUpdate, execer Execer) (*models.UserPreference, error) {
    if execer == nil {
        execer = db.DB
    }
    // Build dynamic update query based on provided fields
    query := "UPDATE user_preferences SET updated_at = NOW()"
    args := []interface{}{}

    // Add fields to update only if they're provided
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

    // Execute update query
    result, err := execer.Exec(query, args...)
    if err != nil {
        return nil, fmt.Errorf("error updating preference: %w", err)
    }

    // Verify update was successful
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return nil, fmt.Errorf("error getting rows affected: %w", err)
    }

    if rowsAffected == 0 {
        return nil, fmt.Errorf("no preference found for device_id: %s and client_id: %s", deviceID, clientID)
    }
    fmt.Printf("Updated preference: device_id=%s, client_id=%s\n", deviceID, clientID)

    // Return updated preference data
    return db.GetPreferenceByDeviceAndClientID(deviceID, clientID, execer)
}

// DeletePreference removes a preference from the database
// Used by VehiclePreferences.vue when removing customizations
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

// CleanupOldPreferences removes preferences that haven't been updated in the specified duration
// Can be called periodically (e.g., once a day) from main.go
func (db *DB) CleanupOldPreferences(age time.Duration) (int64, error) {
    // Delete preferences older than specified age
    result, err := db.Exec(`
        DELETE FROM user_preferences 
        WHERE updated_at < NOW() - INTERVAL ? DAY
    `, int(age.Hours()/24))
    
    if err != nil {
        return 0, fmt.Errorf("error cleaning up old preferences: %w", err)
    }

    // Return number of rows affected
    rowsDeleted, err := result.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("error getting rows affected: %w", err)
    }

    fmt.Printf("Cleaned up %d old preferences\n", rowsDeleted)
    return rowsDeleted, nil
}