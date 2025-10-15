package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"vaultke-backend/database"
	"vaultke-backend/models"
	"vaultke-backend/services"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("🚀 Initializing VaultKe Server...")

	// Get database path
	dbPath := getDBPath()
	log.Printf("📌 Database: %s", dbPath)

	// Initialize database connection
	db, err := initializeDatabase(dbPath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	log.Println("📊 Setting up database...")
	migrator := database.NewMigrationManager(db)
	if err := migrator.RunMigrations(); err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	// Initialize notification preferences for existing users
	log.Println("👥 Setting up notification preferences for existing users...")
	if err := initializeExistingUsers(db); err != nil {
		log.Printf("⚠️  Warning: Failed to initialize existing users: %v", err)
	}

	// Verify system integrity
	log.Println("🔍 Verifying system integrity...")
	if err := verifySystemIntegrity(db, migrator); err != nil {
		log.Fatalf("❌ System integrity check failed: %v", err)
	}

	// Setup notification sounds directory
	log.Println("🔊 Setting up notification sounds...")
	if err := setupNotificationSounds(); err != nil {
		log.Printf("⚠️  Warning: Failed to setup notification sounds: %v", err)
	}

	// Display system status
	displaySystemStatus(db, migrator)

	log.Println("\n🎉 Server initialization completed successfully!")
	log.Println("✅ The notification system is ready to use!")
	log.Println("   • Users can set notification preferences")
	log.Println("   • Sounds will be played from /notification_sound/ring.mp3")
	log.Println("   • Reminders and notifications are fully functional")
}

// getDBPath returns the database file path
func getDBPath() string {
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		return dbPath
	}

	// Default to vaultke.db in the backend directory
	return filepath.Join(".", "vaultke.db")
}

// initializeDatabase creates and configures the database connection
func initializeDatabase(dbPath string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}

// initializeExistingUsers creates notification preferences for existing users
func initializeExistingUsers(db *sql.DB) error {
	notificationService := services.NewNotificationService(db)

	// Get default sound ID
	sounds, err := notificationService.GetAvailableSounds()
	if err != nil {
		return fmt.Errorf("failed to get available sounds: %w", err)
	}

	var defaultSoundID *int
	for _, sound := range sounds {
		if sound.IsDefault {
			defaultSoundID = &sound.ID
			break
		}
	}

	if defaultSoundID == nil {
		return fmt.Errorf("no default notification sound found")
	}

	// Get users without notification preferences
	query := `
		SELECT u.id 
		FROM users u 
		LEFT JOIN user_notification_preferences unp ON u.id = unp.user_id 
		WHERE unp.user_id IS NULL
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var userIDs []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return fmt.Errorf("failed to scan user ID: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 {
		log.Println("ℹ️  All users already have notification preferences.")
		return nil
	}

	// Create default preferences for users
	insertQuery := `
		INSERT INTO user_notification_preferences 
		(user_id, notification_sound_id, sound_enabled, vibration_enabled, volume_level,
		 chama_notifications, transaction_notifications, reminder_notifications, 
		 system_notifications, marketing_notifications, quiet_hours_enabled,
		 quiet_hours_start, quiet_hours_end, timezone, notification_frequency,
		 priority_only_during_quiet, created_at, updated_at)
		VALUES (?, ?, 1, 1, 80, 1, 1, 1, 1, 0, 0, '22:00:00', '07:00:00', 
		        'Africa/Nairobi', 'immediate', 1, datetime('now'), datetime('now'))
	`

	stmt, err := db.Prepare(insertQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for _, userID := range userIDs {
		_, err := stmt.Exec(userID, defaultSoundID)
		if err != nil {
			log.Printf("⚠️  Failed to create preferences for user %d: %v", userID, err)
			continue
		}
		count++
	}

	log.Printf("✅ Initialized notification preferences for %d existing users.", count)
	return nil
}

// verifySystemIntegrity checks if the notification system is properly set up
func verifySystemIntegrity(db *sql.DB, migrator *database.MigrationManager) error {
	// Check if notification system is ready
	ready, err := migrator.IsNotificationSystemReady()
	if err != nil {
		return fmt.Errorf("failed to check system readiness: %w", err)
	}

	if !ready {
		return fmt.Errorf("notification system is not properly configured")
	}

	// Check individual components
	components := map[string]func() error{
		"notification_sounds":           func() error { return checkTable(db, "notification_sounds") },
		"user_notification_preferences": func() error { return checkTable(db, "user_notification_preferences") },
		"notifications":                 func() error { return checkTable(db, "notifications") },
		"notification_templates":        func() error { return checkTable(db, "notification_templates") },
		"user_reminders":                func() error { return checkTable(db, "user_reminders") },
		"notification_delivery_log":     func() error { return checkTable(db, "notification_delivery_log") },
	}

	for component, checkFunc := range components {
		if err := checkFunc(); err != nil {
			log.Printf("  ❌ %s: %v", component, err)
			return fmt.Errorf("component %s failed integrity check: %w", component, err)
		}
		log.Printf("  ✅ %s: OK", component)
	}

	// Check for default sound
	var defaultSoundCount int
	err = db.QueryRow("SELECT COUNT(*) FROM notification_sounds WHERE is_default = 1").Scan(&defaultSoundCount)
	if err != nil {
		return fmt.Errorf("failed to check default sound: %w", err)
	}

	if defaultSoundCount == 0 {
		log.Printf("  ❌ default_sound: MISSING")
		return fmt.Errorf("no default notification sound found")
	}
	log.Printf("  ✅ default_sound: OK")

	// Check for templates
	var templateCount int
	err = db.QueryRow("SELECT COUNT(*) FROM notification_templates").Scan(&templateCount)
	if err != nil {
		return fmt.Errorf("failed to check templates: %w", err)
	}

	if templateCount == 0 {
		log.Printf("  ❌ notification_templates: EMPTY")
		return fmt.Errorf("no notification templates found")
	}
	log.Printf("  ✅ notification_templates: OK (%d templates)", templateCount)

	return nil
}

// checkTable verifies that a table exists and is accessible
func checkTable(db *sql.DB, tableName string) error {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return fmt.Errorf("table not accessible: %w", err)
	}
	return nil
}

// setupNotificationSounds creates the notification sounds directory
func setupNotificationSounds() error {
	soundsDir := filepath.Join(".", "notification_sound")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(soundsDir, 0755); err != nil {
		return fmt.Errorf("failed to create notification sounds directory: %w", err)
	}
	log.Printf("  📁 Created notification sounds directory: %s", soundsDir)

	// Check if ring.mp3 exists
	ringPath := filepath.Join(soundsDir, "ring.mp3")
	if _, err := os.Stat(ringPath); os.IsNotExist(err) {
		log.Printf("  ⚠️  Warning: ring.mp3 not found at %s", ringPath)
		log.Printf("     Please copy your ring.mp3 file to this location.")
	} else {
		log.Printf("  🔊 Default notification sound (ring.mp3) found")
	}

	return nil
}

// displaySystemStatus shows the current system status
func displaySystemStatus(db *sql.DB, migrator *database.MigrationManager) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 VAULTKE NOTIFICATION SYSTEM STATUS")
	fmt.Println(strings.Repeat("=", 50))

	// Migration status
	migrations, err := migrator.GetMigrationStatus()
	if err != nil {
		log.Printf("Failed to get migration status: %v", err)
		return
	}

	fmt.Printf("🔄 Migrations: %d completed\n", len(migrations))
	for _, migration := range migrations {
		fmt.Printf("   ✅ %s (%v)\n", migration["migration"], migration["executed_at"])
	}

	// Default sound status
	notificationService := services.NewNotificationService(db)
	sounds, err := notificationService.GetAvailableSounds()
	if err != nil {
		log.Printf("Failed to get available sounds: %v", err)
		return
	}

	var defaultSound *models.NotificationSound
	for _, sound := range sounds {
		if sound.IsDefault {
			defaultSound = &sound
			break
		}
	}

	if defaultSound != nil {
		fmt.Println("\n🔊 Default Notification Sound:")
		fmt.Printf("   📛 Name: %s\n", defaultSound.Name)
		fmt.Printf("   📁 Path: %s\n", defaultSound.FilePath)
		if defaultSound.DurationSeconds > 0 {
			fmt.Printf("   ⏱️  Duration: %.2fs\n", defaultSound.DurationSeconds)
		}
	}

	// Available sounds
	fmt.Printf("\n🎵 Available Notification Sounds: %d\n", len(sounds))
	for _, sound := range sounds {
		icon := "🎵"
		if sound.IsDefault {
			icon = "🔊"
		}
		fmt.Printf("   %s %s (%s)\n", icon, sound.Name, sound.FilePath)
	}

	// System readiness
	ready, err := migrator.IsNotificationSystemReady()
	if err != nil {
		log.Printf("Failed to check system readiness: %v", err)
		return
	}

	readyIcon := "❌"
	readyStatus := "NOT READY"
	if ready {
		readyIcon = "✅"
		readyStatus = "READY"
	}

	fmt.Printf("\n🚀 System Status: %s %s\n", readyIcon, readyStatus)
	fmt.Println(strings.Repeat("=", 50))
}
