package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"vaultke-backend/internal/models"

	"github.com/google/uuid"
)

// DividendsService handles dividend-related business logic
type DividendsService struct {
	db *sql.DB
}

// NewDividendsService creates a new dividends service
func NewDividendsService(db *sql.DB) *DividendsService {
	return &DividendsService{db: db}
}

// DeclareDividend creates a new dividend declaration
func (s *DividendsService) DeclareDividend(chamaID, declaredBy string, req *models.CreateDividendDeclarationRequest) (*models.DividendDeclaration, error) {
	// Validate input
	if declaredBy == "" {
		return nil, fmt.Errorf("declaredBy cannot be empty")
	}

	// Ensure declaredBy is not empty
	if declaredBy == "" {
		declaredBy = "unknown"
	}

	// Validate that the user has permission to declare dividends (should be chama official)
	if !s.canDeclareDividends(declaredBy, chamaID) {
		return nil, fmt.Errorf("user does not have permission to declare dividends")
	}

	// Generate unique ID
	declarationID := uuid.New().String()
	now := time.Now()

	declaration := &models.DividendDeclaration{
		ID:                  declarationID,
		ChamaID:             chamaID,
		DeclarationDate:     now,
		DividendPerShare:    req.DividendPerShare,
		TotalDividendAmount: req.TotalDividendAmount,
		PaymentDate:         req.PaymentDate,
		Status:              models.DividendStatusDeclared,
		DeclaredBy:          declaredBy,
		Description:         req.Description,
		DividendType:        req.DividendType,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// Extract dividend_type from request if available (for backward compatibility)
	dividendType := "cash" // default
	if req.DividendType != "" {
		dividendType = req.DividendType
	}

	// Insert into database
	query := `
		INSERT INTO dividend_declarations (
			id, chama_id, declaration_date, dividend_per_share, total_amount,
			payment_date, status, declared_by, description, dividend_type, created_at, updated_at
		) VALUES (?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		declaration.ID,
		declaration.ChamaID,
		declaration.DividendPerShare,
		declaration.TotalDividendAmount,
		declaration.PaymentDate,
		declaration.Status,
		declaration.DeclaredBy,
		declaration.Description,
		dividendType,
		declaration.CreatedAt,
		declaration.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to declare dividend: %w", err)
	}

	// Create individual dividend payments for eligible members
	err = s.createDividendPayments(declarationID, chamaID)
	if err != nil {
		log.Printf("Warning: Failed to create dividend payments: %v", err)
	}

	log.Printf("Declared dividend %s for chama %s by user %s", declarationID, chamaID, declaredBy)
	return declaration, nil
}

// GetChamaDividendDeclarations retrieves dividend declarations for a chama
func (s *DividendsService) GetChamaDividendDeclarations(chamaID string, limit, offset int) ([]models.DividendDeclarationWithDetails, error) {
	query := `
		SELECT dd.id, dd.chama_id, dd.declaration_date, dd.dividend_per_share,
			   dd.total_amount, dd.payment_date, dd.status, dd.declared_by,
			   dd.approved_by, dd.description, dd.created_at, dd.updated_at,
			   u1.first_name, u1.last_name, u2.first_name, u2.last_name
		FROM dividend_declarations dd
		JOIN users u1 ON dd.declared_by = u1.id
		LEFT JOIN users u2 ON dd.approved_by = u2.id
		WHERE dd.chama_id = ?
		ORDER BY dd.declaration_date DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get dividend declarations: %w", err)
	}
	defer rows.Close()

	var declarations []models.DividendDeclarationWithDetails
	for rows.Next() {
		var declaration models.DividendDeclarationWithDetails
		var declaredByFirstName, declaredByLastName string
		var approvedByFirstName, approvedByLastName sql.NullString

		err := rows.Scan(
			&declaration.ID,
			&declaration.ChamaID,
			&declaration.DeclarationDate,
			&declaration.DividendPerShare,
			&declaration.TotalDividendAmount,
			&declaration.PaymentDate,
			&declaration.Status,
			&declaration.DeclaredBy,
			&declaration.ApprovedBy,
			&declaration.Description,
			&declaration.CreatedAt,
			&declaration.UpdatedAt,
			&declaredByFirstName,
			&declaredByLastName,
			&approvedByFirstName,
			&approvedByLastName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend declaration: %w", err)
		}

		declaration.DeclaredByName = declaredByFirstName + " " + declaredByLastName
		if approvedByFirstName.Valid && approvedByLastName.Valid {
			approvedByName := approvedByFirstName.String + " " + approvedByLastName.String
			declaration.ApprovedByName = &approvedByName
		}

		// Get payment statistics
		stats, err := s.getDividendPaymentStats(declaration.ID)
		if err != nil {
			log.Printf("Warning: Failed to get payment stats for dividend %s: %v", declaration.ID, err)
		} else {
			declaration.TotalEligibleShares = stats.TotalShares
			declaration.TotalRecipients = stats.TotalRecipients
			declaration.PaidAmount = stats.PaidAmount
			declaration.PendingAmount = stats.PendingAmount
		}

		declarations = append(declarations, declaration)
	}

	return declarations, nil
}

// ApproveDividend approves a dividend declaration
func (s *DividendsService) ApproveDividend(declarationID, approvedBy string) (*models.DividendDeclaration, error) {
	// Get existing declaration
	declaration, err := s.getDividendDeclarationByID(declarationID)
	if err != nil {
		return nil, err
	}

	// Check if can be approved
	if !declaration.CanApprove() {
		return nil, fmt.Errorf("dividend declaration cannot be approved in current status: %s", declaration.Status)
	}

	// Validate that the user has permission to approve dividends
	if !s.canApproveDividends(approvedBy, declaration.ChamaID) {
		return nil, fmt.Errorf("user does not have permission to approve dividends")
	}

	// Update status
	now := time.Now()
	query := `
		UPDATE dividend_declarations 
		SET status = ?, approved_by = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, models.DividendStatusApproved, approvedBy, now, declarationID)
	if err != nil {
		return nil, fmt.Errorf("failed to approve dividend: %w", err)
	}

	declaration.Status = models.DividendStatusApproved
	declaration.ApprovedBy = &approvedBy
	declaration.UpdatedAt = now

	log.Printf("Approved dividend %s by user %s", declarationID, approvedBy)
	return declaration, nil
}

// ProcessDividendPayments processes dividend payments for a declaration
func (s *DividendsService) ProcessDividendPayments(declarationID string, req *models.ProcessDividendPaymentsRequest) error {
	// Get declaration
	declaration, err := s.getDividendDeclarationByID(declarationID)
	if err != nil {
		return err
	}

	// Check if can be processed
	if !declaration.CanProcess() {
		return fmt.Errorf("dividend declaration cannot be processed in current status: %s", declaration.Status)
	}

	// Get pending payments
	payments, err := s.getPendingDividendPayments(declarationID)
	if err != nil {
		return fmt.Errorf("failed to get pending payments: %w", err)
	}

	if len(payments) == 0 {
		return fmt.Errorf("no pending payments found")
	}

	// Process each payment
	successCount := 0
	paymentDate := time.Now()
	if req.PaymentDate != nil {
		paymentDate = *req.PaymentDate
	}

	for _, payment := range payments {
		err := s.processSingleDividendPayment(&payment, req.PaymentMethod, paymentDate)
		if err != nil {
			log.Printf("Failed to process payment %s: %v", payment.ID, err)
		} else {
			successCount++
		}
	}

	// Update declaration status if all payments processed
	if successCount == len(payments) {
		err = s.updateDividendDeclarationStatus(declarationID, models.DividendStatusPaid)
		if err != nil {
			log.Printf("Warning: Failed to update declaration status: %v", err)
		}
	}

	log.Printf("Processed %d/%d dividend payments for declaration %s", successCount, len(payments), declarationID)
	return nil
}

// GetMemberDividendHistory retrieves dividend history for a member
func (s *DividendsService) GetMemberDividendHistory(chamaID, memberID string, limit, offset int) ([]models.DividendPaymentWithMemberInfo, error) {
	query := `
		SELECT dp.id, dp.dividend_declaration_id, dp.member_id, dp.shares_eligible,
			   dp.dividend_amount, dp.payment_status, dp.payment_date, dp.payment_method,
			   dp.transaction_reference, dp.created_at, dp.updated_at,
			   u.first_name, u.last_name, u.email, u.phone
		FROM dividend_payments dp
		JOIN dividend_declarations dd ON dp.dividend_declaration_id = dd.id
		JOIN users u ON dp.member_id = u.id
		WHERE dd.chama_id = ? AND dp.member_id = ?
		ORDER BY dp.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, memberID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get member dividend history: %w", err)
	}
	defer rows.Close()

	var payments []models.DividendPaymentWithMemberInfo
	for rows.Next() {
		var payment models.DividendPaymentWithMemberInfo
		var firstName, lastName string

		err := rows.Scan(
			&payment.ID,
			&payment.DividendDeclarationID,
			&payment.MemberID,
			&payment.SharesEligible,
			&payment.DividendAmount,
			&payment.PaymentStatus,
			&payment.PaymentDate,
			&payment.PaymentMethod,
			&payment.TransactionReference,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&firstName,
			&lastName,
			&payment.MemberEmail,
			&payment.MemberPhone,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dividend payment: %w", err)
		}

		payment.MemberName = firstName + " " + lastName
		payments = append(payments, payment)
	}

	return payments, nil
}

// Helper methods
func (s *DividendsService) getDividendDeclarationByID(declarationID string) (*models.DividendDeclaration, error) {
	query := `
		SELECT id, chama_id, declaration_date, dividend_per_share, total_amount,
			   payment_date, status, declared_by, approved_by, description, created_at, updated_at
		FROM dividend_declarations WHERE id = ?
	`

	var declaration models.DividendDeclaration
	err := s.db.QueryRow(query, declarationID).Scan(
		&declaration.ID,
		&declaration.ChamaID,
		&declaration.DeclarationDate,
		&declaration.DividendPerShare,
		&declaration.TotalDividendAmount,
		&declaration.PaymentDate,
		&declaration.Status,
		&declaration.DeclaredBy,
		&declaration.ApprovedBy,
		&declaration.Description,
		&declaration.CreatedAt,
		&declaration.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dividend declaration not found")
		}
		return nil, fmt.Errorf("failed to get dividend declaration: %w", err)
	}

	return &declaration, nil
}

func (s *DividendsService) createDividendPayments(declarationID, chamaID string) error {
	// Get all active shareholders in the chama
	query := `
		SELECT s.member_id, SUM(s.shares_owned) as total_shares
		FROM shares s
		WHERE s.chama_id = ? AND s.status = 'active'
		GROUP BY s.member_id
	`

	rows, err := s.db.Query(query, chamaID)
	if err != nil {
		return fmt.Errorf("failed to get shareholders: %w", err)
	}
	defer rows.Close()

	// Get dividend per share
	declaration, err := s.getDividendDeclarationByID(declarationID)
	if err != nil {
		return err
	}

	// Create payment records for each shareholder
	for rows.Next() {
		var memberID string
		var totalShares int

		err := rows.Scan(&memberID, &totalShares)
		if err != nil {
			log.Printf("Error scanning shareholder: %v", err)
			continue
		}

		dividendAmount := declaration.CalculateDividendAmount(totalShares)
		if dividendAmount <= 0 {
			continue
		}

		paymentID := uuid.New().String()
		now := time.Now()

		insertQuery := `
			INSERT INTO dividend_payments (
				id, dividend_declaration_id, member_id, shares_eligible,
				dividend_amount, payment_status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = s.db.Exec(
			insertQuery,
			paymentID,
			declarationID,
			memberID,
			totalShares,
			dividendAmount,
			models.DividendPaymentPending,
			now,
			now,
		)

		if err != nil {
			log.Printf("Failed to create dividend payment for member %s: %v", memberID, err)
		}
	}

	return nil
}

type dividendPaymentStats struct {
	TotalShares     int
	TotalRecipients int
	PaidAmount      float64
	PendingAmount   float64
}

func (s *DividendsService) getDividendPaymentStats(declarationID string) (*dividendPaymentStats, error) {
	query := `
		SELECT
			SUM(shares_eligible) as total_shares,
			COUNT(*) as total_recipients,
			SUM(CASE WHEN payment_status = 'paid' THEN dividend_amount ELSE 0 END) as paid_amount,
			SUM(CASE WHEN payment_status = 'pending' THEN dividend_amount ELSE 0 END) as pending_amount
		FROM dividend_payments
		WHERE dividend_declaration_id = ?
	`

	var stats dividendPaymentStats
	err := s.db.QueryRow(query, declarationID).Scan(
		&stats.TotalShares,
		&stats.TotalRecipients,
		&stats.PaidAmount,
		&stats.PendingAmount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}

	return &stats, nil
}

func (s *DividendsService) getPendingDividendPayments(declarationID string) ([]models.DividendPayment, error) {
	query := `
		SELECT id, dividend_declaration_id, member_id, shares_eligible, dividend_amount,
			   payment_status, payment_date, payment_method, transaction_reference,
			   created_at, updated_at
		FROM dividend_payments
		WHERE dividend_declaration_id = ? AND payment_status = 'pending'
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query, declarationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending payments: %w", err)
	}
	defer rows.Close()

	var payments []models.DividendPayment
	for rows.Next() {
		var payment models.DividendPayment
		err := rows.Scan(
			&payment.ID,
			&payment.DividendDeclarationID,
			&payment.MemberID,
			&payment.SharesEligible,
			&payment.DividendAmount,
			&payment.PaymentStatus,
			&payment.PaymentDate,
			&payment.PaymentMethod,
			&payment.TransactionReference,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment: %w", err)
		}
		payments = append(payments, payment)
	}

	return payments, nil
}

func (s *DividendsService) processSingleDividendPayment(payment *models.DividendPayment, paymentMethod string, paymentDate time.Time) error {
	// Generate transaction reference
	transactionRef := fmt.Sprintf("DIV-%s-%d", payment.ID[:8], time.Now().Unix())

	// Update payment record
	query := `
		UPDATE dividend_payments
		SET payment_status = ?, payment_date = ?, payment_method = ?,
			transaction_reference = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.Exec(
		query,
		models.DividendPaymentPaid,
		paymentDate,
		paymentMethod,
		transactionRef,
		time.Now(),
		payment.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Here you would integrate with actual payment processing
	// For now, we'll just mark as paid
	log.Printf("Processed dividend payment %s for member %s: %f", payment.ID, payment.MemberID, payment.DividendAmount)
	return nil
}

func (s *DividendsService) updateDividendDeclarationStatus(declarationID string, status models.DividendDeclarationStatus) error {
	query := `UPDATE dividend_declarations SET status = ?, updated_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, time.Now(), declarationID)
	return err
}

func (s *DividendsService) canDeclareDividends(userID, chamaID string) bool {
	// Check if user is a chama official (chairperson, secretary, or treasurer)
	query := `
		SELECT 1 FROM chama_members
		WHERE user_id = ? AND chama_id = ? AND is_active = TRUE
		AND role IN ('chairperson', 'secretary', 'treasurer')
	`
	var exists int
	err := s.db.QueryRow(query, userID, chamaID).Scan(&exists)
	return err == nil
}

func (s *DividendsService) canApproveDividends(userID, chamaID string) bool {
	// Check if user is a chama official (chairperson or treasurer)
	query := `
		SELECT 1 FROM chama_members
		WHERE user_id = ? AND chama_id = ? AND is_active = TRUE
		AND role IN ('chairperson', 'treasurer')
	`
	var exists int
	err := s.db.QueryRow(query, userID, chamaID).Scan(&exists)
	return err == nil
}
