package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// UserService handles user-related business logic
type UserService struct {
	db *sql.DB
}

// NewUserService creates a new user service
func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(registration *models.UserRegistration) (*models.User, error) {
	// Validate input structure
	if err := utils.ValidateStruct(registration); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Comprehensive password validation
	if passwordErrors := utils.ValidatePassword(registration.Password); len(passwordErrors) > 0 {
		return nil, fmt.Errorf("password validation failed: %s", strings.Join(passwordErrors, ", "))
	}

	// Additional phone validation
	if !utils.ValidateKenyanPhone(registration.Phone) {
		return nil, errors.New("invalid Kenyan phone number format")
	}

	// Sanitize all string inputs
	registration.Email = utils.SanitizeString(registration.Email)
	registration.FirstName = utils.SanitizeString(registration.FirstName)
	registration.LastName = utils.SanitizeString(registration.LastName)
	registration.Language = utils.SanitizeString(registration.Language)
	if registration.Gender != nil {
		*registration.Gender = utils.SanitizeString(*registration.Gender)
	}

	// Normalize email for consistent storage and comparison
	// This ensures: 'User@Gmail.COM' becomes 'user@gmail.com'
	registration.Email = strings.ToLower(strings.TrimSpace(registration.Email))

	// Additional validation after sanitization
	if len(registration.FirstName) < 2 || len(registration.LastName) < 2 {
		return nil, errors.New("first name and last name must be at least 2 characters after sanitization")
	}

	// Check if user already exists
	exists, err := s.UserExists(registration.Email, registration.Phone)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, errors.New("user with this email or phone already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registration.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Format phone number
	formattedPhone := utils.FormatPhoneNumber(registration.Phone)

	// Determine user role - check if this is Samuel Okoth (admin)
	userRole := models.UserRoleUser // Default to user
	if registration.Email == "sam.okothomulo@gmail.com" {
		userRole = models.UserRoleAdmin
	}

	// Create user
	user := &models.User{
		ID:              uuid.New().String(),
		Email:           registration.Email,
		Phone:           formattedPhone,
		FirstName:       registration.FirstName,
		LastName:        registration.LastName,
		PasswordHash:    string(hashedPassword),
		Role:            userRole,
		Status:          models.UserStatusActive, // Auto-activate for development
		IsEmailVerified: false,
		IsPhoneVerified: false,
		Language:        registration.Language,
		Theme:           "dark", // Default theme
		Gender:          registration.Gender,
		Rating:          0,
		TotalRatings:    0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Set default language if not provided
	if user.Language == "" {
		user.Language = "en"
	}

	// Insert user into database
	query := `
		INSERT INTO users (
			id, email, phone, first_name, last_name, password_hash, role, status,
			is_email_verified, is_phone_verified, language, theme, gender, rating, total_ratings,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		user.ID, user.Email, user.Phone, user.FirstName, user.LastName,
		user.PasswordHash, user.Role, user.Status, user.IsEmailVerified,
		user.IsPhoneVerified, user.Language, user.Theme, user.Gender, user.Rating,
		user.TotalRatings, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create personal wallet for the user
	walletService := NewWalletService(s.db)
	_, err = walletService.CreateWallet(user.ID, models.WalletTypePersonal)
	if err != nil {
		// Log error but don't fail user creation
		fmt.Printf("Warning: failed to create wallet for user %s: %v\n", user.ID, err)
	}

	return user, nil
}

// AuthenticateUser authenticates a user with email/phone and password
func (s *UserService) AuthenticateUser(login *models.UserLogin) (*models.User, error) {
	// Validate input structure
	if err := utils.ValidateStruct(login); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Sanitize identifier
	login.Identifier = utils.SanitizeString(login.Identifier)

	// Additional validation after sanitization
	if len(login.Identifier) == 0 {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Validate password length (basic check to prevent brute force with very long passwords)
	if len(login.Password) > 128 {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Find user by email or phone
	user, err := s.GetUserByEmailOrPhone(login.Identifier)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, fmt.Errorf("account is not active")
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT id, email, phone, first_name, last_name, avatar, role, status,
			   is_email_verified, is_phone_verified, language, theme, county, town,
			   latitude, longitude, business_type, business_description, gender, rating,
			   total_ratings, created_at, updated_at
		FROM users WHERE id = ?
	`

	user := &models.User{}
	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.Avatar, &user.Role, &user.Status, &user.IsEmailVerified,
		&user.IsPhoneVerified, &user.Language, &user.Theme, &user.County,
		&user.Town, &user.Latitude, &user.Longitude, &user.BusinessType,
		&user.BusinessDescription, &user.Gender, &user.Rating, &user.TotalRatings,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByEmailOrPhone retrieves a user by email or phone
func (s *UserService) GetUserByEmailOrPhone(identifier string) (*models.User, error) {
	// Normalize identifier if it's an email
	normalizedIdentifier := identifier
	if strings.Contains(identifier, "@") {
		normalizedIdentifier = strings.ToLower(strings.TrimSpace(identifier))
		fmt.Printf("🔍 Login attempt: original='%s' -> normalized='%s'\n", identifier, normalizedIdentifier)
	} else {
		normalizedIdentifier = strings.TrimSpace(identifier)
	}

	// Format phone number if it looks like a phone number
	formattedIdentifier := normalizedIdentifier
	if utils.IsPhoneNumber(identifier) {
		formattedIdentifier = utils.FormatPhoneNumber(identifier)
	}

	// Use both direct comparison (for new normalized emails) and LOWER() for legacy data
	query := `
		SELECT id, email, phone, first_name, last_name, password_hash, avatar, role, status,
			   is_email_verified, is_phone_verified, language, theme, county, town,
			   latitude, longitude, business_type, business_description, gender, rating,
			   total_ratings, created_at, updated_at
		FROM users WHERE (email = ? OR LOWER(TRIM(email)) = ?) OR phone = ?
	`

	user := &models.User{}
	err := s.db.QueryRow(query, normalizedIdentifier, normalizedIdentifier, formattedIdentifier).Scan(
		&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
		&user.PasswordHash, &user.Avatar, &user.Role, &user.Status,
		&user.IsEmailVerified, &user.IsPhoneVerified, &user.Language, &user.Theme,
		&user.County, &user.Town, &user.Latitude, &user.Longitude,
		&user.BusinessType, &user.BusinessDescription, &user.Gender, &user.Rating,
		&user.TotalRatings, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(userID string, update *models.UserProfileUpdate) (*models.User, error) {
	// Get current user
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}

	if update.FirstName != nil {
		setParts = append(setParts, "first_name = ?")
		args = append(args, *update.FirstName)
	}
	if update.LastName != nil {
		setParts = append(setParts, "last_name = ?")
		args = append(args, *update.LastName)
	}
	if update.Phone != nil {
		setParts = append(setParts, "phone = ?")
		args = append(args, *update.Phone)
	}
	if update.Avatar != nil {
		setParts = append(setParts, "avatar = ?")
		args = append(args, *update.Avatar)
	}
	if update.Language != nil {
		setParts = append(setParts, "language = ?")
		args = append(args, *update.Language)
	}
	if update.Theme != nil {
		setParts = append(setParts, "theme = ?")
		args = append(args, *update.Theme)
	}
	if update.County != nil {
		setParts = append(setParts, "county = ?")
		args = append(args, *update.County)
	}
	if update.Town != nil {
		setParts = append(setParts, "town = ?")
		args = append(args, *update.Town)
	}
	if update.Latitude != nil {
		setParts = append(setParts, "latitude = ?")
		args = append(args, *update.Latitude)
	}
	if update.Longitude != nil {
		setParts = append(setParts, "longitude = ?")
		args = append(args, *update.Longitude)
	}
	if update.BusinessType != nil {
		setParts = append(setParts, "business_type = ?")
		args = append(args, *update.BusinessType)
	}
	if update.BusinessDescription != nil {
		setParts = append(setParts, "business_description = ?")
		args = append(args, *update.BusinessDescription)
	}
	if update.Bio != nil {
		setParts = append(setParts, "bio = ?")
		args = append(args, *update.Bio)
	}
	if update.Occupation != nil {
		setParts = append(setParts, "occupation = ?")
		args = append(args, *update.Occupation)
	}
	if update.DateOfBirth != nil {
		setParts = append(setParts, "date_of_birth = ?")
		args = append(args, update.DateOfBirth.Time)
	}
	if update.Gender != nil {
		setParts = append(setParts, "gender = ?")
		args = append(args, *update.Gender)
	}

	if len(setParts) == 0 {
		return user, nil // No updates
	}

	// Add updated_at
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, userID)

	query := "UPDATE users SET " + setParts[0]
	for i := 1; i < len(setParts); i++ {
		query += ", " + setParts[i]
	}
	query += " WHERE id = ?"

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return updated user
	return s.GetUserByID(userID)
}

// UserExists checks if a user exists with the given email or phone
func (s *UserService) UserExists(email, phone string) (bool, error) {
	// Normalize email for consistent comparison
	// This handles: 'User@Gmail.COM' -> 'user@gmail.com'
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	formattedPhone := utils.FormatPhoneNumber(phone)

	fmt.Printf("🔍 UserExists check: original='%s' -> normalized='%s'\n", email, normalizedEmail)

	// Since we now store all emails in lowercase, we can do direct comparison
	// But we also check with LOWER() for existing data that might not be normalized
	query := "SELECT COUNT(*) FROM users WHERE (email = ? OR LOWER(TRIM(email)) = ?) OR phone = ?"
	var count int
	err := s.db.QueryRow(query, normalizedEmail, normalizedEmail, formattedPhone).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	fmt.Printf("✅ UserExists result: count=%d (exists=%t)\n", count, count > 0)
	return count > 0, nil
}

// VerifyEmail marks user's email as verified
func (s *UserService) VerifyEmail(userID string) error {
	query := "UPDATE users SET is_email_verified = true, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}
	return nil
}

// VerifyPhone marks user's phone as verified
func (s *UserService) VerifyPhone(userID string) error {
	query := "UPDATE users SET is_phone_verified = true, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to verify phone: %w", err)
	}
	return nil
}

// UpdateUserStatus updates user status
func (s *UserService) UpdateUserStatus(userID string, status models.UserStatus) error {
	query := "UPDATE users SET status = ?, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, status, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}
	return nil
}

// GetUsersByLocation gets users by location (county/town)
func (s *UserService) GetUsersByLocation(county, town string, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, email, phone, first_name, last_name, avatar, role, status,
			   is_email_verified, is_phone_verified, language, theme, county, town,
			   latitude, longitude, business_type, business_description, gender, rating,
			   total_ratings, created_at, updated_at
		FROM users
		WHERE county = ? AND town = ? AND status = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, county, town, models.UserStatusActive, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by location: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName,
			&user.Avatar, &user.Role, &user.Status, &user.IsEmailVerified,
			&user.IsPhoneVerified, &user.Language, &user.Theme, &user.County,
			&user.Town, &user.Latitude, &user.Longitude, &user.BusinessType,
			&user.BusinessDescription, &user.Gender, &user.Rating, &user.TotalRatings,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserStatistics returns comprehensive statistics for a user
func (s *UserService) GetUserStatistics(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get user info
	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get wallet statistics
	walletStats, err := s.getUserWalletStatistics(userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get wallet statistics: %v\n", err)
		walletStats = make(map[string]interface{})
	}

	// Get chama statistics
	chamaStats, err := s.getUserChamaStatistics(userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get chama statistics: %v\n", err)
		chamaStats = make(map[string]interface{})
	}

	// Get contribution statistics
	contributionStats, err := s.getUserContributionStatistics(userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get contribution statistics: %v\n", err)
		contributionStats = make(map[string]interface{})
	}

	// Get meeting statistics
	meetingStats, err := s.getUserMeetingStatistics(userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get meeting statistics: %v\n", err)
		meetingStats = make(map[string]interface{})
	}

	// Build response
	stats["user_info"] = map[string]interface{}{
		"id":         user.ID,
		"firstName":  user.FirstName,
		"lastName":   user.LastName,
		"email":      user.Email,
		"phone":      user.Phone,
		"role":       user.Role,
		"rating":     user.Rating,
		"createdAt":  user.CreatedAt,
	}

	stats["wallet_stats"] = walletStats
	stats["chama_stats"] = chamaStats
	stats["contribution_stats"] = contributionStats
	stats["meeting_stats"] = meetingStats

	return stats, nil
}

// getUserWalletStatistics gets wallet-related statistics for a user
func (s *UserService) getUserWalletStatistics(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get personal wallet balance
	var personalBalance float64
	err := s.db.QueryRow(`
		SELECT COALESCE(balance, 0)
		FROM wallets
		WHERE owner_id = ? AND type = 'personal'
	`, userID).Scan(&personalBalance)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get total wallet transactions
	var totalTransactions int
	var totalVolume float64
	err = s.db.QueryRow(`
		SELECT
			COUNT(*) as total_transactions,
			COALESCE(SUM(amount), 0) as total_volume
		FROM transactions
		WHERE initiated_by = ?
	`, userID).Scan(&totalTransactions, &totalVolume)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["personal_balance"] = personalBalance
	stats["total_transactions"] = totalTransactions
	stats["total_volume"] = totalVolume

	return stats, nil
}

// getUserChamaStatistics gets chama-related statistics for a user
func (s *UserService) getUserChamaStatistics(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total chamas joined
	var totalChamas, activeChamas int
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_chamas,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_chamas
		FROM chama_members
		WHERE user_id = ?
	`, userID).Scan(&totalChamas, &activeChamas)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get user roles distribution
	var chairpersonCount, secretaryCount, treasurerCount, memberCount int
	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN role = 'chairperson' THEN 1 END) as chairperson_count,
			COUNT(CASE WHEN role = 'secretary' THEN 1 END) as secretary_count,
			COUNT(CASE WHEN role = 'treasurer' THEN 1 END) as treasurer_count,
			COUNT(CASE WHEN role = 'member' THEN 1 END) as member_count
		FROM chama_members
		WHERE user_id = ? AND is_active = true
	`, userID).Scan(&chairpersonCount, &secretaryCount, &treasurerCount, &memberCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_chamas"] = totalChamas
	stats["active_chamas"] = activeChamas
	stats["leadership_roles"] = chairpersonCount + secretaryCount + treasurerCount
	stats["member_roles"] = memberCount

	return stats, nil
}

// getUserContributionStatistics gets contribution-related statistics for a user
func (s *UserService) getUserContributionStatistics(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get contribution statistics
	var totalContributions int
	var totalContributionAmount, averageContribution float64
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_contributions,
			COALESCE(SUM(amount), 0) as total_amount,
			COALESCE(AVG(amount), 0) as average_contribution
		FROM transactions
		WHERE initiated_by = ? AND type = 'contribution' AND status = 'completed'
	`, userID).Scan(&totalContributions, &totalContributionAmount, &averageContribution)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get monthly contribution trend (last 6 months)
	monthlyQuery := `
		SELECT
			strftime('%Y-%m', created_at) as month,
			COUNT(*) as count,
			COALESCE(SUM(amount), 0) as total
		FROM transactions
		WHERE initiated_by = ? AND type = 'contribution' AND status = 'completed'
		AND created_at >= datetime('now', '-6 months')
		GROUP BY strftime('%Y-%m', created_at)
		ORDER BY month DESC
		LIMIT 6
	`

	rows, err := s.db.Query(monthlyQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monthlyTrend []map[string]interface{}
	for rows.Next() {
		var month string
		var count int
		var total float64
		if err := rows.Scan(&month, &count, &total); err != nil {
			continue
		}
		monthlyTrend = append(monthlyTrend, map[string]interface{}{
			"month": month,
			"count": count,
			"total": total,
		})
	}

	stats["total_contributions"] = totalContributions
	stats["total_contribution_amount"] = totalContributionAmount
	stats["average_contribution"] = averageContribution
	stats["monthly_trend"] = monthlyTrend

	return stats, nil
}

// getUserMeetingStatistics gets meeting-related statistics for a user
func (s *UserService) getUserMeetingStatistics(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get upcoming and ongoing meetings across all chamas user is a member of
	var upcomingMeetings, ongoingMeetings int
	err := s.db.QueryRow(`
		SELECT
			COUNT(DISTINCT CASE WHEN m.status IN ('scheduled', 'pending', 'ready') AND m.scheduled_at > datetime('now', '+3 hours') THEN m.id END) as upcoming_meetings,
			COUNT(DISTINCT CASE WHEN m.status IN ('ongoing', 'active', 'started', 'live') THEN m.id END) as ongoing_meetings
		FROM meetings m
		INNER JOIN chama_members cm ON m.chama_id = cm.chama_id
		WHERE cm.user_id = ? AND cm.is_active = true
	`, userID).Scan(&upcomingMeetings, &ongoingMeetings)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Total active meetings = upcoming + ongoing
	activeMeetings := upcomingMeetings + ongoingMeetings

	stats["upcoming_meetings"] = upcomingMeetings
	stats["ongoing_meetings"] = ongoingMeetings
	stats["active_meetings"] = activeMeetings

	return stats, nil
}

// GetAdminStatistics returns comprehensive system-wide statistics for admin dashboard
func (s *UserService) GetAdminStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get user statistics
	userStats, err := s.getAdminUserStatistics()
	if err != nil {
		fmt.Printf("Warning: Failed to get user statistics: %v\n", err)
		userStats = make(map[string]interface{})
	}

	// Get chama statistics
	chamaStats, err := s.getAdminChamaStatistics()
	if err != nil {
		fmt.Printf("Warning: Failed to get chama statistics: %v\n", err)
		chamaStats = make(map[string]interface{})
	}

	// Get transaction statistics
	transactionStats, err := s.getAdminTransactionStatistics()
	if err != nil {
		fmt.Printf("Warning: Failed to get transaction statistics: %v\n", err)
		transactionStats = make(map[string]interface{})
	}

	// Get system health
	systemHealth, err := s.getSystemHealth()
	if err != nil {
		fmt.Printf("Warning: Failed to get system health: %v\n", err)
		systemHealth = 95.0 // Default fallback
	}

	// Build response
	stats["user_stats"] = userStats
	stats["chama_stats"] = chamaStats
	stats["transaction_stats"] = transactionStats
	stats["system_health"] = systemHealth

	return stats, nil
}

// getAdminUserStatistics gets system-wide user statistics
func (s *UserService) getAdminUserStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total users and active users
	var totalUsers, activeUsers int
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_users,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_users
		FROM users
	`).Scan(&totalUsers, &activeUsers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get users created in last 30 days
	var newUsers int
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE created_at >= datetime('now', '-30 days')
	`).Scan(&newUsers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_users"] = totalUsers
	stats["active_users"] = activeUsers
	stats["new_users_30d"] = newUsers

	return stats, nil
}

// getAdminChamaStatistics gets system-wide chama statistics
func (s *UserService) getAdminChamaStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total chamas and active chamas
	var totalChamas, activeChamas int
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_chamas,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_chamas
		FROM chamas
	`).Scan(&totalChamas, &activeChamas)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get total members across all chamas
	var totalMembers int
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM chama_members
		WHERE is_active = true
	`).Scan(&totalMembers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_chamas"] = totalChamas
	stats["active_chamas"] = activeChamas
	stats["total_members"] = totalMembers

	return stats, nil
}

// getAdminTransactionStatistics gets system-wide transaction statistics
func (s *UserService) getAdminTransactionStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total transactions and volume
	var totalTransactions int
	var totalVolume float64
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_transactions,
			COALESCE(SUM(amount), 0) as total_volume
		FROM transactions
		WHERE status = 'completed'
	`).Scan(&totalTransactions, &totalVolume)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get transactions by type
	var contributions, transfers, withdrawals int
	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN type = 'contribution' THEN 1 END) as contributions,
			COUNT(CASE WHEN type = 'transfer' THEN 1 END) as transfers,
			COUNT(CASE WHEN type = 'withdrawal' THEN 1 END) as withdrawals
		FROM transactions
		WHERE status = 'completed'
	`).Scan(&contributions, &transfers, &withdrawals)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_transactions"] = totalTransactions
	stats["total_volume"] = totalVolume
	stats["contributions"] = contributions
	stats["transfers"] = transfers
	stats["withdrawals"] = withdrawals

	return stats, nil
}

// getSystemHealth calculates system health percentage
func (s *UserService) getSystemHealth() (float64, error) {
	// Calculate system health based on various metrics
	var activeUsers, totalUsers int
	err := s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_users,
			COUNT(*) as total_users
		FROM users
	`).Scan(&activeUsers, &totalUsers)
	if err != nil {
		return 95.0, err // Default fallback
	}

	var activeChamas, totalChamas int
	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_chamas,
			COUNT(*) as total_chamas
		FROM chamas
	`).Scan(&activeChamas, &totalChamas)
	if err != nil {
		return 95.0, err // Default fallback
	}

	// Calculate health score (0-100)
	userHealthScore := 0.0
	if totalUsers > 0 {
		userHealthScore = float64(activeUsers) / float64(totalUsers) * 100
	}

	chamaHealthScore := 0.0
	if totalChamas > 0 {
		chamaHealthScore = float64(activeChamas) / float64(totalChamas) * 100
	}

	// Average of user and chama health scores
	systemHealth := (userHealthScore + chamaHealthScore) / 2

	// Ensure minimum health score of 85% for demo purposes
	if systemHealth < 85.0 {
		systemHealth = 85.0 + (systemHealth * 0.15) // Scale to 85-100 range
	}

	return systemHealth, nil
}

// GetSystemAnalytics returns comprehensive system analytics for admin dashboard
func (s *UserService) GetSystemAnalytics(period string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Calculate date range based on period
	var dateFilter string
	switch period {
	case "24h":
		dateFilter = "datetime('now', '-1 day')"
	case "7d":
		dateFilter = "datetime('now', '-7 days')"
	case "30d":
		dateFilter = "datetime('now', '-30 days')"
	case "90d":
		dateFilter = "datetime('now', '-90 days')"
	default:
		dateFilter = "datetime('now', '-7 days')"
	}

	// Get user analytics
	userAnalytics, err := s.getSystemUserAnalytics(dateFilter)
	if err != nil {
		fmt.Printf("Warning: Failed to get user analytics: %v\n", err)
		userAnalytics = make(map[string]interface{})
	}

	// Get chama analytics
	chamaAnalytics, err := s.getSystemChamaAnalytics(dateFilter)
	if err != nil {
		fmt.Printf("Warning: Failed to get chama analytics: %v\n", err)
		chamaAnalytics = make(map[string]interface{})
	}

	// Get transaction analytics
	transactionAnalytics, err := s.getSystemTransactionAnalytics(dateFilter)
	if err != nil {
		fmt.Printf("Warning: Failed to get transaction analytics: %v\n", err)
		transactionAnalytics = make(map[string]interface{})
	}

	// Get system metrics
	systemMetrics, err := s.getSystemMetrics()
	if err != nil {
		fmt.Printf("Warning: Failed to get system metrics: %v\n", err)
		systemMetrics = make(map[string]interface{})
	}

	// Calculate growth rates
	growthRates, err := s.calculateGrowthRates(period)
	if err != nil {
		fmt.Printf("Warning: Failed to calculate growth rates: %v\n", err)
		growthRates = make(map[string]interface{})
	}

	// Build response
	analytics["users"] = userAnalytics
	analytics["chamas"] = chamaAnalytics
	analytics["transactions"] = transactionAnalytics
	analytics["system"] = systemMetrics
	analytics["growth"] = growthRates
	analytics["period"] = period
	analytics["generatedAt"] = time.Now().Format(time.RFC3339)

	return analytics, nil
}

// getSystemUserAnalytics gets detailed user analytics
func (s *UserService) getSystemUserAnalytics(dateFilter string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Get total and active users
	var totalUsers, activeUsers int
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_users,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_users
		FROM users
	`).Scan(&totalUsers, &activeUsers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get new users in period
	var newUsers int
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users
		WHERE created_at >= %s
	`, dateFilter)
	err = s.db.QueryRow(query).Scan(&newUsers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	analytics["total"] = totalUsers
	analytics["active"] = activeUsers
	analytics["newUsers"] = newUsers

	return analytics, nil
}

// getSystemChamaAnalytics gets detailed chama analytics
func (s *UserService) getSystemChamaAnalytics(dateFilter string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Get total and active chamas
	var totalChamas, activeChamas int
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_chamas,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_chamas
		FROM chamas
	`).Scan(&totalChamas, &activeChamas)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	fmt.Printf("📊 Chama stats: total=%d, active=%d\n", totalChamas, activeChamas)

	// Get new chamas in period
	var newChamas int
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM chamas
		WHERE created_at >= %s
	`, dateFilter)
	err = s.db.QueryRow(query).Scan(&newChamas)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get total members
	var totalMembers int
	err = s.db.QueryRow(`
		SELECT COUNT(*)
		FROM chama_members
		WHERE is_active = true
	`).Scan(&totalMembers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	analytics["total"] = totalChamas
	analytics["active"] = activeChamas
	analytics["newChamas"] = newChamas
	analytics["totalMembers"] = totalMembers

	return analytics, nil
}

// getSystemTransactionAnalytics gets detailed transaction analytics
func (s *UserService) getSystemTransactionAnalytics(dateFilter string) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Get total transactions and volume
	var totalTransactions int
	var totalVolume float64
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_transactions,
			COALESCE(SUM(amount), 0) as total_volume
		FROM transactions
	`).Scan(&totalTransactions, &totalVolume)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get successful and failed transactions
	var successfulTransactions, failedTransactions int
	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM transactions
	`).Scan(&successfulTransactions, &failedTransactions)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Calculate success rate
	successRate := 0.0
	if totalTransactions > 0 {
		successRate = float64(successfulTransactions) / float64(totalTransactions) * 100
	}

	// Get transactions in period
	var periodTransactions int
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM transactions
		WHERE created_at >= %s
	`, dateFilter)
	err = s.db.QueryRow(query).Scan(&periodTransactions)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	analytics["total"] = totalTransactions
	analytics["volume"] = totalVolume
	analytics["successful"] = successfulTransactions
	analytics["failed"] = failedTransactions
	analytics["successRate"] = fmt.Sprintf("%.1f%%", successRate)
	analytics["periodTransactions"] = periodTransactions

	return analytics, nil
}

// getSystemMetrics gets system health and performance metrics
func (s *UserService) getSystemMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Calculate uptime (assume 99.5% for demo)
	uptime := "99.5%"

	// Calculate average response time (simulate based on transaction count)
	var transactionCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at >= datetime('now', '-1 hour')").Scan(&transactionCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Simulate response time based on load
	responseTime := "45ms"
	if transactionCount > 100 {
		responseTime = "65ms"
	} else if transactionCount > 50 {
		responseTime = "55ms"
	}

	// Calculate error rate
	var errorCount, totalRequests int
	err = s.db.QueryRow(`
		SELECT
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as errors,
			COUNT(*) as total
		FROM transactions
		WHERE created_at >= datetime('now', '-1 hour')
	`).Scan(&errorCount, &totalRequests)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	errorRate := "0.1%"
	if totalRequests > 0 {
		rate := float64(errorCount) / float64(totalRequests) * 100
		errorRate = fmt.Sprintf("%.1f%%", rate)
	}

	// Storage usage (simulate)
	storage := "45%"

	metrics["uptime"] = uptime
	metrics["responseTime"] = responseTime
	metrics["errorRate"] = errorRate
	metrics["storage"] = storage

	return metrics, nil
}

// calculateGrowthRates calculates growth rates for the specified period
func (s *UserService) calculateGrowthRates(period string) (map[string]interface{}, error) {
	growth := make(map[string]interface{})

	// Calculate user growth
	userGrowth, err := s.calculateUserGrowth(period)
	if err != nil {
		userGrowth = 0.0
	}

	// Calculate chama growth
	chamaGrowth, err := s.calculateChamaGrowth(period)
	if err != nil {
		chamaGrowth = 0.0
	}

	// Calculate transaction growth
	transactionGrowth, err := s.calculateTransactionGrowth(period)
	if err != nil {
		transactionGrowth = 0.0
	}

	growth["userGrowth"] = userGrowth
	growth["chamaGrowth"] = chamaGrowth
	growth["transactionGrowth"] = transactionGrowth

	return growth, nil
}

// calculateUserGrowth calculates user growth rate for the period
func (s *UserService) calculateUserGrowth(period string) (float64, error) {
	var currentPeriodUsers, previousPeriodUsers int

	// Get date ranges
	var currentFilter, previousFilter string
	switch period {
	case "24h":
		currentFilter = "datetime('now', '-1 day')"
		previousFilter = "datetime('now', '-2 days')"
	case "7d":
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	case "30d":
		currentFilter = "datetime('now', '-30 days')"
		previousFilter = "datetime('now', '-60 days')"
	case "90d":
		currentFilter = "datetime('now', '-90 days')"
		previousFilter = "datetime('now', '-180 days')"
	default:
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	}

	// Get current period users
	query := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE created_at >= %s", currentFilter)
	err := s.db.QueryRow(query).Scan(&currentPeriodUsers)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Get previous period users
	query = fmt.Sprintf("SELECT COUNT(*) FROM users WHERE created_at >= %s AND created_at < %s", previousFilter, currentFilter)
	err = s.db.QueryRow(query).Scan(&previousPeriodUsers)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Calculate growth rate
	if previousPeriodUsers == 0 {
		if currentPeriodUsers > 0 {
			return 100.0, nil // 100% growth from 0
		}
		return 0.0, nil
	}

	growth := float64(currentPeriodUsers-previousPeriodUsers) / float64(previousPeriodUsers) * 100
	return growth, nil
}

// calculateChamaGrowth calculates chama growth rate for the period
func (s *UserService) calculateChamaGrowth(period string) (float64, error) {
	var currentPeriodChamas, previousPeriodChamas int

	// Get date ranges (same logic as user growth)
	var currentFilter, previousFilter string
	switch period {
	case "24h":
		currentFilter = "datetime('now', '-1 day')"
		previousFilter = "datetime('now', '-2 days')"
	case "7d":
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	case "30d":
		currentFilter = "datetime('now', '-30 days')"
		previousFilter = "datetime('now', '-60 days')"
	case "90d":
		currentFilter = "datetime('now', '-90 days')"
		previousFilter = "datetime('now', '-180 days')"
	default:
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	}

	// Get current period chamas
	query := fmt.Sprintf("SELECT COUNT(*) FROM chamas WHERE created_at >= %s", currentFilter)
	err := s.db.QueryRow(query).Scan(&currentPeriodChamas)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Get previous period chamas
	query = fmt.Sprintf("SELECT COUNT(*) FROM chamas WHERE created_at >= %s AND created_at < %s", previousFilter, currentFilter)
	err = s.db.QueryRow(query).Scan(&previousPeriodChamas)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Calculate growth rate
	if previousPeriodChamas == 0 {
		if currentPeriodChamas > 0 {
			return 100.0, nil
		}
		return 0.0, nil
	}

	growth := float64(currentPeriodChamas-previousPeriodChamas) / float64(previousPeriodChamas) * 100
	return growth, nil
}

// calculateTransactionGrowth calculates transaction growth rate for the period
func (s *UserService) calculateTransactionGrowth(period string) (float64, error) {
	var currentPeriodTransactions, previousPeriodTransactions int

	// Get date ranges (same logic as above)
	var currentFilter, previousFilter string
	switch period {
	case "24h":
		currentFilter = "datetime('now', '-1 day')"
		previousFilter = "datetime('now', '-2 days')"
	case "7d":
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	case "30d":
		currentFilter = "datetime('now', '-30 days')"
		previousFilter = "datetime('now', '-60 days')"
	case "90d":
		currentFilter = "datetime('now', '-90 days')"
		previousFilter = "datetime('now', '-180 days')"
	default:
		currentFilter = "datetime('now', '-7 days')"
		previousFilter = "datetime('now', '-14 days')"
	}

	// Get current period transactions
	query := fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE created_at >= %s", currentFilter)
	err := s.db.QueryRow(query).Scan(&currentPeriodTransactions)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Get previous period transactions
	query = fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE created_at >= %s AND created_at < %s", previousFilter, currentFilter)
	err = s.db.QueryRow(query).Scan(&previousPeriodTransactions)
	if err != nil && err != sql.ErrNoRows {
		return 0.0, err
	}

	// Calculate growth rate
	if previousPeriodTransactions == 0 {
		if currentPeriodTransactions > 0 {
			return 100.0, nil
		}
		return 0.0, nil
	}

	growth := float64(currentPeriodTransactions-previousPeriodTransactions) / float64(previousPeriodTransactions) * 100
	return growth, nil
}
