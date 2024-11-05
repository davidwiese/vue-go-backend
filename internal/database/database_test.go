package database

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/davidwiese/fleet-tracker-backend/internal/models"
	"github.com/joho/godotenv"
)

// maskPassword replaces password in DSN with ****** for safe logging
func maskPassword(dsn string) string {
	parts := strings.Split(dsn, "@")
	if len(parts) != 2 {
		return "invalid-dsn-format"
	}
	credentialParts := strings.Split(parts[0], ":")
	if len(credentialParts) != 2 {
		return "invalid-credentials-format"
	}
	return fmt.Sprintf("%s:******@%s", credentialParts[0], parts[1])
}
func TestPreferenceCRUD(t *testing.T) {
	// Load .env file from project root
	if err := godotenv.Load("../../.env"); err != nil {
		t.Logf("Warning: .env file not found, using environment variables")
	}

	// Get DSN from environment variable
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Fatal("DB_DSN environment variable is required")
	}

	// Print DSN for debugging (remove sensitive info)
	safeDSN := maskPassword(dsn)
	t.Logf("Using DSN: %s", safeDSN)

	// Connect to database
	db, err := NewDB(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}
	t.Log("Successfully connected to database")

	// Clean up any existing test data
	cleanup(t, db)
	defer cleanup(t, db) // Clean up after tests complete

	// Ensure tables exist
	err = db.CreateTableIfNotExists()
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}
	t.Log("Successfully created/verified tables")

	// Test Create
	t.Run("Create Preference", func(t *testing.T) {
		testPref := &models.PreferenceCreate{
			DeviceID:    "test-device-1",
			DisplayName: "Test Device",
			IsHidden:    false,
			SortOrder:   1,
		}

		created, err := db.CreatePreference(testPref)
		if err != nil {
			t.Fatalf("Failed to create preference: %v", err)
		}
		if created.DeviceID != testPref.DeviceID {
			t.Errorf("Expected device ID %s, got %s", testPref.DeviceID, created.DeviceID)
		}
	})

	// Test Get
	t.Run("Get Preference", func(t *testing.T) {
		retrieved, err := db.GetPreferenceByDeviceID("test-device-1")
		if err != nil {
			t.Fatalf("Failed to get preference: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected to find preference, got nil")
		}
		if retrieved.DeviceID != "test-device-1" {
			t.Errorf("Expected device ID test-device-1, got %s", retrieved.DeviceID)
		}
	})

	// Test Update
	t.Run("Update Preference", func(t *testing.T) {
		newDisplayName := "Updated Test Device"
		updates := &models.PreferenceUpdate{
			DisplayName: &newDisplayName,
		}
		updated, err := db.UpdatePreference("test-device-1", updates)
		if err != nil {
			t.Fatalf("Failed to update preference: %v", err)
		}
		if updated.DisplayName != newDisplayName {
			t.Errorf("Expected display name %s, got %s", newDisplayName, updated.DisplayName)
		}
	})

	// Test Delete
	t.Run("Delete Preference", func(t *testing.T) {
		err = db.DeletePreference("test-device-1")
		if err != nil {
			t.Fatalf("Failed to delete preference: %v", err)
		}

		// Verify deletion
		deleted, err := db.GetPreferenceByDeviceID("test-device-1")
		if err != nil {
			t.Fatalf("Error checking deleted preference: %v", err)
		}
		if deleted != nil {
			t.Error("Preference still exists after deletion")
		}
	})
}

// cleanup removes any test data from the database
func cleanup(t *testing.T, db *DB) {
	_, err := db.Exec("DELETE FROM user_preferences WHERE device_id LIKE 'test-%'")
	if err != nil {
		t.Logf("Warning: Failed to cleanup test data: %v", err)
	}
}
