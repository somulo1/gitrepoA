package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// ChamaService handles chama-related business logic
type ChamaService struct {
	db           *sql.DB
	emailService *EmailService
}

// Global email service instance to avoid recreating it
var globalEmailService *EmailService

// NewChamaService creates a new chama service
func NewChamaService(db *sql.DB) *ChamaService {
	// Use singleton pattern for email service to avoid recreating it
	if globalEmailService == nil {
		fmt.Printf("🔧 [CHAMA SERVICE] Creating new email service instance\n")
		globalEmailService = NewEmailService()
	} else {
		fmt.Printf("♻️  [CHAMA SERVICE] Reusing existing email service instance\n")
	}

	return &ChamaService{
		db:           db,
		emailService: globalEmailService,
	}
}

// GetEmailService returns the email service instance
func (s *ChamaService) GetEmailService() *EmailService {
	return s.emailService
}

// CreateChama creates a new chama
func (s *ChamaService) CreateChama(creation *models.ChamaCreation, createdBy string) (*models.Chama, error) {
	// Validate input
	if err := utils.ValidateStruct(creation); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Create chama
	chama := &models.Chama{
		ID:                    uuid.New().String(),
		Name:                  creation.Name,
		Description:           creation.Description,
		Category:              creation.Category,
		Type:                  creation.Type,
		Status:                models.ChamaStatusActive,
		County:                creation.County,
		Town:                  creation.Town,
		Latitude:              creation.Latitude,
		Longitude:             creation.Longitude,
		ContributionAmount:    creation.ContributionAmount,
		ContributionFrequency: creation.ContributionFrequency,
		TargetAmount:          creation.TargetAmount,
		TargetDeadline:        creation.TargetDeadline,
		PaymentMethod:         creation.PaymentMethod,
		TillNumber:            creation.TillNumber,
		PaybillBusinessNumber: creation.PaybillBusinessNumber,
		PaybillAccountNumber:  creation.PaybillAccountNumber,
		PaymentRecipientName:  creation.PaymentRecipientName,
		MaxMembers:            creation.MaxMembers,
		CurrentMembers:        1, // Creator is the first member
		TotalFunds:            0,
		IsPublic:              creation.IsPublic,
		RequiresApproval:      creation.RequiresApproval,
		Rules:                 creation.Rules,
		MeetingSchedule:       creation.MeetingSchedule,
		CreatedBy:             createdBy,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	// Serialize JSON fields
	rulesJSON, err := chama.GetRulesJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize rules: %w", err)
	}

	_, err = chama.GetMeetingScheduleJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize meeting schedule: %w", err)
	}

	// Start database transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert chama
	query := `
		INSERT INTO chamas (
			id, name, description, category, type, status, county, town, latitude, longitude,
			contribution_amount, contribution_frequency, target_amount, target_deadline,
			payment_method, till_number, paybill_business_number, paybill_account_number, payment_recipient_name,
			max_members, current_members, total_funds, is_public, requires_approval, rules,
			meeting_frequency, meeting_day_of_week, meeting_day_of_month, meeting_time,
			created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var meetingFreq, meetingTime *string
	var meetingDayOfWeek, meetingDayOfMonth *int

	if chama.MeetingSchedule != nil {
		meetingFreq = &chama.MeetingSchedule.Frequency
		meetingTime = &chama.MeetingSchedule.Time
		meetingDayOfWeek = chama.MeetingSchedule.DayOfWeek
		meetingDayOfMonth = chama.MeetingSchedule.DayOfMonth
	}

	_, err = tx.Exec(query,
		chama.ID, chama.Name, chama.Description, chama.Category, chama.Type, chama.Status,
		chama.County, chama.Town, chama.Latitude, chama.Longitude,
		chama.ContributionAmount, chama.ContributionFrequency, chama.TargetAmount, chama.TargetDeadline,
		chama.PaymentMethod, chama.TillNumber, chama.PaybillBusinessNumber, chama.PaybillAccountNumber, chama.PaymentRecipientName,
		chama.MaxMembers, chama.CurrentMembers, chama.TotalFunds, chama.IsPublic, chama.RequiresApproval,
		rulesJSON, meetingFreq, meetingDayOfWeek, meetingDayOfMonth, meetingTime,
		chama.CreatedBy, chama.CreatedAt, chama.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chama: %w", err)
	}

	// Add creator as chairperson
	member := &models.ChamaMember{
		ID:                 uuid.New().String(),
		ChamaID:            chama.ID,
		UserID:             createdBy,
		Role:               models.ChamaRoleChairperson,
		JoinedAt:           time.Now(),
		IsActive:           true,
		TotalContributions: 0,
		Rating:             0,
		TotalRatings:       0,
	}

	memberQuery := `
		INSERT INTO chama_members (
			id, chama_id, user_id, role, joined_at, is_active,
			total_contributions, rating, total_ratings
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(memberQuery,
		member.ID, member.ChamaID, member.UserID, member.Role, member.JoinedAt,
		member.IsActive, member.TotalContributions, member.Rating, member.TotalRatings,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator as member: %w", err)
	}

	// Create chama wallet within the same transaction
	walletService := NewWalletService(s.db)
	_, err = walletService.CreateWalletWithTx(tx, chama.ID, models.WalletTypeChama)
	if err != nil {
		return nil, fmt.Errorf("failed to create chama wallet: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chama, nil
}

// GetChamaByID retrieves a chama by ID
func (s *ChamaService) GetChamaByID(chamaID string) (*models.Chama, error) {
	query := `
		SELECT id, name, description, category, type, status, avatar, county, town,
			   latitude, longitude, contribution_amount, contribution_frequency,
			   max_members, current_members, total_funds, is_public, requires_approval,
			   rules, meeting_frequency, meeting_day_of_week, meeting_day_of_month,
			   meeting_time, permissions, created_by, created_at, updated_at
		FROM chamas WHERE id = ?
	`

	chama := &models.Chama{}
	var rulesJSON, permissionsJSON string
	var meetingFreq, meetingTime *string
	var meetingDayOfWeek, meetingDayOfMonth *int

	err := s.db.QueryRow(query, chamaID).Scan(
		&chama.ID, &chama.Name, &chama.Description, &chama.Category, &chama.Type, &chama.Status,
		&chama.Avatar, &chama.County, &chama.Town, &chama.Latitude, &chama.Longitude,
		&chama.ContributionAmount, &chama.ContributionFrequency, &chama.MaxMembers,
		&chama.CurrentMembers, &chama.TotalFunds, &chama.IsPublic, &chama.RequiresApproval,
		&rulesJSON, &meetingFreq, &meetingDayOfWeek, &meetingDayOfMonth, &meetingTime,
		&permissionsJSON, &chama.CreatedBy, &chama.CreatedAt, &chama.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chama not found")
		}
		return nil, fmt.Errorf("failed to get chama: %w", err)
	}

	// Parse JSON fields
	if err = chama.SetRulesFromJSON(rulesJSON); err != nil {
		return nil, fmt.Errorf("failed to parse rules: %w", err)
	}

	// Parse permissions JSON
	if permissionsJSON != "" {
		var permissions map[string]interface{}
		if err := json.Unmarshal([]byte(permissionsJSON), &permissions); err == nil {
			chama.Permissions = permissions
		}
	}
	// Set default permissions if none exist
	if chama.Permissions == nil {
		chama.Permissions = map[string]interface{}{
			"allowMerryGoRound": true,
			"allowWelfare":      true,
			"allowMarketplace":  true,
		}
	}

	// Reconstruct meeting schedule
	if meetingFreq != nil {
		chama.MeetingSchedule = &models.MeetingSchedule{
			Frequency:  *meetingFreq,
			DayOfWeek:  meetingDayOfWeek,
			DayOfMonth: meetingDayOfMonth,
			Time:       utils.DerefString(meetingTime),
		}
	}

	return chama, nil
}

// GetChamas retrieves all public chamas
func (s *ChamaService) GetChamas(limit, offset int) ([]*models.Chama, error) {
	query := `
		SELECT id, name, description, category, type, status, avatar, county, town,
			   latitude, longitude, contribution_amount, contribution_frequency,
			   max_members, current_members, total_funds, is_public, requires_approval,
			   rules, meeting_frequency, meeting_day_of_week, meeting_day_of_month,
			   meeting_time, created_by, created_at, updated_at
		FROM chamas
		WHERE is_public = true AND status = 'active'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chamas: %w", err)
	}
	defer rows.Close()

	var chamas []*models.Chama
	for rows.Next() {
		chama := &models.Chama{}
		var rulesJSON string
		var meetingFreq, meetingTime *string
		var meetingDayOfWeek, meetingDayOfMonth *int

		err := rows.Scan(
			&chama.ID, &chama.Name, &chama.Description, &chama.Category, &chama.Type, &chama.Status,
			&chama.Avatar, &chama.County, &chama.Town, &chama.Latitude, &chama.Longitude,
			&chama.ContributionAmount, &chama.ContributionFrequency, &chama.MaxMembers,
			&chama.CurrentMembers, &chama.TotalFunds, &chama.IsPublic, &chama.RequiresApproval,
			&rulesJSON, &meetingFreq, &meetingDayOfWeek, &meetingDayOfMonth, &meetingTime,
			&chama.CreatedBy, &chama.CreatedAt, &chama.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chama: %w", err)
		}

		// Parse JSON fields
		if err = chama.SetRulesFromJSON(rulesJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rules: %w", err)
		}

		// Reconstruct meeting schedule
		if meetingFreq != nil {
			chama.MeetingSchedule = &models.MeetingSchedule{
				Frequency:  *meetingFreq,
				DayOfWeek:  meetingDayOfWeek,
				DayOfMonth: meetingDayOfMonth,
				Time:       utils.DerefString(meetingTime),
			}
		}

		chamas = append(chamas, chama)
	}

	return chamas, nil
}

// GetAllChamasForAdmin retrieves all chamas for admin management (no filters)
func (s *ChamaService) GetAllChamasForAdmin(limit, offset int) ([]*models.Chama, error) {
	query := `
		SELECT id, name, description, type, status, avatar, county, town,
			   latitude, longitude, contribution_amount, contribution_frequency,
			   max_members, current_members, total_funds, is_public, requires_approval,
			   rules, meeting_frequency, meeting_day_of_week, meeting_day_of_month,
			   meeting_time, created_by, created_at, updated_at
		FROM chamas
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all chamas for admin: %w", err)
	}
	defer rows.Close()

	var chamas []*models.Chama
	for rows.Next() {
		chama := &models.Chama{}
		var rulesJSON string
		var meetingFreq, meetingTime *string
		var meetingDayOfWeek, meetingDayOfMonth *int

		err := rows.Scan(
			&chama.ID, &chama.Name, &chama.Description, &chama.Type, &chama.Status,
			&chama.Avatar, &chama.County, &chama.Town, &chama.Latitude, &chama.Longitude,
			&chama.ContributionAmount, &chama.ContributionFrequency, &chama.MaxMembers,
			&chama.CurrentMembers, &chama.TotalFunds, &chama.IsPublic, &chama.RequiresApproval,
			&rulesJSON, &meetingFreq, &meetingDayOfWeek, &meetingDayOfMonth, &meetingTime,
			&chama.CreatedBy, &chama.CreatedAt, &chama.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chama: %w", err)
		}

		// Parse JSON fields
		if err = chama.SetRulesFromJSON(rulesJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rules: %w", err)
		}

		// Reconstruct meeting schedule
		if meetingFreq != nil {
			chama.MeetingSchedule = &models.MeetingSchedule{
				Frequency:  *meetingFreq,
				DayOfWeek:  meetingDayOfWeek,
				DayOfMonth: meetingDayOfMonth,
				Time:       utils.DerefString(meetingTime),
			}
		}

		chamas = append(chamas, chama)
	}

	return chamas, nil
}

// GetChamasByUser retrieves chamas for a user
func (s *ChamaService) GetChamasByUser(userID string, limit, offset int) ([]*models.Chama, error) {
	query := `
		SELECT c.id, c.name, c.description, c.category, c.type, c.status, c.avatar, c.county, c.town,
			   c.latitude, c.longitude, c.contribution_amount, c.contribution_frequency,
			   c.max_members, c.current_members, c.total_funds, c.is_public, c.requires_approval,
			   c.rules, c.meeting_frequency, c.meeting_day_of_week, c.meeting_day_of_month,
			   c.meeting_time, c.created_by, c.created_at, c.updated_at
		FROM chamas c
		INNER JOIN chama_members cm ON c.id = cm.chama_id
		WHERE cm.user_id = ? AND cm.is_active = true
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user chamas: %w", err)
	}
	defer rows.Close()

	var chamas []*models.Chama
	for rows.Next() {
		chama := &models.Chama{}
		var rulesJSON string
		var meetingFreq, meetingTime *string
		var meetingDayOfWeek, meetingDayOfMonth *int

		err := rows.Scan(
			&chama.ID, &chama.Name, &chama.Description, &chama.Category, &chama.Type, &chama.Status,
			&chama.Avatar, &chama.County, &chama.Town, &chama.Latitude, &chama.Longitude,
			&chama.ContributionAmount, &chama.ContributionFrequency, &chama.MaxMembers,
			&chama.CurrentMembers, &chama.TotalFunds, &chama.IsPublic, &chama.RequiresApproval,
			&rulesJSON, &meetingFreq, &meetingDayOfWeek, &meetingDayOfMonth, &meetingTime,
			&chama.CreatedBy, &chama.CreatedAt, &chama.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chama: %w", err)
		}

		// Parse JSON fields
		if err = chama.SetRulesFromJSON(rulesJSON); err != nil {
			return nil, fmt.Errorf("failed to parse rules: %w", err)
		}

		// Reconstruct meeting schedule
		if meetingFreq != nil {
			chama.MeetingSchedule = &models.MeetingSchedule{
				Frequency:  *meetingFreq,
				DayOfWeek:  meetingDayOfWeek,
				DayOfMonth: meetingDayOfMonth,
				Time:       utils.DerefString(meetingTime),
			}
		}

		chamas = append(chamas, chama)
	}

	return chamas, nil
}

// JoinChama adds a user to a chama
func (s *ChamaService) JoinChama(chamaID, userID string) error {
	// Get chama
	chama, err := s.GetChamaByID(chamaID)
	if err != nil {
		return err
	}

	// Check if user can join
	if !chama.CanJoin() {
		return fmt.Errorf("cannot join this chama")
	}

	// Check if user is already a member
	exists, err := s.IsUserMember(chamaID, userID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("user is already a member")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Add member
	member := &models.ChamaMember{
		ID:                 uuid.New().String(),
		ChamaID:            chamaID,
		UserID:             userID,
		Role:               models.ChamaRoleMember,
		JoinedAt:           time.Now(),
		IsActive:           true,
		TotalContributions: 0,
		Rating:             0,
		TotalRatings:       0,
	}

	memberQuery := `
		INSERT INTO chama_members (
			id, chama_id, user_id, role, joined_at, is_active,
			total_contributions, rating, total_ratings
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(memberQuery,
		member.ID, member.ChamaID, member.UserID, member.Role, member.JoinedAt,
		member.IsActive, member.TotalContributions, member.Rating, member.TotalRatings,
	)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	// Update chama member count
	_, err = tx.Exec(
		"UPDATE chamas SET current_members = current_members + 1, updated_at = ? WHERE id = ?",
		time.Now(), chamaID,
	)
	if err != nil {
		return fmt.Errorf("failed to update member count: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LeaveChama removes a user from a chama
func (s *ChamaService) LeaveChama(chamaID, userID string) error {
	// Check if user is a member
	exists, err := s.IsUserMember(chamaID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("user is not a member")
	}

	// Get member details
	member, err := s.GetChamaMember(chamaID, userID)
	if err != nil {
		return err
	}

	// Check if user is chairperson (chairperson cannot leave unless transferring role)
	if member.Role == models.ChamaRoleChairperson {
		return fmt.Errorf("chairperson cannot leave without transferring role")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Deactivate member
	_, err = tx.Exec(
		"UPDATE chama_members SET is_active = false WHERE chama_id = ? AND user_id = ?",
		chamaID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to deactivate member: %w", err)
	}

	// Update chama member count
	_, err = tx.Exec(
		"UPDATE chamas SET current_members = current_members - 1, updated_at = ? WHERE id = ?",
		time.Now(), chamaID,
	)
	if err != nil {
		return fmt.Errorf("failed to update member count: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetChamaMembers retrieves members of a chama
func (s *ChamaService) GetChamaMembers(chamaID string) ([]*models.ChamaMember, error) {
	query := `
		SELECT cm.id, cm.chama_id, cm.user_id, cm.role, cm.joined_at, cm.is_active,
			   cm.total_contributions, cm.last_contribution, cm.rating, cm.total_ratings,
			   u.first_name, u.last_name, u.email, u.phone, u.avatar
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		WHERE cm.chama_id = ? AND cm.is_active = true
		ORDER BY cm.joined_at ASC
	`

	rows, err := s.db.Query(query, chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama members: %w", err)
	}
	defer rows.Close()

	var members []*models.ChamaMember
	for rows.Next() {
		member := &models.ChamaMember{}
		user := &models.User{}

		err := rows.Scan(
			&member.ID, &member.ChamaID, &member.UserID, &member.Role, &member.JoinedAt,
			&member.IsActive, &member.TotalContributions, &member.LastContribution,
			&member.Rating, &member.TotalRatings, &user.FirstName, &user.LastName,
			&user.Email, &user.Phone, &user.Avatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		user.ID = member.UserID
		member.User = user
		members = append(members, member)
	}

	return members, nil
}

// GetChamaMember retrieves a specific chama member
func (s *ChamaService) GetChamaMember(chamaID, userID string) (*models.ChamaMember, error) {
	query := `
		SELECT id, chama_id, user_id, role, joined_at, is_active,
			   total_contributions, last_contribution, rating, total_ratings
		FROM chama_members
		WHERE chama_id = ? AND user_id = ?
	`

	member := &models.ChamaMember{}
	err := s.db.QueryRow(query, chamaID, userID).Scan(
		&member.ID, &member.ChamaID, &member.UserID, &member.Role, &member.JoinedAt,
		&member.IsActive, &member.TotalContributions, &member.LastContribution,
		&member.Rating, &member.TotalRatings,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("member not found")
		}
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	return member, nil
}

// IsUserMember checks if a user is a member of a chama
func (s *ChamaService) IsUserMember(chamaID, userID string) (bool, error) {
	query := "SELECT COUNT(*) FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = true"
	var count int
	err := s.db.QueryRow(query, chamaID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}
	return count > 0, nil
}

// UpdateMemberRole updates a member's role in a chama (with permission check)
func (s *ChamaService) UpdateMemberRole(chamaID, userID string, newRole models.ChamaRole, updatedBy string) error {
	// Check if updater has permission (only chairperson can change roles)
	updaterMember, err := s.GetChamaMember(chamaID, updatedBy)
	if err != nil {
		return err
	}
	if updaterMember.Role != models.ChamaRoleChairperson {
		return fmt.Errorf("only chairperson can update member roles")
	}

	// Update role
	query := "UPDATE chama_members SET role = ? WHERE chama_id = ? AND user_id = ?"
	_, err = s.db.Exec(query, newRole, chamaID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

// UpdateMemberRoleSimple updates a member's role in a chama (for API compatibility)
func (s *ChamaService) UpdateMemberRoleSimple(chamaID, userID, newRole string) error {
	// Update role directly with string
	query := "UPDATE chama_members SET role = ?, updated_at = ? WHERE chama_id = ? AND user_id = ?"
	_, err := s.db.Exec(query, newRole, time.Now(), chamaID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	return nil
}

// UpdateMemberStatus updates a member's status in a chama
func (s *ChamaService) UpdateMemberStatus(chamaID, userID, status string) error {
	// Validate status
	validStatuses := map[string]bool{
		"active":    true,
		"inactive":  true,
		"suspended": true,
		"pending":   true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	// Update status by setting is_active based on status
	isActive := status == "active"
	query := "UPDATE chama_members SET is_active = ?, updated_at = ? WHERE chama_id = ? AND user_id = ?"
	_, err := s.db.Exec(query, isActive, time.Now(), chamaID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member status: %w", err)
	}

	return nil
}

// GetChamaStatistics returns comprehensive statistics for a chama with user-specific data
func (s *ChamaService) GetChamaStatistics(chamaID, userID string) (map[string]interface{}, error) {
	// fmt.Printf("🔍 Getting chama statistics for chamaID: '%s', userID: '%s'\n", chamaID, userID)
	stats := make(map[string]interface{})

	// Get basic chama info
	chama, err := s.GetChamaByID(chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama: %w", err)
	}

	// Get member statistics
	memberStats, err := s.getMemberStatistics(chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member statistics: %w", err)
	}

	// Get financial statistics
	financialStats, err := s.getFinancialStatistics(chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial statistics: %w", err)
	}

	// Get activity statistics
	fmt.Printf("🔍 Getting activity statistics for chama: %s\n", chamaID)
	activityStats, err := s.getActivityStatistics(chamaID)
	if err != nil {
		fmt.Printf("❌ Error getting activity statistics: %v\n", err)
		return nil, fmt.Errorf("failed to get activity statistics: %w", err)
	}
	fmt.Printf("🔍 Activity statistics result: %+v\n", activityStats)

	// Get chama wallet balance
	walletBalance, err := s.getChamaWalletBalance(chamaID)
	if err != nil {
		fmt.Printf("Warning: Failed to get chama wallet balance: %v\n", err)
		walletBalance = 0
	}

	// Get user-specific statistics
	userStats, err := s.getUserChamaStatistics(chamaID, userID)
	if err != nil {
		fmt.Printf("Warning: Failed to get user statistics: %v\n", err)
		userStats = make(map[string]interface{})
	}

	stats["chama_info"] = map[string]interface{}{
		"id":                     chama.ID,
		"name":                   chama.Name,
		"type":                   chama.Type,
		"status":                 chama.Status,
		"created_at":             chama.CreatedAt,
		"contribution_amount":    chama.ContributionAmount,
		"contribution_frequency": chama.ContributionFrequency,
		"max_members":            chama.MaxMembers,
		"current_members":        chama.CurrentMembers,
		"total_funds":            walletBalance, // Use actual wallet balance
		"wallet_balance":         walletBalance,
	}
	stats["user_stats"] = userStats

	stats["member_stats"] = memberStats
	stats["financial_stats"] = financialStats
	stats["activity_stats"] = activityStats

	fmt.Printf("🔍 Final statistics response structure:\n")
	fmt.Printf("  - member_stats: %+v\n", memberStats)
	fmt.Printf("  - financial_stats: %+v\n", financialStats)
	fmt.Printf("  - activity_stats: %+v\n", activityStats)

	return stats, nil
}

// getMemberStatistics gets member-related statistics
func (s *ChamaService) getMemberStatistics(chamaID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count members by role (include all members, not just active ones)
	roleQuery := `
		SELECT role, COUNT(*) as count
		FROM chama_members
		WHERE chama_id = ?
		GROUP BY role
	`
	rows, err := s.db.Query(roleQuery, chamaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roleStats := make(map[string]int)
	totalMembers := 0
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			continue
		}
		roleStats[role] = count
		totalMembers += count
	}

	// Get active members count separately
	activeMembersQuery := `
		SELECT COUNT(*) as active_count
		FROM chama_members
		WHERE chama_id = ? AND (is_active = true OR is_active IS NULL)
	`
	var activeMembers int
	err = s.db.QueryRow(activeMembersQuery, chamaID).Scan(&activeMembers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_members"] = totalMembers
	stats["active_members"] = activeMembers
	stats["members_by_role"] = roleStats

	// Get member join trend (last 6 months)
	joinTrendQuery := `
		SELECT DATE(joined_at) as join_date, COUNT(*) as count
		FROM chama_members
		WHERE chama_id = ? AND joined_at >= DATE('now', '-6 months')
		GROUP BY DATE(joined_at)
		ORDER BY join_date
	`
	rows, err = s.db.Query(joinTrendQuery, chamaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	joinTrend := make([]map[string]interface{}, 0)
	for rows.Next() {
		var joinDate string
		var count int
		if err := rows.Scan(&joinDate, &count); err != nil {
			continue
		}
		joinTrend = append(joinTrend, map[string]interface{}{
			"date":  joinDate,
			"count": count,
		})
	}

	stats["join_trend"] = joinTrend

	return stats, nil
}

// getFinancialStatistics gets financial-related statistics
func (s *ChamaService) getFinancialStatistics(chamaID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get total contributions (chama transactions use recipient_id for chama_id)
	contributionQuery := `
		SELECT
			COALESCE(SUM(amount), 0) as total_contributions,
			COUNT(*) as total_transactions,
			COALESCE(AVG(amount), 0) as average_contribution
		FROM transactions
		WHERE recipient_id = ? AND type = 'contribution' AND status = 'completed'
	`
	var totalContributions, averageContribution float64
	var totalTransactions int
	err := s.db.QueryRow(contributionQuery, chamaID).Scan(&totalContributions, &totalTransactions, &averageContribution)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_contributions"] = totalContributions
	stats["total_transactions"] = totalTransactions
	stats["average_contribution"] = averageContribution

	// Get monthly contribution trend
	monthlyQuery := `
		SELECT
			strftime('%Y-%m', created_at) as month,
			COALESCE(SUM(amount), 0) as total,
			COUNT(*) as count
		FROM transactions
		WHERE recipient_id = ? AND type = 'contribution' AND status = 'completed'
		AND created_at >= DATE('now', '-12 months')
		GROUP BY strftime('%Y-%m', created_at)
		ORDER BY month
	`
	rows, err := s.db.Query(monthlyQuery, chamaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monthlyTrend := make([]map[string]interface{}, 0)
	for rows.Next() {
		var month string
		var total float64
		var count int
		if err := rows.Scan(&month, &total, &count); err != nil {
			continue
		}
		monthlyTrend = append(monthlyTrend, map[string]interface{}{
			"month": month,
			"total": total,
			"count": count,
		})
	}

	stats["monthly_trend"] = monthlyTrend

	return stats, nil
}

// getActivityStatistics gets activity-related statistics
func (s *ChamaService) getActivityStatistics(chamaID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get meeting statistics (only upcoming and ongoing meetings)
	fmt.Printf("🔍 Getting meeting statistics for chama: %s\n", chamaID)

	// First, let's check if there are any meetings at all for this chama
	var debugCount int
	debugQuery := "SELECT COUNT(*) FROM meetings WHERE chama_id = ?"
	debugErr := s.db.QueryRow(debugQuery, chamaID).Scan(&debugCount)
	if debugErr != nil {
		fmt.Printf("❌ Error in debug query: %v\n", debugErr)
	} else {
		fmt.Printf("🔍 Debug: Found %d meetings for chama %s\n", debugCount, chamaID)
	}

	// Let's see what the actual meeting data looks like
	if debugCount > 0 {
		detailQuery := `SELECT id, title, status, scheduled_at,
			datetime('now') as current_utc,
			datetime('now', '+3 hours') as current_eat,
			CASE WHEN scheduled_at > datetime('now', '+3 hours') THEN 'UPCOMING' ELSE 'PAST' END as time_status
			FROM meetings WHERE chama_id = ? LIMIT 3`
		rows, detailErr := s.db.Query(detailQuery, chamaID)
		if detailErr == nil {
			defer rows.Close()
			fmt.Printf("🔍 Meeting details for chama %s:\n", chamaID)
			for rows.Next() {
				var id, title, status, scheduledAt, currentUTC, currentEAT, timeStatus string
				if scanErr := rows.Scan(&id, &title, &status, &scheduledAt, &currentUTC, &currentEAT, &timeStatus); scanErr == nil {
					fmt.Printf("  - ID: %s, Title: %s, Status: %s\n", id, title, status)
					fmt.Printf("    Scheduled: %s, UTC: %s, EAT: %s, TimeStatus: %s\n",
						scheduledAt, currentUTC, currentEAT, timeStatus)
				}
			}
		}
	}

	// Fixed query - use EAT timezone and correct status values
	meetingQuery := `
		SELECT
			COUNT(*) as total_meetings,
			COUNT(CASE WHEN status IN ('completed', 'ended') THEN 1 END) as completed_meetings,
			COUNT(CASE WHEN status IN ('scheduled', 'pending', 'ready') AND scheduled_at > datetime('now', '+3 hours') THEN 1 END) as upcoming_meetings,
			COUNT(CASE WHEN status IN ('ongoing', 'active', 'started', 'live') THEN 1 END) as ongoing_meetings,
			COUNT(CASE WHEN status NOT IN ('completed', 'cancelled', 'ended') THEN 1 END) as active_meetings_alt
		FROM meetings
		WHERE chama_id = ?
	`
	var totalMeetings, completedMeetings, upcomingMeetings, ongoingMeetings, activeMeetingsAlt int
	err := s.db.QueryRow(meetingQuery, chamaID).Scan(&totalMeetings, &completedMeetings, &upcomingMeetings, &ongoingMeetings, &activeMeetingsAlt)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("❌ Error querying meetings: %v\n", err)
		return nil, err
	}

	fmt.Printf("📊 Meeting statistics: total=%d, completed=%d, upcoming=%d, ongoing=%d, active_alt=%d\n",
		totalMeetings, completedMeetings, upcomingMeetings, ongoingMeetings, activeMeetingsAlt)

	// Use total meetings for dashboard (as requested)
	activeMeetings := upcomingMeetings + ongoingMeetings

	stats["total_meetings"] = totalMeetings
	stats["completed_meetings"] = completedMeetings
	stats["upcoming_meetings"] = upcomingMeetings
	stats["ongoing_meetings"] = ongoingMeetings
	stats["active_meetings"] = activeMeetings

	// Get loan statistics
	loanQuery := `
		SELECT
			COUNT(*) as total_loans,
			COUNT(CASE WHEN status = 'approved' THEN 1 END) as approved_loans,
			COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending_loans,
			COALESCE(SUM(CASE WHEN status = 'approved' THEN amount ELSE 0 END), 0) as total_loan_amount
		FROM loans
		WHERE chama_id = ?
	`
	var totalLoans, approvedLoans, pendingLoans int
	var totalLoanAmount float64
	err = s.db.QueryRow(loanQuery, chamaID).Scan(&totalLoans, &approvedLoans, &pendingLoans, &totalLoanAmount)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	stats["total_loans"] = totalLoans
	stats["approved_loans"] = approvedLoans
	stats["pending_loans"] = pendingLoans
	stats["total_loan_amount"] = totalLoanAmount

	fmt.Printf("🔍 getActivityStatistics returning: %+v\n", stats)
	return stats, nil
}

// getChamaWalletBalance gets the actual wallet balance for a chama
func (s *ChamaService) getChamaWalletBalance(chamaID string) (float64, error) {
	// First, ensure chama wallet exists
	err := s.ensureChamaWallet(chamaID)
	if err != nil {
		return 0, fmt.Errorf("failed to ensure chama wallet: %w", err)
	}

	query := `
		SELECT COALESCE(balance, 0) as balance
		FROM wallets
		WHERE owner_id = ? AND type = 'chama'
	`
	var balance float64
	err = s.db.QueryRow(query, chamaID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("failed to get chama wallet balance: %w", err)
	}
	return balance, nil
}

// ensureChamaWallet ensures that a chama has a wallet
func (s *ChamaService) ensureChamaWallet(chamaID string) error {
	// Check if wallet exists
	var exists bool
	checkQuery := `
		SELECT EXISTS(SELECT 1 FROM wallets WHERE owner_id = ? AND type = 'chama')
	`
	err := s.db.QueryRow(checkQuery, chamaID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check wallet existence: %w", err)
	}

	if !exists {
		// Create chama wallet
		walletID := fmt.Sprintf("wallet-%s", chamaID)
		_, err = s.db.Exec(`
			INSERT INTO wallets (id, owner_id, type, balance, created_at, updated_at)
			VALUES (?, ?, 'chama', 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, walletID, chamaID)
		if err != nil {
			return fmt.Errorf("failed to create chama wallet: %w", err)
		}
		fmt.Printf("✅ Created chama wallet for chama ID: %s\n", chamaID)
	}

	return nil
}

// getUserChamaStatistics gets user-specific statistics for a chama
func (s *ChamaService) getUserChamaStatistics(chamaID, userID string) (map[string]interface{}, error) {
	userStats := make(map[string]interface{})

	// Get user's role in the chama
	var role string
	roleQuery := `
		SELECT role
		FROM chama_members
		WHERE chama_id = ? AND user_id = ?
	`
	err := s.db.QueryRow(roleQuery, chamaID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			role = "not_member"
		} else {
			return nil, fmt.Errorf("failed to get user role: %w", err)
		}
	}

	// Get user's transaction summary in this chama
	transactionQuery := `
		SELECT
			COUNT(*) as total_transactions,
			COALESCE(SUM(CASE WHEN type = 'contribution' THEN amount ELSE 0 END), 0) as total_contributions,
			COALESCE(SUM(CASE WHEN type = 'loan' THEN amount ELSE 0 END), 0) as total_loans,
			COALESCE(SUM(CASE WHEN type = 'withdrawal' THEN amount ELSE 0 END), 0) as total_withdrawals
		FROM transactions
		WHERE initiated_by = ? AND recipient_id = ?
	`
	var totalTransactions int
	var totalContributions, totalLoans, totalWithdrawals float64
	err = s.db.QueryRow(transactionQuery, userID, chamaID).Scan(
		&totalTransactions, &totalContributions, &totalLoans, &totalWithdrawals)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get user transaction summary: %w", err)
	}

	// Get user's contribution count (number of contribution records)
	contributionCountQuery := `
		SELECT COUNT(*)
		FROM transactions
		WHERE initiated_by = ? AND recipient_id = ? AND type = 'contribution' AND status = 'completed'
	`
	var contributionCount int
	err = s.db.QueryRow(contributionCountQuery, userID, chamaID).Scan(&contributionCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get user contribution count: %w", err)
	}

	userStats["role"] = role
	userStats["total_transactions"] = totalTransactions
	userStats["total_contributions_amount"] = totalContributions
	userStats["total_loans_amount"] = totalLoans
	userStats["total_withdrawals_amount"] = totalWithdrawals
	userStats["contribution_count"] = contributionCount

	return userStats, nil
}

// GetChamaTransactions retrieves all transactions for a chama
func (s *ChamaService) GetChamaTransactions(chamaID string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT
			t.id, t.from_wallet_id, t.to_wallet_id, t.type, t.status, t.amount, t.currency,
			t.description, t.reference, t.payment_method, t.metadata, t.fees,
			t.initiated_by, t.approved_by, t.requires_approval, t.approval_deadline,
			t.created_at, t.updated_at, t.recipient_id,
			u.first_name, u.last_name, u.email, u.phone
		FROM transactions t
		LEFT JOIN users u ON t.initiated_by = u.id
		WHERE t.recipient_id = ?
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		transaction := &models.Transaction{}
		var userFirstName, userLastName, userEmail, userPhone sql.NullString
		var metadataJSON sql.NullString

		var recipientID sql.NullString
		err := rows.Scan(
			&transaction.ID,
			&transaction.FromWalletID,
			&transaction.ToWalletID,
			&transaction.Type,
			&transaction.Status,
			&transaction.Amount,
			&transaction.Currency,
			&transaction.Description,
			&transaction.Reference,
			&transaction.PaymentMethod,
			&metadataJSON,
			&transaction.Fees,
			&transaction.InitiatedBy,
			&transaction.ApprovedBy,
			&transaction.RequiresApproval,
			&transaction.ApprovalDeadline,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
			&recipientID,
			&userFirstName,
			&userLastName,
			&userEmail,
			&userPhone,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Set recipient ID
		if recipientID.Valid {
			transaction.RecipientID = &recipientID.String
		}

		// Set metadata from JSON
		if metadataJSON.Valid && metadataJSON.String != "" {
			err = transaction.SetMetadataFromJSON(metadataJSON.String)
			if err != nil {
				fmt.Printf("Warning: Failed to parse metadata for transaction %s: %v\n", transaction.ID, err)
			}
		}

		// Add user information if available
		if userFirstName.Valid || userLastName.Valid {
			transaction.User = &models.User{
				ID:        transaction.InitiatedBy,
				FirstName: userFirstName.String,
				LastName:  userLastName.String,
				Email:     userEmail.String,
				Phone:     userPhone.String,
			}
		}

		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// AddMemberToChama adds a member to a chama with specified role
func (s *ChamaService) AddMemberToChama(chamaID, userID, role string) error {
	// Validate role
	validRoles := map[string]bool{
		"chairperson": true,
		"treasurer":   true,
		"secretary":   true,
		"member":      true,
		"assistant":   true,
	}

	if !validRoles[role] {
		role = "member" // Default to member if invalid role
	}

	// Check if user is already a member
	checkQuery := `SELECT COUNT(*) FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = true`
	var count int
	err := s.db.QueryRow(checkQuery, chamaID, userID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing membership: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("user is already a member of this chama")
	}

	// Add member
	member := &models.ChamaMember{
		ID:                 uuid.New().String(),
		ChamaID:            chamaID,
		UserID:             userID,
		Role:               models.ChamaRole(role),
		JoinedAt:           time.Now(),
		IsActive:           true,
		TotalContributions: 0,
		Rating:             0,
		TotalRatings:       0,
	}

	memberQuery := `
		INSERT INTO chama_members (
			id, chama_id, user_id, role, joined_at, is_active,
			total_contributions, rating, total_ratings
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(memberQuery,
		member.ID, member.ChamaID, member.UserID, member.Role, member.JoinedAt,
		member.IsActive, member.TotalContributions, member.Rating, member.TotalRatings,
	)
	if err != nil {
		return fmt.Errorf("failed to add member to chama: %w", err)
	}

	return nil
}

// UpdateChamaMemberCount updates the current member count for a chama
func (s *ChamaService) UpdateChamaMemberCount(chamaID string, memberCount int) error {
	query := `UPDATE chamas SET current_members = ?, updated_at = ? WHERE id = ?`

	_, err := s.db.Exec(query, memberCount, time.Now(), chamaID)
	if err != nil {
		return fmt.Errorf("failed to update chama member count: %w", err)
	}

	return nil
}

// GetUserRoleInChama gets the user's role in a specific chama
func (s *ChamaService) GetUserRoleInChama(chamaID, userID string) (string, error) {
	query := `
		SELECT role
		FROM chama_members
		WHERE chama_id = ? AND user_id = ? AND is_active = true
	`

	var role string
	err := s.db.QueryRow(query, chamaID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user is not a member of this chama")
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// UpdateChamaSettings updates chama settings (only for chairperson)
func (s *ChamaService) UpdateChamaSettings(chamaID string, updates interface{}) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Build dynamic update query based on provided fields
	setParts := []string{}
	args := []interface{}{}

	// Use type assertion to handle the updates struct
	if updateMap, ok := updates.(*struct {
		Name                  *string                 `json:"name,omitempty"`
		Description           *string                 `json:"description,omitempty"`
		IsPublic              *bool                   `json:"is_public,omitempty"`
		RequiresApproval      *bool                   `json:"requires_approval,omitempty"`
		MaxMembers            *int                    `json:"max_members,omitempty"`
		ContributionAmount    *float64                `json:"contribution_amount,omitempty"`
		ContributionFrequency *string                 `json:"contribution_frequency,omitempty"`
		Rules                 *[]string               `json:"rules,omitempty"`
		MeetingSchedule       *map[string]interface{} `json:"meeting_schedule,omitempty"`
		Permissions           *map[string]bool        `json:"permissions,omitempty"`
		Notifications         *map[string]bool        `json:"notifications,omitempty"`
	}); ok {

		if updateMap.Name != nil {
			setParts = append(setParts, "name = ?")
			args = append(args, *updateMap.Name)
		}

		if updateMap.Description != nil {
			setParts = append(setParts, "description = ?")
			args = append(args, *updateMap.Description)
		}

		if updateMap.IsPublic != nil {
			setParts = append(setParts, "is_public = ?")
			args = append(args, *updateMap.IsPublic)
		}

		if updateMap.RequiresApproval != nil {
			setParts = append(setParts, "requires_approval = ?")
			args = append(args, *updateMap.RequiresApproval)
		}

		if updateMap.MaxMembers != nil {
			setParts = append(setParts, "max_members = ?")
			args = append(args, *updateMap.MaxMembers)
		}

		if updateMap.ContributionAmount != nil {
			setParts = append(setParts, "contribution_amount = ?")
			args = append(args, *updateMap.ContributionAmount)
		}

		if updateMap.ContributionFrequency != nil {
			setParts = append(setParts, "contribution_frequency = ?")
			args = append(args, *updateMap.ContributionFrequency)
		}

		if updateMap.Rules != nil {
			rulesJSON, err := json.Marshal(*updateMap.Rules)
			if err != nil {
				return fmt.Errorf("failed to marshal rules: %w", err)
			}
			setParts = append(setParts, "rules = ?")
			args = append(args, string(rulesJSON))
		}

		if updateMap.Permissions != nil {
			permissionsJSON, err := json.Marshal(*updateMap.Permissions)
			if err != nil {
				return fmt.Errorf("failed to marshal permissions: %w", err)
			}
			setParts = append(setParts, "permissions = ?")
			args = append(args, string(permissionsJSON))
		}
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add updated_at timestamp
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())

	// Add chama ID for WHERE clause
	args = append(args, chamaID)

	// Build and execute update query
	query := fmt.Sprintf("UPDATE chamas SET %s WHERE id = ?", strings.Join(setParts, ", "))

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update chama: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RemoveUserFromChama removes a user from a chama (leave chama functionality)
func (s *ChamaService) RemoveUserFromChama(chamaID, userID string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if user is a member
	var memberExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = true)", chamaID, userID).Scan(&memberExists)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}

	if !memberExists {
		return fmt.Errorf("user is not a member of this chama")
	}

	// Mark member as inactive instead of deleting (for audit trail)
	_, err = tx.Exec("UPDATE chama_members SET is_active = false, updated_at = ? WHERE chama_id = ? AND user_id = ?", time.Now(), chamaID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from chama: %w", err)
	}

	// Update chama member count
	_, err = tx.Exec("UPDATE chamas SET current_members = current_members - 1, updated_at = ? WHERE id = ?", time.Now(), chamaID)
	if err != nil {
		return fmt.Errorf("failed to update member count: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteChama deletes a chama and all related data (only for chairperson)
func (s *ChamaService) DeleteChama(chamaID string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if chama exists
	var chamaExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chamas WHERE id = ?)", chamaID).Scan(&chamaExists)
	if err != nil {
		return fmt.Errorf("failed to check chama existence: %w", err)
	}

	if !chamaExists {
		return fmt.Errorf("chama not found")
	}

	// Delete related data in correct order (respecting foreign key constraints)

	// Delete meeting attendance
	_, err = tx.Exec("DELETE FROM meeting_attendance WHERE meeting_id IN (SELECT id FROM meetings WHERE chama_id = ?)", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete meeting attendance: %w", err)
	}

	// Delete meeting documents
	_, err = tx.Exec("DELETE FROM meeting_documents WHERE meeting_id IN (SELECT id FROM meetings WHERE chama_id = ?)", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete meeting documents: %w", err)
	}

	// Delete meetings
	_, err = tx.Exec("DELETE FROM meetings WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete meetings: %w", err)
	}

	// Delete welfare contributions
	_, err = tx.Exec("DELETE FROM welfare_contributions WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete welfare contributions: %w", err)
	}

	// Delete welfare requests
	_, err = tx.Exec("DELETE FROM welfare_requests WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete welfare requests: %w", err)
	}

	// Delete loan guarantors
	_, err = tx.Exec("DELETE FROM loan_guarantors WHERE loan_id IN (SELECT id FROM loans WHERE chama_id = ?)", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete loan guarantors: %w", err)
	}

	// Delete loans
	_, err = tx.Exec("DELETE FROM loans WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete loans: %w", err)
	}

	// Delete contributions
	_, err = tx.Exec("DELETE FROM contributions WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete contributions: %w", err)
	}

	// Delete chama members
	_, err = tx.Exec("DELETE FROM chama_members WHERE chama_id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete chama members: %w", err)
	}

	// Finally, delete the chama itself
	_, err = tx.Exec("DELETE FROM chamas WHERE id = ?", chamaID)
	if err != nil {
		return fmt.Errorf("failed to delete chama: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SendInvitation sends an invitation to join a chama with role information
func (s *ChamaService) SendInvitation(chamaID, inviterID, email, phoneNumber, message, role, roleName, roleDescription string) (string, error) {
	fmt.Printf("🔍 SendInvitation called with:\n")
	fmt.Printf("  - chamaID: %s\n", chamaID)
	fmt.Printf("  - inviterID: %s\n", inviterID)
	fmt.Printf("  - email: %s\n", email)
	fmt.Printf("  - phoneNumber: %s\n", phoneNumber)

	// Start transaction
	fmt.Printf("🔍 Starting database transaction...\n")
	tx, err := s.db.Begin()
	if err != nil {
		fmt.Printf("❌ Failed to start transaction: %v\n", err)
		return "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if user with email already exists and is already a member
	var existingUserID string
	err = tx.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingUserID)
	if err == nil {
		// User exists, check if already a member
		var memberExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = true)", chamaID, existingUserID).Scan(&memberExists)
		if err != nil {
			return "", fmt.Errorf("failed to check existing membership: %w", err)
		}
		if memberExists {
			return "", fmt.Errorf("user is already a member of this chama")
		}
	}

	// Check if there's already a pending invitation for this email and chama
	var pendingInvitation bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chama_invitations WHERE chama_id = ? AND email = ? AND status = 'pending')", chamaID, email).Scan(&pendingInvitation)
	if err != nil {
		return "", fmt.Errorf("failed to check pending invitations: %w", err)
	}
	if pendingInvitation {
		return "", fmt.Errorf("invitation already sent to this email")
	}

	// Create invitation
	invitationID := uuid.New().String()
	invitationToken := uuid.New().String() // Token for accepting invitation

	_, err = tx.Exec(`
		INSERT INTO chama_invitations (
			id, chama_id, inviter_id, email, phone_number, message,
			role, role_name, role_description,
			invitation_token, status, created_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
	`, invitationID, chamaID, inviterID, email, phoneNumber, message,
		role, roleName, roleDescription,
		invitationToken, time.Now(), time.Now().Add(7*24*time.Hour)) // Expires in 7 days
	if err != nil {
		return "", fmt.Errorf("failed to create invitation: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Get chama and inviter details for the email
	var chamaName, inviterFirstName, inviterLastName string
	err = s.db.QueryRow(`
		SELECT c.name, u.first_name, u.last_name
		FROM chamas c, users u
		WHERE c.id = ? AND u.id = ?
	`, chamaID, inviterID).Scan(&chamaName, &inviterFirstName, &inviterLastName)
	if err != nil {
		fmt.Printf("Warning: Could not get chama/inviter details for email: %v\n", err)
		// Don't fail the invitation if email fails
		return invitationID, nil
	}

	// Send email invitation
	fmt.Printf("🔍 Checking email invitation conditions:\n")
	fmt.Printf("  - Email service initialized: %t\n", s.emailService != nil)
	fmt.Printf("  - Email provided: '%s' (empty: %t)\n", email, email == "")
	fmt.Printf("  - Chama name: '%s'\n", chamaName)
	fmt.Printf("  - Inviter: '%s %s'\n", inviterFirstName, inviterLastName)

	if s.emailService != nil && email != "" {
		fmt.Printf("📧 Attempting to send chama invitation email...\n")
		inviterFullName := fmt.Sprintf("%s %s", inviterFirstName, inviterLastName)
		err = s.emailService.SendChamaInvitationEmail(email, chamaName, inviterFullName, message, invitationToken)
		if err != nil {
			fmt.Printf("❌ Failed to send invitation email: %v\n", err)
			// Don't fail the invitation if email fails
		} else {
			fmt.Printf("✅ Invitation email sent successfully to: %s\n", email)
		}
	} else {
		if s.emailService == nil {
			fmt.Printf("❌ Email service not initialized\n")
		}
		if email == "" {
			fmt.Printf("❌ No email address provided\n")
		}
	}

	return invitationID, nil
}

// RespondToInvitation handles accepting or rejecting an invitation
func (s *ChamaService) RespondToInvitation(invitationID, userID, response string) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Get invitation details
	var invitation struct {
		ChamaID   string
		Email     string
		Status    string
		ExpiresAt time.Time
	}

	err = tx.QueryRow(`
		SELECT chama_id, email, status, expires_at
		FROM chama_invitations
		WHERE id = ?
	`, invitationID).Scan(&invitation.ChamaID, &invitation.Email, &invitation.Status, &invitation.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invitation not found")
		}
		return fmt.Errorf("failed to get invitation: %w", err)
	}

	// Check if invitation is still valid
	if invitation.Status != "pending" {
		return fmt.Errorf("invitation has already been %s", invitation.Status)
	}

	if time.Now().After(invitation.ExpiresAt) {
		return fmt.Errorf("invitation has expired")
	}

	// Verify that the user's email matches the invitation
	var userEmail string
	err = tx.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	if userEmail != invitation.Email {
		return fmt.Errorf("invitation email does not match user email")
	}

	// Update invitation status
	_, err = tx.Exec(`
		UPDATE chama_invitations
		SET status = ?, responded_at = ?, responded_by = ?
		WHERE id = ?
	`, response+"ed", time.Now(), userID, invitationID) // "accepted" or "rejected"
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	// If accepted, add user to chama
	if response == "accept" {
		// Check if user is already a member (double-check)
		var memberExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = true)", invitation.ChamaID, userID).Scan(&memberExists)
		if err != nil {
			return fmt.Errorf("failed to check membership: %w", err)
		}

		if !memberExists {
			// Add user as member
			memberID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO chama_members (
					id, chama_id, user_id, role, joined_at, is_active,
					total_contributions, rating, total_ratings
				) VALUES (?, ?, ?, 'member', ?, true, 0, 0, 0)
			`, memberID, invitation.ChamaID, userID, time.Now())
			if err != nil {
				return fmt.Errorf("failed to add user to chama: %w", err)
			}

			// Update chama member count
			_, err = tx.Exec("UPDATE chamas SET current_members = current_members + 1, updated_at = ? WHERE id = ?", time.Now(), invitation.ChamaID)
			if err != nil {
				return fmt.Errorf("failed to update member count: %w", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserInvitations gets pending invitations for a user
func (s *ChamaService) GetUserInvitations(userID string) ([]map[string]interface{}, error) {
	// Get user's email
	var userEmail string
	err := s.db.QueryRow("SELECT email FROM users WHERE id = ?", userID).Scan(&userEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user email: %w", err)
	}

	// Get pending invitations for this email
	query := `
		SELECT
			ci.id, ci.chama_id, ci.inviter_id, ci.email, ci.phone_number,
			ci.message, ci.created_at, ci.expires_at,
			c.name as chama_name, c.description as chama_description,
			c.contribution_amount, c.contribution_frequency,
			u.first_name as inviter_first_name, u.last_name as inviter_last_name
		FROM chama_invitations ci
		INNER JOIN chamas c ON ci.chama_id = c.id
		INNER JOIN users u ON ci.inviter_id = u.id
		WHERE ci.email = ? AND ci.status = 'pending' AND ci.expires_at > ?
		ORDER BY ci.created_at DESC
	`

	rows, err := s.db.Query(query, userEmail, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}
	defer rows.Close()

	var invitations []map[string]interface{}
	for rows.Next() {
		var (
			id, chamaID, inviterID, email, chamaName, chamaDescription string
			contributionFrequency, inviterFirstName, inviterLastName   string
			createdAt, expiresAt                                       string
			phoneNumber, message                                       *string
			contributionAmount                                         float64
		)

		err := rows.Scan(
			&id, &chamaID, &inviterID, &email, &phoneNumber,
			&message, &createdAt, &expiresAt,
			&chamaName, &chamaDescription, &contributionAmount, &contributionFrequency,
			&inviterFirstName, &inviterLastName,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		invitation := map[string]interface{}{
			"id":         id,
			"chama_id":   chamaID,
			"inviter_id": inviterID,
			"email":      email,
			"phone":      phoneNumber,
			"message":    message,
			"created_at": createdAt,
			"expires_at": expiresAt,
			"chama": map[string]interface{}{
				"id":                     chamaID,
				"name":                   chamaName,
				"description":            chamaDescription,
				"contribution_amount":    contributionAmount,
				"contribution_frequency": contributionFrequency,
			},
			"inviter": map[string]interface{}{
				"id":         inviterID,
				"first_name": inviterFirstName,
				"last_name":  inviterLastName,
			},
		}

		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

// GetChamaSentInvitations gets all invitations sent for a specific chama
func (s *ChamaService) GetChamaSentInvitations(chamaID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			ci.id, ci.chama_id, ci.inviter_id, ci.email, ci.phone_number,
			ci.message, ci.role, ci.role_name, ci.role_description,
			ci.status, ci.created_at, ci.expires_at, ci.responded_at,
			u.first_name as inviter_first_name, u.last_name as inviter_last_name
		FROM chama_invitations ci
		INNER JOIN users u ON ci.inviter_id = u.id
		WHERE ci.chama_id = ?
		ORDER BY ci.created_at DESC
	`

	rows, err := s.db.Query(query, chamaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent invitations: %w", err)
	}
	defer rows.Close()

	var invitations []map[string]interface{}

	for rows.Next() {
		var (
			id, chamaID, inviterID, email                         string
			phoneNumber, message, role, roleName, roleDescription sql.NullString
			status                                                string
			createdAt, expiresAt                                  time.Time
			respondedAt                                           sql.NullTime
			inviterFirstName, inviterLastName                     string
		)

		err := rows.Scan(
			&id, &chamaID, &inviterID, &email, &phoneNumber,
			&message, &role, &roleName, &roleDescription,
			&status, &createdAt, &expiresAt, &respondedAt,
			&inviterFirstName, &inviterLastName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}

		invitation := map[string]interface{}{
			"id":               id,
			"chama_id":         chamaID,
			"inviter_id":       inviterID,
			"email":            email,
			"phone":            phoneNumber.String,
			"message":          message.String,
			"role":             role.String,
			"role_name":        roleName.String,
			"role_description": roleDescription.String,
			"status":           status,
			"created_at":       createdAt,
			"expires_at":       expiresAt,
			"inviter": map[string]interface{}{
				"id":         inviterID,
				"first_name": inviterFirstName,
				"last_name":  inviterLastName,
			},
		}

		if respondedAt.Valid {
			invitation["responded_at"] = respondedAt.Time
		}

		invitations = append(invitations, invitation)
	}

	return invitations, nil
}
