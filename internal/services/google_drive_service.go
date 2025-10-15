package services

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveService handles Google Drive backup operations
type GoogleDriveService struct {
	db            *sql.DB
	encryptionKey []byte
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	BackupID  string    `json:"backup_id"`
	FileSize  int64     `json:"file_size"`
	Timestamp time.Time `json:"timestamp"`
}

// RestoreResult represents the result of a restore operation
type RestoreResult struct {
	RestoredItems int       `json:"restored_items"`
	Timestamp     time.Time `json:"timestamp"`
}

// BackupInfo represents backup information
type BackupInfo struct {
	Connected   bool      `json:"connected"`
	LastBackup  time.Time `json:"last_backup"`
	BackupSize  int64     `json:"backup_size"`
	BackupCount int       `json:"backup_count"`
}

// UserBackupData represents the structure of user backup data
type UserBackupData struct {
	UserID      string                 `json:"user_id"`
	BackupDate  time.Time              `json:"backup_date"`
	Profile     map[string]interface{} `json:"profile"`
	Chamas      []map[string]interface{} `json:"chamas"`
	Transactions []map[string]interface{} `json:"transactions"`
	Meetings    []map[string]interface{} `json:"meetings"`
	Documents   []map[string]interface{} `json:"documents"`
	Settings    map[string]interface{} `json:"settings"`
}

// NewGoogleDriveService creates a new Google Drive service
func NewGoogleDriveService(db *sql.DB) *GoogleDriveService {
	// Get encryption key from environment variables
	encryptionKey := os.Getenv("GOOGLE_DRIVE_ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "vaultke-google-drive-encryption-key-32" // Default fallback
	}

	// Ensure key is exactly 32 bytes for AES-256
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	return &GoogleDriveService{
		db:            db,
		encryptionKey: key,
	}
}

// StoreUserTokens stores encrypted Google Drive tokens for a user
func (gds *GoogleDriveService) StoreUserTokens(userID, accessToken, refreshToken string, expiresIn int) error {
	// Ensure tables exist
	err := gds.ensureTablesExist()
	if err != nil {
		return fmt.Errorf("failed to ensure tables exist: %v", err)
	}

	// Encrypt tokens
	encryptedAccessToken, err := gds.encrypt(accessToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt access token: %v", err)
	}

	encryptedRefreshToken, err := gds.encrypt(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to encrypt refresh token: %v", err)
	}

	// Calculate expiry time
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	// Store tokens using SQLite UPSERT syntax
	query := `
		INSERT INTO google_drive_tokens (user_id, access_token, refresh_token, expires_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
		access_token = excluded.access_token,
		refresh_token = excluded.refresh_token,
		expires_at = excluded.expires_at,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err = gds.db.Exec(query, userID, encryptedAccessToken, encryptedRefreshToken, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store Google Drive tokens: %v", err)
	}

	return nil
}

// GetUserTokens retrieves and decrypts Google Drive tokens for a user
func (gds *GoogleDriveService) GetUserTokens(userID string) (*oauth2.Token, error) {
	var encryptedAccessToken, encryptedRefreshToken string
	var expiresAtStr string

	query := `
		SELECT access_token, refresh_token, expires_at
		FROM google_drive_tokens
		WHERE user_id = ?
	`

	err := gds.db.QueryRow(query, userID).Scan(&encryptedAccessToken, &encryptedRefreshToken, &expiresAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no Google Drive tokens found for user")
		}
		return nil, fmt.Errorf("failed to retrieve tokens: %v", err)
	}

	// Parse the expires_at string to time.Time
	expiresAt, err := parseTimeString(expiresAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expires_at '%s': %v", expiresAtStr, err)
	}

	// Decrypt tokens
	accessToken, err := gds.decrypt(encryptedAccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %v", err)
	}

	refreshToken, err := gds.decrypt(encryptedRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %v", err)
	}

	return &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// DisconnectUser revokes tokens and removes them from database
func (gds *GoogleDriveService) DisconnectUser(userID string) error {
	// Get tokens first to revoke them
	token, err := gds.GetUserTokens(userID)
	if err != nil {
		// If no tokens found, just remove any database entries
		_, deleteErr := gds.db.Exec("DELETE FROM google_drive_tokens WHERE user_id = ?", userID)
		return deleteErr
	}

	// Revoke the token with Google
	revokeURL := fmt.Sprintf("https://oauth2.googleapis.com/revoke?token=%s", token.AccessToken)
	resp, err := http.Post(revokeURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		// Log error but continue with database cleanup
		fmt.Printf("Failed to revoke Google token: %v\n", err)
	} else {
		resp.Body.Close()
	}

	// Remove tokens from database
	_, err = gds.db.Exec("DELETE FROM google_drive_tokens WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to remove tokens from database: %v", err)
	}

	return nil
}

// IsUserConnected checks if user has valid Google Drive tokens
func (gds *GoogleDriveService) IsUserConnected(userID string) (bool, error) {
	fmt.Printf("🔍 IsUserConnected: Checking connection for user: %s\n", userID)

	// First, ensure the table exists
	err := gds.ensureTablesExist()
	if err != nil {
		fmt.Printf("❌ IsUserConnected: Failed to ensure tables exist: %v\n", err)
		return false, fmt.Errorf("failed to ensure tables exist: %v", err)
	}

	// Check if Google Drive credentials are configured
	clientID := os.Getenv("GOOGLE_DRIVE_CLIENT_ID")
	if clientID == "" {
		fmt.Printf("❌ Google Drive credentials not configured on server\n")
		return false, fmt.Errorf("Google Drive credentials not configured on server. Please set GOOGLE_DRIVE_CLIENT_ID and GOOGLE_DRIVE_CLIENT_SECRET environment variables")
	}

	var count int
	query := `
		SELECT COUNT(*)
		FROM google_drive_tokens
		WHERE user_id = ? AND expires_at > datetime('now')
	`

	fmt.Printf("🔍 IsUserConnected: Executing query for user %s\n", userID)
	err = gds.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		fmt.Printf("❌ IsUserConnected: Query failed: %v\n", err)
		return false, fmt.Errorf("failed to check connection status: %v", err)
	}

	fmt.Printf("🔍 IsUserConnected: Found %d token records for user %s\n", count, userID)

	// If user has tokens, they are considered connected
	if count > 0 {
		fmt.Printf("🔍 IsUserConnected: User %s has Google Drive tokens\n", userID)
		return true, nil
	} else {
		fmt.Printf("🔍 IsUserConnected: No tokens found for user %s\n", userID)
		return false, nil
	}

	result := count > 0
	fmt.Printf("✅ IsUserConnected: Final result for user %s: %v\n", userID, result)
	return result, nil
}

// ensureTablesExist creates the necessary tables if they don't exist
func (gds *GoogleDriveService) ensureTablesExist() error {
	// Create google_drive_tokens table
	createTokensTable := `
		CREATE TABLE IF NOT EXISTS google_drive_tokens (
			user_id TEXT PRIMARY KEY,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := gds.db.Exec(createTokensTable)
	if err != nil {
		return fmt.Errorf("failed to create google_drive_tokens table: %v", err)
	}

	// Create google_drive_backups table
	createBackupsTable := `
		CREATE TABLE IF NOT EXISTS google_drive_backups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			backup_date DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err = gds.db.Exec(createBackupsTable)
	if err != nil {
		return fmt.Errorf("failed to create google_drive_backups table: %v", err)
	}

	// Create index for google_drive_backups table
	createBackupsIndex := `
		CREATE INDEX IF NOT EXISTS idx_google_drive_backups_user_id ON google_drive_backups (user_id)
	`
	_, err = gds.db.Exec(createBackupsIndex)
	if err != nil {
		return fmt.Errorf("failed to create index on google_drive_backups table: %v", err)
	}

	return nil
}

// CreateUserBackup creates a backup of user data to Google Drive
func (gds *GoogleDriveService) CreateUserBackup(userID string) (*BackupResult, error) {
	// First check if user is connected to Google Drive
	connected, err := gds.IsUserConnected(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check Google Drive connection: %v", err)
	}

	if !connected {
		return nil, fmt.Errorf("user is not connected to Google Drive. Please connect your Google Drive account first")
	}

	// Get user tokens
	token, err := gds.GetUserTokens(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %v", err)
	}

	// Proceed with Google Drive backup for authenticated users
	fmt.Printf("🔧 Creating Google Drive backup for user %s\n", userID)

	// Check if using mock tokens (for development)
	if strings.Contains(token.AccessToken, "mock_access_token_for_testing") {
		fmt.Printf("⚠️ WARNING: Using mock tokens for testing. This will NOT create actual files in Google Drive.\n")
		fmt.Printf("   Use real OAuth tokens for production backups.\n")
	}



	// Create OAuth2 config with proper credentials
	clientID := os.Getenv("GOOGLE_DRIVE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_DRIVE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_DRIVE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("Google Drive credentials not configured on server. Please set GOOGLE_DRIVE_CLIENT_ID and GOOGLE_DRIVE_CLIENT_SECRET environment variables")
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{drive.DriveFileScope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	// Check if token is expired and refresh if needed
	if token.Expiry.Before(time.Now()) {
		// Token is expired, try to refresh
		newToken, err := config.TokenSource(context.Background(), token).Token()
		if err != nil {
			// Check if the error is due to invalid/expired refresh token
			if strings.Contains(err.Error(), "invalid_grant") {
				// Automatically clean up the expired tokens
				cleanupErr := gds.DisconnectUser(userID)
				if cleanupErr != nil {
					fmt.Printf("Warning: failed to clean up expired tokens: %v\n", cleanupErr)
				}
				return nil, fmt.Errorf("Google Drive tokens have expired and cannot be refreshed. The connection has been automatically disconnected. Please reconnect your Google Drive account")
			}
			return nil, fmt.Errorf("failed to refresh expired token: %v", err)
		}

		// Update stored token
		err = gds.StoreUserTokens(userID, newToken.AccessToken, newToken.RefreshToken, int(newToken.Expiry.Sub(time.Now()).Seconds()))
		if err != nil {
			// Log error but continue with new token
			fmt.Printf("Warning: failed to update refreshed token: %v\n", err)
		}

		token = newToken
	}

	// Create authenticated client
	client := config.Client(context.Background(), token)

	// Create Drive service
	driveService, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Drive service: %v", err)
	}

	// Collect user data
	backupData, err := gds.collectUserData(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to collect user data: %v", err)
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup data: %v", err)
	}

	// Create or get backup folder
	fmt.Printf("📁 Creating/getting backup folder...\n")
	folderID, err := gds.getOrCreateBackupFolder(driveService)
	if err != nil {
		fmt.Printf("❌ Failed to create/get backup folder: %v\n", err)
		return nil, fmt.Errorf("failed to create backup folder: %v", err)
	}
	fmt.Printf("✅ Backup folder ready: %s\n", folderID)

	// Create backup file
	fileName := fmt.Sprintf("vaultke_backup_%s_%s.json", userID, time.Now().Format("20060102_150405"))
	fmt.Printf("📄 Creating backup file: %s\n", fileName)

	file := &drive.File{
		Name:    fileName,
		Parents: []string{folderID},
	}

	// Upload file
	fmt.Printf("⬆️ Uploading file to Google Drive...\n")
	uploadedFile, err := driveService.Files.Create(file).Media(bytes.NewReader(jsonData)).Do()
	if err != nil {
		fmt.Printf("❌ Failed to upload backup file: %v\n", err)
		return nil, fmt.Errorf("failed to upload backup file: %v", err)
	}
	fmt.Printf("✅ File uploaded successfully! File ID: %s\n", uploadedFile.Id)
	fmt.Printf("📊 File size: %d bytes\n", len(jsonData))

	// Record backup in database
	err = gds.recordBackup(userID, fileName, int64(len(jsonData)))
	if err != nil {
		// Log error but don't fail the backup
		fmt.Printf("Failed to record backup in database: %v\n", err)
	}

	return &BackupResult{
		BackupID:  fileName,
		FileSize:  int64(len(jsonData)),
		Timestamp: time.Now(),
	}, nil
}

// Helper function to encrypt data
func (gds *GoogleDriveService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(gds.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Helper function to decrypt data
func (gds *GoogleDriveService) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(gds.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// collectUserData gathers all user-specific data for backup
func (gds *GoogleDriveService) collectUserData(userID string) (*UserBackupData, error) {
	backupData := &UserBackupData{
		UserID:     userID,
		BackupDate: time.Now(),
	}

	// Collect user profile
	profile, err := gds.getUserProfile(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %v", err)
	}
	backupData.Profile = profile

	// Collect user's chamas
	chamas, err := gds.getUserChamas(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user chamas: %v", err)
	}
	backupData.Chamas = chamas

	// Collect user transactions
	transactions, err := gds.getUserTransactions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user transactions: %v", err)
	}
	backupData.Transactions = transactions

	// Collect user meetings
	meetings, err := gds.getUserMeetings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user meetings: %v", err)
	}
	backupData.Meetings = meetings

	// Collect user documents
	documents, err := gds.getUserDocuments(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user documents: %v", err)
	}
	backupData.Documents = documents

	// Collect user settings
	settings, err := gds.getUserSettings(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user settings: %v", err)
	}
	backupData.Settings = settings

	return backupData, nil
}

// getUserProfile gets user profile data
func (gds *GoogleDriveService) getUserProfile(userID string) (map[string]interface{}, error) {
	query := `
		SELECT id, first_name, last_name, email, phone, role, created_at, updated_at
		FROM users
		WHERE id = ?
	`

	var id string
	var firstName, lastName, email, phone, role sql.NullString
	var createdAtStr, updatedAtStr string

	err := gds.db.QueryRow(query, userID).Scan(
		&id, &firstName, &lastName, &email, &phone, &role, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	// Parse time strings
	createdAt, _ := parseTimeString(createdAtStr)
	updatedAt, _ := parseTimeString(updatedAtStr)

	profile := map[string]interface{}{
		"id":         id,
		"first_name": firstName.String,
		"last_name":  lastName.String,
		"email":      email.String,
		"phone":      phone.String,
		"role":       role.String,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}

	return profile, nil
}

// getUserChamas gets user's chama memberships
func (gds *GoogleDriveService) getUserChamas(userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT c.id, c.name, c.description, cm.role, cm.joined_at
		FROM chamas c
		JOIN chama_members cm ON c.id = cm.chama_id
		WHERE cm.user_id = ?
	`

	rows, err := gds.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chamas []map[string]interface{}
	for rows.Next() {
		var id string
		var name, description, role sql.NullString
		var joinedAtStr string

		err := rows.Scan(&id, &name, &description, &role, &joinedAtStr)
		if err != nil {
			return nil, err
		}

		// Parse time string
		joinedAt, _ := parseTimeString(joinedAtStr)

		chama := map[string]interface{}{
			"id":          id,
			"name":        name.String,
			"description": description.String,
			"role":        role.String,
			"joined_at":   joinedAt,
		}

		chamas = append(chamas, chama)
	}

	return chamas, nil
}

// getUserTransactions gets user's transaction history
func (gds *GoogleDriveService) getUserTransactions(userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, from_wallet_id, to_wallet_id, amount, type, description, status, created_at
		FROM transactions
		WHERE initiated_by = ?
		ORDER BY created_at DESC
		LIMIT 1000
	`

	rows, err := gds.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var id string
		var fromWalletID, toWalletID, description, status sql.NullString
		var amount float64
		var transactionType string
		var createdAtStr string

		err := rows.Scan(
			&id, &fromWalletID, &toWalletID, &amount, &transactionType,
			&description, &status, &createdAtStr,
		)
		if err != nil {
			return nil, err
		}

		// Parse time string
		createdAt, _ := parseTimeString(createdAtStr)

		transaction := map[string]interface{}{
			"id":             id,
			"from_wallet_id": fromWalletID.String,
			"to_wallet_id":   toWalletID.String,
			"amount":         amount,
			"type":           transactionType,
			"description":    description.String,
			"status":         status.String,
			"created_at":     createdAt,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// getUserMeetings gets user's meeting history
func (gds *GoogleDriveService) getUserMeetings(userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT m.id, m.chama_id, m.title, m.description, m.scheduled_at, m.status
		FROM meetings m
		JOIN chama_members cm ON m.chama_id = cm.chama_id
		WHERE cm.user_id = ?
		ORDER BY m.scheduled_at DESC
		LIMIT 500
	`

	rows, err := gds.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []map[string]interface{}
	for rows.Next() {
		var id string
		var chamaID, title, description, status sql.NullString
		var scheduledAtStr string

		err := rows.Scan(
			&id, &chamaID, &title, &description, &scheduledAtStr, &status,
		)
		if err != nil {
			return nil, err
		}

		// Parse time string
		scheduledAt, _ := parseTimeString(scheduledAtStr)

		meeting := map[string]interface{}{
			"id":           id,
			"chama_id":     chamaID.String,
			"title":        title.String,
			"description":  description.String,
			"scheduled_at": scheduledAt,
			"status":       status.String,
		}

		meetings = append(meetings, meeting)
	}

	return meetings, nil
}

// getUserDocuments gets user's document history
func (gds *GoogleDriveService) getUserDocuments(userID string) ([]map[string]interface{}, error) {
	// This is a placeholder - implement based on your document storage system
	return []map[string]interface{}{}, nil
}

// getUserSettings gets user's settings and preferences
func (gds *GoogleDriveService) getUserSettings(userID string) (map[string]interface{}, error) {
	// Get notification preferences
	notificationQuery := `
		SELECT sound_enabled, system_notifications, chama_notifications,
		       transaction_notifications, marketing_notifications, vibration_enabled,
		       volume_level, notification_sound_id
		FROM user_notification_preferences
		WHERE user_id = ?
	`

	settings := make(map[string]interface{})
	var soundEnabled, systemNotifications, chamaNotifications bool
	var transactionNotifications, marketingNotifications, vibrationEnabled bool
	var volumeLevel, notificationSoundID sql.NullInt64

	err := gds.db.QueryRow(notificationQuery, userID).Scan(
		&soundEnabled, &systemNotifications, &chamaNotifications,
		&transactionNotifications, &marketingNotifications, &vibrationEnabled,
		&volumeLevel, &notificationSoundID,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	settings["notifications"] = map[string]interface{}{
		"sound_enabled":             soundEnabled,
		"system_notifications":      systemNotifications,
		"chama_notifications":       chamaNotifications,
		"transaction_notifications": transactionNotifications,
		"marketing_notifications":   marketingNotifications,
		"vibration_enabled":         vibrationEnabled,
		"volume_level":              volumeLevel.Int64,
		"notification_sound_id":     notificationSoundID.Int64,
	}

	return settings, nil
}

// getOrCreateBackupFolder creates or gets the VaultKe backup folder in Google Drive
func (gds *GoogleDriveService) getOrCreateBackupFolder(service *drive.Service) (string, error) {
	// Search for existing VaultKe backup folder
	query := "name='VaultKe Backups' and mimeType='application/vnd.google-apps.folder' and trashed=false"
	fmt.Printf("🔍 Searching for existing backup folder...\n")
	fileList, err := service.Files.List().Q(query).Do()
	if err != nil {
		fmt.Printf("❌ Failed to search for backup folder: %v\n", err)
		return "", fmt.Errorf("failed to search for backup folder: %v", err)
	}

	// If folder exists, return its ID
	if len(fileList.Files) > 0 {
		fmt.Printf("✅ Found existing backup folder: %s\n", fileList.Files[0].Id)
		return fileList.Files[0].Id, nil
	}

	// Create new backup folder
	fmt.Printf("📁 Creating new backup folder...\n")
	folder := &drive.File{
		Name:     "VaultKe Backups",
		MimeType: "application/vnd.google-apps.folder",
	}

	createdFolder, err := service.Files.Create(folder).Do()
	if err != nil {
		fmt.Printf("❌ Failed to create backup folder: %v\n", err)
		return "", fmt.Errorf("failed to create backup folder: %v", err)
	}

	fmt.Printf("✅ Created new backup folder: %s\n", createdFolder.Id)
	return createdFolder.Id, nil
}

// recordBackup records backup information in the database
func (gds *GoogleDriveService) recordBackup(userID, fileName string, fileSize int64) error {
	// Tables are ensured to exist by calling methods

	// Insert backup record
	query := `
		INSERT INTO google_drive_backups (user_id, file_name, file_size)
		VALUES (?, ?, ?)
	`
	_, err := gds.db.Exec(query, userID, fileName, fileSize)
	if err != nil {
		return fmt.Errorf("failed to record backup: %v", err)
	}

	return nil
}

// RestoreUserBackup restores user data from Google Drive backup
func (gds *GoogleDriveService) RestoreUserBackup(userID string) (*RestoreResult, error) {
	// This is a complex operation that would need careful implementation
	// For now, return a placeholder
	return &RestoreResult{
		RestoredItems: 0,
		Timestamp:     time.Now(),
	}, fmt.Errorf("restore functionality not yet implemented")
}

// GetUserBackupInfo gets backup information for a user
func (gds *GoogleDriveService) GetUserBackupInfo(userID string) (*BackupInfo, error) {
	// Ensure tables exist
	err := gds.ensureTablesExist()
	if err != nil {
		return nil, fmt.Errorf("failed to ensure tables exist: %v", err)
	}

	// Check if user is connected
	connected, err := gds.IsUserConnected(userID)
	if err != nil {
		return nil, err
	}

	info := &BackupInfo{
		Connected: connected,
	}

	// If not connected, return basic info
	if !connected {
		return info, nil
	}

	// Get backup statistics
	query := `
		SELECT COUNT(*), COALESCE(MAX(backup_date), '1970-01-01'), COALESCE(SUM(file_size), 0)
		FROM google_drive_backups
		WHERE user_id = ?
	`

	var lastBackupStr string
	err = gds.db.QueryRow(query, userID).Scan(&info.BackupCount, &lastBackupStr, &info.BackupSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup info: %v", err)
	}

	// Parse the last backup date string
	if lastBackupStr != "" && lastBackupStr != "1970-01-01" {
		if lastBackup, err := parseTimeString(lastBackupStr); err == nil {
			info.LastBackup = lastBackup
		} else {
			// If parsing fails, log the error but don't fail the entire request
			fmt.Printf("Warning: failed to parse backup date %s: %v\n", lastBackupStr, err)
		}
	}

	return info, nil
}

// parseTimeString is a helper function to parse time strings from SQLite
func parseTimeString(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	// Try multiple formats that SQLite might use
	formats := []string{
		time.RFC3339Nano,           // 2025-07-25T05:57:13.742311077+03:00
		time.RFC3339,               // 2025-07-25T05:57:13+03:00
		"2006-01-02 15:04:05.999999999-07:00", // SQLite with timezone
		"2006-01-02 15:04:05.999999999",       // SQLite with nanoseconds
		"2006-01-02 15:04:05",                 // Standard format
		"2006-01-02",                          // Date only
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time string: %s", timeStr)
}

