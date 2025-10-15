package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// LoanService handles loan-related business logic
type LoanService struct {
	db *sql.DB
}

// NewLoanService creates a new loan service
func NewLoanService(db *sql.DB) *LoanService {
	return &LoanService{db: db}
}

// ApplyForLoan creates a new loan application
func (s *LoanService) ApplyForLoan(application *models.LoanApplication, borrowerID, chamaID string) (*models.Loan, error) {
	// Validate input
	if err := utils.ValidateStruct(application); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Check if user is a member of the chama
	chamaService := NewChamaService(s.db)
	isMember, err := chamaService.IsUserMember(chamaID, borrowerID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, fmt.Errorf("user is not a member of this chama")
	}

	// Check if user has any active loans
	hasActiveLoan, err := s.hasActiveLoan(borrowerID, chamaID)
	if err != nil {
		return nil, err
	}
	if hasActiveLoan {
		return nil, fmt.Errorf("user already has an active loan in this chama")
	}

	// Create loan
	loan := &models.Loan{
		ID:                 uuid.New().String(),
		BorrowerID:         borrowerID,
		ChamaID:            chamaID,
		Type:               application.Type,
		Amount:             application.Amount,
		InterestRate:       0, // Will be set during approval
		Duration:           application.Duration,
		Purpose:            application.Purpose,
		Status:             models.LoanStatusPending,
		TotalAmount:        0, // Will be calculated during approval
		PaidAmount:         0,
		RemainingAmount:    0,
		RequiredGuarantors: application.RequiredGuarantors,
		ApprovedGuarantors: 0,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert loan
	loanQuery := `
		INSERT INTO loans (
			id, borrower_id, chama_id, type, amount, interest_rate, duration,
			purpose, status, total_amount, paid_amount, remaining_amount,
			required_guarantors, approved_guarantors, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(loanQuery,
		loan.ID, loan.BorrowerID, loan.ChamaID, loan.Type, loan.Amount,
		loan.InterestRate, loan.Duration, loan.Purpose, loan.Status,
		loan.TotalAmount, loan.PaidAmount, loan.RemainingAmount,
		loan.RequiredGuarantors, loan.ApprovedGuarantors,
		loan.CreatedAt, loan.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create loan: %w", err)
	}

	// Add guarantors
	for _, guarantorUserID := range application.GuarantorUserIDs {
		// Check if guarantor is a chama member
		isGuarantorMember, err := chamaService.IsUserMember(chamaID, guarantorUserID)
		if err != nil {
			return nil, err
		}
		if !isGuarantorMember {
			return nil, fmt.Errorf("guarantor %s is not a member of this chama", guarantorUserID)
		}

		guarantor := &models.Guarantor{
			ID:        uuid.New().String(),
			LoanID:    loan.ID,
			UserID:    guarantorUserID,
			Amount:    application.Amount / float64(len(application.GuarantorUserIDs)),
			Status:    models.GuarantorStatusPending,
			CreatedAt: time.Now(),
		}

		guarantorQuery := `
			INSERT INTO guarantors (id, loan_id, user_id, amount, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`

		_, err = tx.Exec(guarantorQuery,
			guarantor.ID, guarantor.LoanID, guarantor.UserID,
			guarantor.Amount, guarantor.Status, guarantor.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to add guarantor: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send notifications to guarantors
	notificationService := NewNotificationService(s.db, nil)
	for _, guarantorUserID := range application.GuarantorUserIDs {
		go func(userID string) {
			title := "Loan Guarantee Request"
			message := fmt.Sprintf("You have been requested to guarantee a loan of KSh %.2f", application.Amount)
			data := map[string]interface{}{
				"type":   "loan_guarantee_request",
				"loanId": loan.ID,
				"amount": application.Amount,
			}
			notificationService.CreateNotification(userID, "loan", title, message, data, true, true, false)
		}(guarantorUserID)
	}

	return loan, nil
}

// RespondToGuaranteeRequest handles guarantor response
func (s *LoanService) RespondToGuaranteeRequest(loanID, guarantorUserID string, response *models.GuarantorResponse) error {
	// Get guarantor record
	guarantor, err := s.getGuarantor(loanID, guarantorUserID)
	if err != nil {
		return err
	}

	if guarantor.HasResponded() {
		return fmt.Errorf("guarantor has already responded")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Update guarantor status
	now := time.Now()
	status := models.GuarantorStatusRejected
	if response.Accept {
		status = models.GuarantorStatusAccepted
	}

	updateQuery := `
		UPDATE guarantors
		SET status = ?, message = ?, responded_at = ?
		WHERE id = ?
	`

	_, err = tx.Exec(updateQuery, status, response.Message, now, guarantor.ID)
	if err != nil {
		return fmt.Errorf("failed to update guarantor: %w", err)
	}

	// Update loan's approved guarantors count if accepted
	if response.Accept {
		_, err = tx.Exec(
			"UPDATE loans SET approved_guarantors = approved_guarantors + 1, updated_at = ? WHERE id = ?",
			now, loanID,
		)
		if err != nil {
			return fmt.Errorf("failed to update loan guarantors count: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Check if loan now has sufficient guarantors
	loan, err := s.GetLoanByID(loanID)
	if err != nil {
		return err
	}

	if loan.CanBeApproved() {
		// Notify chama leaders that loan is ready for approval
		s.notifyLoanReadyForApproval(loan)
	}

	return nil
}

// ApproveLoan approves or rejects a loan
func (s *LoanService) ApproveLoan(loanID, approverID string, approval *models.LoanApproval) error {
	// Get loan
	loan, err := s.GetLoanByID(loanID)
	if err != nil {
		return err
	}

	if !loan.IsPending() {
		return fmt.Errorf("loan is not pending approval")
	}

	// Check if approver has permission (chairperson or treasurer)
	chamaService := NewChamaService(s.db)
	member, err := chamaService.GetChamaMember(loan.ChamaID, approverID)
	if err != nil {
		return err
	}

	if !member.CanManageFinances() {
		return fmt.Errorf("user does not have permission to approve loans")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	var newStatus models.LoanStatus
	var totalAmount, remainingAmount float64

	if approval.Approved {
		newStatus = models.LoanStatusApproved
		loan.InterestRate = approval.InterestRate
		totalAmount = loan.CalculateTotalAmount()
		remainingAmount = totalAmount
	} else {
		newStatus = models.LoanStatusRejected
		totalAmount = 0
		remainingAmount = 0
	}

	// Update loan
	updateQuery := `
		UPDATE loans
		SET status = ?, interest_rate = ?, total_amount = ?, remaining_amount = ?,
			approved_by = ?, approved_at = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.Exec(updateQuery,
		newStatus, loan.InterestRate, totalAmount, remainingAmount,
		approverID, now, now, loanID,
	)
	if err != nil {
		return fmt.Errorf("failed to update loan: %w", err)
	}

	// If approved, disburse funds
	if approval.Approved {
		err = s.disburseLoan(tx, loan, totalAmount)
		if err != nil {
			return fmt.Errorf("failed to disburse loan: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send notification to borrower
	notificationService := NewNotificationService(s.db, nil)
	go notificationService.NotifyLoanApproval(loan.BorrowerID, loan.Amount, approval.Approved)

	return nil
}

// MakeLoanPayment processes a loan payment
func (s *LoanService) MakeLoanPayment(loanID, payerID string, amount float64, paymentMethod string) (*models.LoanPayment, error) {
	// Get loan
	loan, err := s.GetLoanByID(loanID)
	if err != nil {
		return nil, err
	}

	if !loan.IsActive() {
		return nil, fmt.Errorf("loan is not active")
	}

	if loan.BorrowerID != payerID {
		return nil, fmt.Errorf("only the borrower can make payments")
	}

	if amount <= 0 {
		return nil, fmt.Errorf("payment amount must be positive")
	}

	if amount > loan.RemainingAmount {
		return nil, fmt.Errorf("payment amount exceeds remaining balance")
	}

	// Calculate principal and interest portions
	interestPortion := (loan.TotalAmount - loan.Amount) * (amount / loan.TotalAmount)
	principalPortion := amount - interestPortion

	payment := &models.LoanPayment{
		ID:              uuid.New().String(),
		LoanID:          loanID,
		Amount:          amount,
		PrincipalAmount: principalPortion,
		InterestAmount:  interestPortion,
		PaymentMethod:   paymentMethod,
		PaidAt:          time.Now(),
		CreatedAt:       time.Now(),
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert payment
	paymentQuery := `
		INSERT INTO loan_payments (
			id, loan_id, amount, principal_amount, interest_amount,
			payment_method, reference, paid_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(paymentQuery,
		payment.ID, payment.LoanID, payment.Amount, payment.PrincipalAmount,
		payment.InterestAmount, payment.PaymentMethod, payment.Reference,
		payment.PaidAt, payment.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to record payment: %w", err)
	}

	// Update loan
	newPaidAmount := loan.PaidAmount + amount
	newRemainingAmount := loan.RemainingAmount - amount
	newStatus := loan.Status

	if newRemainingAmount <= 0 {
		newStatus = models.LoanStatusCompleted
		newRemainingAmount = 0
	}

	updateLoanQuery := `
		UPDATE loans
		SET paid_amount = ?, remaining_amount = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.Exec(updateLoanQuery,
		newPaidAmount, newRemainingAmount, newStatus, time.Now(), loanID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update loan: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return payment, nil
}

// GetLoanByID retrieves a loan by ID
func (s *LoanService) GetLoanByID(loanID string) (*models.Loan, error) {
	query := `
		SELECT id, borrower_id, chama_id, type, amount, interest_rate, duration,
			   purpose, status, approved_by, approved_at, disbursed_at, due_date,
			   total_amount, paid_amount, remaining_amount, required_guarantors,
			   approved_guarantors, created_at, updated_at
		FROM loans WHERE id = ?
	`

	loan := &models.Loan{}
	err := s.db.QueryRow(query, loanID).Scan(
		&loan.ID, &loan.BorrowerID, &loan.ChamaID, &loan.Type, &loan.Amount,
		&loan.InterestRate, &loan.Duration, &loan.Purpose, &loan.Status,
		&loan.ApprovedBy, &loan.ApprovedAt, &loan.DisbursedAt, &loan.DueDate,
		&loan.TotalAmount, &loan.PaidAmount, &loan.RemainingAmount,
		&loan.RequiredGuarantors, &loan.ApprovedGuarantors,
		&loan.CreatedAt, &loan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("loan not found")
		}
		return nil, fmt.Errorf("failed to get loan: %w", err)
	}

	return loan, nil
}

// GetChamaLoans retrieves loans for a chama
func (s *LoanService) GetChamaLoans(chamaID string, status *models.LoanStatus, limit, offset int) ([]*models.Loan, error) {
	query := `
		SELECT l.id, l.borrower_id, l.chama_id, l.type, l.amount, l.interest_rate,
			   l.duration, l.purpose, l.status, l.approved_by, l.approved_at,
			   l.disbursed_at, l.due_date, l.total_amount, l.paid_amount,
			   l.remaining_amount, l.required_guarantors, l.approved_guarantors,
			   l.created_at, l.updated_at,
			   u.first_name, u.last_name, u.avatar
		FROM loans l
		INNER JOIN users u ON l.borrower_id = u.id
		WHERE l.chama_id = ?
	`
	args := []interface{}{chamaID}

	if status != nil {
		query += " AND l.status = ?"
		args = append(args, *status)
	}

	query += " ORDER BY l.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama loans: %w", err)
	}
	defer rows.Close()

	var loans []*models.Loan
	for rows.Next() {
		loan := &models.Loan{}
		borrower := &models.User{}

		err := rows.Scan(
			&loan.ID, &loan.BorrowerID, &loan.ChamaID, &loan.Type, &loan.Amount,
			&loan.InterestRate, &loan.Duration, &loan.Purpose, &loan.Status,
			&loan.ApprovedBy, &loan.ApprovedAt, &loan.DisbursedAt, &loan.DueDate,
			&loan.TotalAmount, &loan.PaidAmount, &loan.RemainingAmount,
			&loan.RequiredGuarantors, &loan.ApprovedGuarantors,
			&loan.CreatedAt, &loan.UpdatedAt,
			&borrower.FirstName, &borrower.LastName, &borrower.Avatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan: %w", err)
		}

		borrower.ID = loan.BorrowerID
		loan.Borrower = borrower
		loans = append(loans, loan)
	}

	return loans, nil
}

// Helper methods

func (s *LoanService) hasActiveLoan(borrowerID, chamaID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM loans
		WHERE borrower_id = ? AND chama_id = ? AND status IN (?, ?)
	`
	var count int
	err := s.db.QueryRow(query, borrowerID, chamaID, models.LoanStatusApproved, models.LoanStatusActive).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active loans: %w", err)
	}
	return count > 0, nil
}

func (s *LoanService) getGuarantor(loanID, userID string) (*models.Guarantor, error) {
	query := `
		SELECT id, loan_id, user_id, amount, status, message, responded_at, created_at
		FROM guarantors WHERE loan_id = ? AND user_id = ?
	`

	guarantor := &models.Guarantor{}
	err := s.db.QueryRow(query, loanID, userID).Scan(
		&guarantor.ID, &guarantor.LoanID, &guarantor.UserID, &guarantor.Amount,
		&guarantor.Status, &guarantor.Message, &guarantor.RespondedAt, &guarantor.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("guarantor not found")
		}
		return nil, fmt.Errorf("failed to get guarantor: %w", err)
	}

	return guarantor, nil
}

func (s *LoanService) disburseLoan(tx *sql.Tx, loan *models.Loan, _ float64) error {
	now := time.Now()
	dueDate := now.AddDate(0, loan.Duration, 0)

	// Update loan with disbursement details
	_, err := tx.Exec(
		"UPDATE loans SET status = ?, disbursed_at = ?, due_date = ? WHERE id = ?",
		models.LoanStatusActive, now, dueDate, loan.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update loan disbursement: %w", err)
	}

	// Transfer funds from chama wallet to borrower's wallet
	// This would integrate with the wallet service
	// For now, we'll just log the transaction

	return nil
}

func (s *LoanService) notifyLoanReadyForApproval(loan *models.Loan) {
	// Get chama leaders
	query := `
		SELECT user_id FROM chama_members
		WHERE chama_id = ? AND role IN (?, ?) AND is_active = true
	`

	rows, err := s.db.Query(query, loan.ChamaID, models.ChamaRoleChairperson, models.ChamaRoleTreasurer)
	if err != nil {
		return
	}
	defer rows.Close()

	notificationService := NewNotificationService(s.db, nil)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}

		go func(uid string) {
			title := "Loan Ready for Approval"
			message := fmt.Sprintf("A loan application for KSh %.2f is ready for your approval", loan.Amount)
			data := map[string]interface{}{
				"type":   "loan_approval_needed",
				"loanId": loan.ID,
				"amount": loan.Amount,
			}
			notificationService.CreateNotification(uid, "loan", title, message, data, true, true, false)
		}(userID)
	}
}
