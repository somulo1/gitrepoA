package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"vaultke-backend/internal/models"

	"github.com/google/uuid"
)

// SharesService handles shares-related business logic
type SharesService struct {
	db *sql.DB
}

// NewSharesService creates a new shares service
func NewSharesService(db *sql.DB) *SharesService {
	return &SharesService{db: db}
}

// CreateShareOffering creates a new share offering for a chama
func (s *SharesService) CreateShareOffering(chamaID, userID string, req *models.CreateShareOfferingRequest) (*models.ShareOffering, error) {
	// Validate user is member of chama and has permission to create offerings
	if !s.isMemberOfChama(userID, chamaID) {
		return nil, fmt.Errorf("user is not a member of this chama")
	}

	// Generate unique ID and security hash
	offeringID := uuid.New().String()
	securityHash := uuid.New().String()
	now := time.Now()

	offering := &models.ShareOffering{
		ID:                  offeringID,
		ChamaID:             chamaID,
		Name:                req.Name,
		ShareType:           req.ShareType,
		TotalShares:         req.TotalShares,
		PricePerShare:       req.PricePerShare,
		MinimumPurchase:     req.MinimumPurchase,
		Description:         req.Description,
		EligibilityCriteria: req.EligibilityCriteria,
		ApprovalRequired:    req.ApprovalRequired,
		TotalValue:          float64(req.TotalShares) * req.PricePerShare,
		CreatedBy:           userID,
		CreatedByID:         userID,
		Timestamp:           now,
		Status:              "active",
		TransactionID:       uuid.New().String(),
		SecurityHash:        securityHash,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// Insert into database
	query := `
		INSERT INTO share_offerings (
			id, chama_id, name, share_type, total_shares, price_per_share, minimum_purchase,
			description, eligibility_criteria, approval_required, total_value,
			created_by, created_by_id, timestamp, status, transaction_id, security_hash,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		offering.ID,
		offering.ChamaID,
		offering.Name,
		offering.ShareType,
		offering.TotalShares,
		offering.PricePerShare,
		offering.MinimumPurchase,
		offering.Description,
		offering.EligibilityCriteria,
		offering.ApprovalRequired,
		offering.TotalValue,
		offering.CreatedBy,
		offering.CreatedByID,
		offering.Timestamp,
		offering.Status,
		offering.TransactionID,
		offering.SecurityHash,
		offering.CreatedAt,
		offering.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create share offering: %w", err)
	}

	log.Printf("Created share offering %s (%s) for chama %s", offeringID, req.Name, chamaID)
	return offering, nil
}

// CreateShares creates new shares for a member
func (s *SharesService) CreateShares(chamaID string, req *models.CreateShareRequest) (*models.Share, error) {
	// Validate member is part of the chama
	if !s.isMemberOfChama(req.MemberID, chamaID) {
		return nil, fmt.Errorf("member is not part of this chama")
	}

	// Generate unique ID
	shareID := uuid.New().String()
	now := time.Now()

	share := &models.Share{
		ID:                shareID,
		ChamaID:           chamaID,
		MemberID:          req.MemberID,
		Name:              req.Name,
		ShareType:         req.ShareType,
		SharesOwned:       req.SharesCount,
		ShareValue:        req.ShareValue,
		PurchaseDate:      req.PurchaseDate,
		CertificateNumber: req.CertificateNumber,
		Status:            models.ShareStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Calculate total value
	share.CalculateTotalValue()

	// Insert into database
	query := `
		INSERT INTO shares (
			id, chama_id, member_id, name, share_type, shares_owned, share_value,
			total_value, purchase_date, certificate_number, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		share.ID,
		share.ChamaID,
		share.MemberID,
		share.Name,
		share.ShareType,
		share.SharesOwned,
		share.ShareValue,
		share.TotalValue,
		share.PurchaseDate,
		share.CertificateNumber,
		share.Status,
		share.CreatedAt,
		share.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create shares: %w", err)
	}

	// Create share transaction record
	_, err = s.createShareTransaction(chamaID, &models.CreateShareTransactionRequest{
		ToMemberID:      &req.MemberID,
		TransactionType: models.ShareTransactionPurchase,
		SharesCount:     req.SharesCount,
		ShareValue:      req.ShareValue,
		TransactionDate: req.PurchaseDate,
		Description:     stringPtr("Initial share purchase"),
	})

	if err != nil {
		log.Printf("Warning: Failed to create share transaction record: %v", err)
	}

	log.Printf("Created shares %s for member %s in chama %s", shareID, req.MemberID, chamaID)
	return share, nil
}

// GetChamaShares retrieves all share offerings for a chama
func (s *SharesService) GetChamaShares(chamaID string, limit, offset int) ([]models.ShareOffering, error) {
	query := `
		SELECT id, chama_id, name, share_type, total_shares, price_per_share, minimum_purchase,
			   description, eligibility_criteria, approval_required, total_value,
			   created_by, created_by_id, timestamp, status, transaction_id, security_hash,
			   created_at, updated_at
		FROM share_offerings
		WHERE chama_id = ? AND status IN ('active', 'pending_approval')
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get chama share offerings: %w", err)
	}
	defer rows.Close()

	var offerings []models.ShareOffering
	for rows.Next() {
		var offering models.ShareOffering

		err := rows.Scan(
			&offering.ID,
			&offering.ChamaID,
			&offering.Name,
			&offering.ShareType,
			&offering.TotalShares,
			&offering.PricePerShare,
			&offering.MinimumPurchase,
			&offering.Description,
			&offering.EligibilityCriteria,
			&offering.ApprovalRequired,
			&offering.TotalValue,
			&offering.CreatedBy,
			&offering.CreatedByID,
			&offering.Timestamp,
			&offering.Status,
			&offering.TransactionID,
			&offering.SecurityHash,
			&offering.CreatedAt,
			&offering.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share offering: %w", err)
		}

		offerings = append(offerings, offering)
	}

	return offerings, nil
}

// GetMemberShares retrieves all shares for a specific member in a chama
func (s *SharesService) GetMemberShares(chamaID, memberID string) ([]models.Share, error) {
	query := `
		SELECT id, chama_id, member_id, name, share_type, shares_owned, share_value,
			   total_value, purchase_date, certificate_number, status, created_at, updated_at
		FROM shares
		WHERE chama_id = ? AND member_id = ? AND status = 'active'
		ORDER BY purchase_date DESC
	`

	rows, err := s.db.Query(query, chamaID, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get member shares: %w", err)
	}
	defer rows.Close()

	var shares []models.Share
	for rows.Next() {
		var share models.Share
		err := rows.Scan(
			&share.ID,
			&share.ChamaID,
			&share.MemberID,
			&share.Name,
			&share.ShareType,
			&share.SharesOwned,
			&share.ShareValue,
			&share.TotalValue,
			&share.PurchaseDate,
			&share.CertificateNumber,
			&share.Status,
			&share.CreatedAt,
			&share.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share: %w", err)
		}
		shares = append(shares, share)
	}

	return shares, nil
}

// GetChamaSharesSummary retrieves aggregated share information for a chama
func (s *SharesService) GetChamaSharesSummary(chamaID string) ([]models.ShareSummary, error) {
	query := `
		SELECT s.member_id, u.first_name, u.last_name, 
			   SUM(s.shares_owned) as total_shares, 
			   SUM(s.total_value) as total_value,
			   MAX(s.purchase_date) as last_purchase
		FROM shares s
		JOIN users u ON s.member_id = u.id
		WHERE s.chama_id = ? AND s.status = 'active'
		GROUP BY s.member_id, u.first_name, u.last_name
		ORDER BY total_shares DESC
	`

	rows, err := s.db.Query(query, chamaID)
	if err != nil {
		log.Printf("Error querying shares summary for chama %s: %v", chamaID, err)
		return nil, fmt.Errorf("failed to get chama shares summary: %w", err)
	}
	defer rows.Close()

	var summaries []models.ShareSummary
	for rows.Next() {
		var summary models.ShareSummary
		var firstName, lastName string
		var lastPurchase sql.NullTime

		err := rows.Scan(
			&summary.MemberID,
			&firstName,
			&lastName,
			&summary.TotalShares,
			&summary.TotalValue,
			&lastPurchase,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share summary: %w", err)
		}

		summary.MemberName = firstName + " " + lastName
		if lastPurchase.Valid {
			summary.LastPurchase = &lastPurchase.Time
		}

		// Get detailed share types for this member
		shareTypes, err := s.GetMemberShares(chamaID, summary.MemberID)
		if err != nil {
			log.Printf("Warning: Failed to get share types for member %s: %v", summary.MemberID, err)
		} else {
			summary.ShareTypes = shareTypes
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// UpdateShares updates existing shares
func (s *SharesService) UpdateShares(shareID string, req *models.UpdateShareRequest) (*models.Share, error) {
	// Get existing share
	existing, err := s.getShareByID(shareID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.SharesOwned != nil {
		existing.SharesOwned = *req.SharesOwned
	}
	if req.ShareValue != nil {
		existing.ShareValue = *req.ShareValue
	}
	if req.CertificateNumber != nil {
		existing.CertificateNumber = req.CertificateNumber
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}

	// Recalculate total value
	existing.CalculateTotalValue()
	existing.UpdatedAt = time.Now()

	// Update in database
	query := `
		UPDATE shares 
		SET shares_owned = ?, share_value = ?, total_value = ?, 
			certificate_number = ?, status = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(
		query,
		existing.SharesOwned,
		existing.ShareValue,
		existing.TotalValue,
		existing.CertificateNumber,
		existing.Status,
		existing.UpdatedAt,
		shareID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update shares: %w", err)
	}

	log.Printf("Updated shares %s", shareID)
	return existing, nil
}

// Helper functions
func (s *SharesService) getShareByID(shareID string) (*models.Share, error) {
	query := `
		SELECT id, chama_id, member_id, name, share_type, shares_owned, share_value,
			   total_value, purchase_date, certificate_number, status, created_at, updated_at
		FROM shares WHERE id = ?
	`

	var share models.Share
	err := s.db.QueryRow(query, shareID).Scan(
		&share.ID,
		&share.ChamaID,
		&share.MemberID,
		&share.Name,
		&share.ShareType,
		&share.SharesOwned,
		&share.ShareValue,
		&share.TotalValue,
		&share.PurchaseDate,
		&share.CertificateNumber,
		&share.Status,
		&share.CreatedAt,
		&share.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("share not found")
		}
		return nil, fmt.Errorf("failed to get share: %w", err)
	}

	return &share, nil
}

func (s *SharesService) isMemberOfChama(memberID, chamaID string) bool {
	query := `SELECT 1 FROM chama_members WHERE user_id = ? AND chama_id = ? AND is_active = TRUE`
	var exists int
	err := s.db.QueryRow(query, memberID, chamaID).Scan(&exists)
	return err == nil
}

func (s *SharesService) createShareTransaction(chamaID string, req *models.CreateShareTransactionRequest) (*models.ShareTransaction, error) {
	transactionID := uuid.New().String()
	now := time.Now()

	transaction := &models.ShareTransaction{
		ID:              transactionID,
		ChamaID:         chamaID,
		FromMemberID:    req.FromMemberID,
		ToMemberID:      req.ToMemberID,
		TransactionType: req.TransactionType,
		SharesCount:     req.SharesCount,
		ShareValue:      req.ShareValue,
		TotalAmount:     float64(req.SharesCount) * req.ShareValue,
		TransactionDate: req.TransactionDate,
		Status:          models.ShareTransactionCompleted,
		Description:     req.Description,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	query := `
		INSERT INTO share_transactions (
			id, chama_id, from_member_id, to_member_id, transaction_type,
			shares_count, share_value, total_amount, transaction_date, status,
			description, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(
		query,
		transaction.ID,
		transaction.ChamaID,
		transaction.FromMemberID,
		transaction.ToMemberID,
		transaction.TransactionType,
		transaction.SharesCount,
		transaction.ShareValue,
		transaction.TotalAmount,
		transaction.TransactionDate,
		transaction.Status,
		transaction.Description,
		transaction.CreatedAt,
		transaction.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create share transaction: %w", err)
	}

	return transaction, nil
}

// GetShareTransactions retrieves share transactions for a chama
func (s *SharesService) GetShareTransactions(chamaID string, limit, offset int) ([]models.ShareTransaction, error) {
	query := `
		SELECT id, chama_id, from_member_id, to_member_id, transaction_type,
			   shares_count, share_value, total_amount, transaction_date, status,
			   approved_by, description, created_at, updated_at
		FROM share_transactions
		WHERE chama_id = ?
		ORDER BY transaction_date DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, chamaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get share transactions: %w", err)
	}
	defer rows.Close()

	var transactions []models.ShareTransaction
	for rows.Next() {
		var transaction models.ShareTransaction
		err := rows.Scan(
			&transaction.ID,
			&transaction.ChamaID,
			&transaction.FromMemberID,
			&transaction.ToMemberID,
			&transaction.TransactionType,
			&transaction.SharesCount,
			&transaction.ShareValue,
			&transaction.TotalAmount,
			&transaction.TransactionDate,
			&transaction.Status,
			&transaction.ApprovedBy,
			&transaction.Description,
			&transaction.CreatedAt,
			&transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan share transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// BuyShares allows a member to purchase shares from an offering
func (s *SharesService) BuyShares(chamaID, userID string, req *models.BuySharesRequest) (*models.Share, error) {
	// Get the share offering
	offering, err := s.getShareOfferingByID(req.OfferingID)
	if err != nil {
		return nil, fmt.Errorf("share offering not found: %w", err)
	}

	// Validate offering belongs to the chama
	if offering.ChamaID != chamaID {
		return nil, fmt.Errorf("share offering does not belong to this chama")
	}

	// Validate offering is active
	if offering.Status != "active" {
		return nil, fmt.Errorf("share offering is not active")
	}

	// Check if user is member of chama
	if !s.isMemberOfChama(userID, chamaID) {
		return nil, fmt.Errorf("user is not a member of this chama")
	}

	// Validate quantity
	if req.Quantity < offering.MinimumPurchase {
		return nil, fmt.Errorf("minimum purchase is %d shares", offering.MinimumPurchase)
	}

	if req.Quantity > offering.TotalShares {
		return nil, fmt.Errorf("not enough shares available")
	}

	// Validate price matches offering
	if req.PricePerShare != offering.PricePerShare {
		return nil, fmt.Errorf("price per share does not match offering price")
	}

	// Validate total amount
	expectedTotal := float64(req.Quantity) * offering.PricePerShare
	if req.TotalAmount != expectedTotal {
		return nil, fmt.Errorf("total amount does not match expected amount")
	}

	totalAmount := req.TotalAmount

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Handle payment deduction
	if req.PaymentMethod == "wallet" {
		err = s.deductFromWallet(tx, userID, totalAmount)
		if err != nil {
			return nil, fmt.Errorf("payment failed: %w", err)
		}
	} else if req.PaymentMethod == "mpesa" || req.PaymentMethod == "mobile_money" {
		// For M-Pesa, we would initiate the payment process
		// For now, we'll assume it's handled externally
		// In a real implementation, this would integrate with M-Pesa API
		log.Printf("M-Pesa payment initiated for amount: %.2f", totalAmount)
	} else if req.PaymentMethod == "bank_transfer" || req.PaymentMethod == "cash" {
		// For bank transfer and cash, payment is handled externally
		log.Printf("%s payment recorded for amount: %.2f", req.PaymentMethod, totalAmount)
	}

	// Generate certificate number
	certificateNumber := s.generateCertificateNumber()

	// Create share record
	now := time.Now()
	share := &models.Share{
		ID:                uuid.New().String(),
		ChamaID:           chamaID,
		MemberID:          userID,
		Name:              offering.Name, // Set the share name from the offering
		ShareType:         models.ShareType(offering.ShareType),
		SharesOwned:       req.Quantity,
		ShareValue:        offering.PricePerShare,
		PurchaseDate:      req.PurchaseDate,
		CertificateNumber: &certificateNumber,
		Status:            models.ShareStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Calculate total value
	share.CalculateTotalValue()

	// Insert share record
	shareQuery := `
		INSERT INTO shares (
			id, chama_id, member_id, name, share_type, shares_owned, share_value,
			total_value, purchase_date, certificate_number, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(
		shareQuery,
		share.ID, share.ChamaID, share.MemberID, share.Name, share.ShareType, share.SharesOwned,
		share.ShareValue, share.TotalValue, share.PurchaseDate, share.CertificateNumber,
		share.Status, share.CreatedAt, share.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create share record: %w", err)
	}

	// Update offering (reduce available shares)
	updateOfferingQuery := `
		UPDATE share_offerings
		SET total_shares = total_shares - ?, updated_at = ?
		WHERE id = ?
	`

	_, err = tx.Exec(updateOfferingQuery, req.Quantity, now, req.OfferingID)
	if err != nil {
		return nil, fmt.Errorf("failed to update share offering: %w", err)
	}

	// Add to chama wallet (if wallet payment)
	if req.PaymentMethod == "wallet" {
		err = s.addToChamaWallet(tx, chamaID, totalAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to update chama wallet: %w", err)
		}
	}

	// Record transaction
	err = s.recordSharePurchaseTransaction(tx, chamaID, userID, share.ID, req.Quantity, offering.PricePerShare, totalAmount, req.PaymentMethod, req.Notes)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully purchased %d shares for user %s in chama %s", req.Quantity, userID, chamaID)
	return share, nil
}

// BuyDividends allows a member to purchase dividend certificates
func (s *SharesService) BuyDividends(chamaID, userID string, req *models.BuyDividendsRequest) (*models.Share, error) {
	// Get the dividend declaration
	declaration, err := s.getDividendDeclarationByID(req.DeclarationID)
	if err != nil {
		return nil, fmt.Errorf("dividend declaration not found: %w", err)
	}

	// Validate declaration belongs to the chama
	if declaration.ChamaID != chamaID {
		return nil, fmt.Errorf("dividend declaration does not belong to this chama")
	}

	// Validate declaration is approved
	if declaration.Status != models.DividendStatusApproved {
		return nil, fmt.Errorf("dividend declaration is not approved for purchase")
	}

	// Check if user is member of chama
	if !s.isMemberOfChama(userID, chamaID) {
		return nil, fmt.Errorf("user is not a member of this chama")
	}

	// Validate quantity (for dividends, quantity represents number of certificates)
	if req.Quantity < 1 {
		return nil, fmt.Errorf("minimum quantity is 1")
	}

	// Validate price matches declaration
	if req.PricePerShare != declaration.DividendPerShare {
		return nil, fmt.Errorf("price per share does not match declaration")
	}

	// Validate total amount
	expectedTotal := float64(req.Quantity) * declaration.DividendPerShare
	if req.TotalAmount != expectedTotal {
		return nil, fmt.Errorf("total amount does not match expected amount")
	}

	totalAmount := req.TotalAmount

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Handle payment deduction
	if req.PaymentMethod == "wallet" {
		err = s.deductFromWallet(tx, userID, totalAmount)
		if err != nil {
			return nil, fmt.Errorf("payment failed: %w", err)
		}
	} else if req.PaymentMethod == "mpesa" || req.PaymentMethod == "mobile_money" {
		log.Printf("M-Pesa payment initiated for dividend certificates: %.2f", totalAmount)
	} else if req.PaymentMethod == "bank_transfer" || req.PaymentMethod == "cash" {
		log.Printf("%s payment recorded for dividend certificates: %.2f", req.PaymentMethod, totalAmount)
	}

	// Generate certificate number
	certificateNumber := s.generateCertificateNumber()

	// Create dividend certificate record (stored as share with dividend type)
	now := time.Now()
	share := &models.Share{
		ID:                uuid.New().String(),
		ChamaID:           chamaID,
		MemberID:          userID,
		Name:              "Dividend Certificate",       // Set name for dividend certificates
		ShareType:         models.ShareType("dividend"), // Special type for dividend certificates
		SharesOwned:       req.Quantity,
		ShareValue:        declaration.DividendPerShare,
		PurchaseDate:      req.PurchaseDate,
		CertificateNumber: &certificateNumber,
		Status:            models.ShareStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Calculate total value
	share.CalculateTotalValue()

	// Insert dividend certificate record
	shareQuery := `
		INSERT INTO shares (
			id, chama_id, member_id, name, share_type, shares_owned, share_value,
			total_value, purchase_date, certificate_number, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(
		shareQuery,
		share.ID, share.ChamaID, share.MemberID, share.Name, share.ShareType, share.SharesOwned,
		share.ShareValue, share.TotalValue, share.PurchaseDate, share.CertificateNumber,
		share.Status, share.CreatedAt, share.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dividend certificate record: %w", err)
	}

	// Add to chama wallet (if wallet payment)
	if req.PaymentMethod == "wallet" {
		err = s.addToChamaWallet(tx, chamaID, totalAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to update chama wallet: %w", err)
		}
	}

	// Record transaction
	err = s.recordDividendPurchaseTransaction(tx, chamaID, userID, share.ID, req.Quantity, declaration.DividendPerShare, totalAmount, req.PaymentMethod, req.Notes)
	if err != nil {
		return nil, fmt.Errorf("failed to record transaction: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully purchased %d dividend certificates for user %s in chama %s", req.Quantity, userID, chamaID)
	return share, nil
}

// Helper methods for BuyShares

func (s *SharesService) getShareOfferingByID(offeringID string) (*models.ShareOffering, error) {
	query := `
		SELECT id, chama_id, name, share_type, total_shares, price_per_share, minimum_purchase,
			   description, eligibility_criteria, approval_required, total_value,
			   created_by, created_by_id, timestamp, status, transaction_id, security_hash,
			   created_at, updated_at
		FROM share_offerings WHERE id = ?
	`

	var offering models.ShareOffering
	err := s.db.QueryRow(query, offeringID).Scan(
		&offering.ID, &offering.ChamaID, &offering.Name, &offering.ShareType, &offering.TotalShares,
		&offering.PricePerShare, &offering.MinimumPurchase, &offering.Description,
		&offering.EligibilityCriteria, &offering.ApprovalRequired, &offering.TotalValue,
		&offering.CreatedBy, &offering.CreatedByID, &offering.Timestamp, &offering.Status,
		&offering.TransactionID, &offering.SecurityHash, &offering.CreatedAt, &offering.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &offering, nil
}

func (s *SharesService) deductFromWallet(tx *sql.Tx, userID string, amount float64) error {
	// Check balance
	var balance float64
	err := tx.QueryRow("SELECT balance FROM wallets WHERE owner_id = ? AND type = 'personal'", userID).Scan(&balance)
	if err != nil {
		return fmt.Errorf("failed to check wallet balance: %w", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	// Deduct amount
	_, err = tx.Exec(
		"UPDATE wallets SET balance = balance - ?, updated_at = ? WHERE owner_id = ? AND type = 'personal'",
		amount, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("failed to deduct from wallet: %w", err)
	}

	return nil
}

func (s *SharesService) addToChamaWallet(tx *sql.Tx, chamaID string, amount float64) error {
	// Check if chama wallet exists
	var exists bool
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM wallets WHERE owner_id = ? AND type = 'chama')", chamaID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check chama wallet: %w", err)
	}

	if !exists {
		// Create chama wallet
		walletID := uuid.New().String()
		_, err = tx.Exec(
			"INSERT INTO wallets (id, owner_id, type, balance, created_at, updated_at) VALUES (?, ?, 'chama', ?, ?, ?)",
			walletID, chamaID, amount, time.Now(), time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to create chama wallet: %w", err)
		}
	} else {
		// Update existing wallet
		_, err = tx.Exec(
			"UPDATE wallets SET balance = balance + ?, updated_at = ? WHERE owner_id = ? AND type = 'chama'",
			amount, time.Now(), chamaID,
		)
		if err != nil {
			return fmt.Errorf("failed to update chama wallet: %w", err)
		}
	}

	return nil
}

func (s *SharesService) recordSharePurchaseTransaction(tx *sql.Tx, chamaID, userID, shareID string, quantity int, pricePerShare, totalAmount float64, paymentMethod string, notes *string) error {
	transactionID := uuid.New().String()
	now := time.Now()

	metadata := map[string]interface{}{
		"shareId":         shareID,
		"quantity":        quantity,
		"pricePerShare":   pricePerShare,
		"paymentMethod":   paymentMethod,
		"transactionType": "share_purchase",
	}
	if notes != nil {
		metadata["notes"] = *notes
	}

	metadataJSON, _ := json.Marshal(metadata)

	_, err := tx.Exec(`
		INSERT INTO transactions (
			id, type, amount, currency, description, status, payment_method,
			initiated_by, recipient_id, metadata, created_at, updated_at
		) VALUES (?, 'share_purchase', ?, 'KES', ?, 'completed', ?, ?, ?, ?, ?, ?)
	`,
		transactionID, totalAmount, fmt.Sprintf("Purchase of %d shares", quantity),
		paymentMethod, userID, chamaID, string(metadataJSON), now, now,
	)

	return err
}

func (s *SharesService) getDividendDeclarationByID(declarationID string) (*models.DividendDeclaration, error) {
	query := `
		SELECT id, chama_id, declaration_date, dividend_per_share, total_dividend_amount,
			   payment_date, status, declared_by, approved_by, description, created_at, updated_at
		FROM dividend_declarations WHERE id = ?
	`

	var declaration models.DividendDeclaration
	err := s.db.QueryRow(query, declarationID).Scan(
		&declaration.ID, &declaration.ChamaID, &declaration.DeclarationDate,
		&declaration.DividendPerShare, &declaration.TotalDividendAmount,
		&declaration.PaymentDate, &declaration.Status, &declaration.DeclaredBy,
		&declaration.ApprovedBy, &declaration.Description, &declaration.CreatedAt, &declaration.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &declaration, nil
}

func (s *SharesService) recordDividendPurchaseTransaction(tx *sql.Tx, chamaID, userID, shareID string, quantity int, pricePerShare, totalAmount float64, paymentMethod string, notes *string) error {
	transactionID := uuid.New().String()
	now := time.Now()

	metadata := map[string]interface{}{
		"shareId":         shareID,
		"quantity":        quantity,
		"pricePerShare":   pricePerShare,
		"paymentMethod":   paymentMethod,
		"transactionType": "dividend_certificate_purchase",
	}
	if notes != nil {
		metadata["notes"] = *notes
	}

	metadataJSON, _ := json.Marshal(metadata)

	_, err := tx.Exec(`
		INSERT INTO transactions (
			id, type, amount, currency, description, status, payment_method,
			initiated_by, recipient_id, metadata, created_at, updated_at
		) VALUES (?, 'dividend_purchase', ?, 'KES', ?, 'completed', ?, ?, ?, ?, ?, ?)
	`,
		transactionID, totalAmount, fmt.Sprintf("Purchase of %d dividend certificates", quantity),
		paymentMethod, userID, chamaID, string(metadataJSON), now, now,
	)

	return err
}

func (s *SharesService) generateCertificateNumber() string {
	// Generate a unique certificate number
	timestamp := time.Now().Unix()
	random := uuid.New().String()[:8]
	return fmt.Sprintf("CERT-%d-%s", timestamp, random)
}

// TransferShares allows a member to transfer shares to another member
func (s *SharesService) TransferShares(chamaID, fromUserID string, req *models.TransferSharesRequest) error {
	// Get the share record
	share, err := s.getShareByID(req.ShareID)
	if err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	// Validate share belongs to the chama
	if share.ChamaID != chamaID {
		return fmt.Errorf("share does not belong to this chama")
	}

	// Validate share belongs to the seller
	if share.MemberID != fromUserID {
		return fmt.Errorf("share does not belong to the seller")
	}

	// Validate share can be transferred
	if !share.CanTransfer() {
		return fmt.Errorf("share cannot be transferred")
	}

	// Validate buyer is a member of the chama
	if !s.isMemberOfChama(req.ToMemberID, chamaID) {
		return fmt.Errorf("buyer is not a member of this chama")
	}

	// Validate shares count
	if req.SharesCount > share.SharesOwned {
		return fmt.Errorf("insufficient shares available for transfer")
	}

	// Validate transfer amount
	expectedTotal := float64(req.SharesCount) * req.TransferPrice
	if req.TotalAmount != expectedTotal {
		return fmt.Errorf("total amount does not match expected amount")
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Deduct from buyer's wallet
	err = s.deductFromWallet(tx, req.ToMemberID, req.TotalAmount)
	if err != nil {
		return fmt.Errorf("payment failed: %w", err)
	}

	// Add to seller's wallet
	err = s.addToWallet(tx, fromUserID, req.TotalAmount)
	if err != nil {
		return fmt.Errorf("failed to credit seller: %w", err)
	}

	now := time.Now()

	// Handle share transfer based on quantity
	if req.SharesCount == share.SharesOwned {
		// Transfer all shares - update ownership
		_, err = tx.Exec(
			"UPDATE shares SET member_id = ?, status = 'transferred', updated_at = ? WHERE id = ?",
			req.ToMemberID, now, req.ShareID,
		)
		if err != nil {
			return fmt.Errorf("failed to transfer share ownership: %w", err)
		}

		// Create new active share record for buyer
		newShareID := uuid.New().String()
		_, err = tx.Exec(
			`INSERT INTO shares (
				id, chama_id, member_id, name, share_type, shares_owned, share_value,
				total_value, purchase_date, certificate_number, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
			newShareID, share.ChamaID, req.ToMemberID, share.Name, share.ShareType,
			req.SharesCount, req.TransferPrice, req.TotalAmount, req.TransferDate,
			share.CertificateNumber, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to create new share record for buyer: %w", err)
		}
	} else {
		// Partial transfer - reduce seller's shares and create new record for buyer
		newSharesOwned := share.SharesOwned - req.SharesCount

		// Update seller's share record
		_, err = tx.Exec(
			"UPDATE shares SET shares_owned = ?, total_value = ?, updated_at = ? WHERE id = ?",
			newSharesOwned, float64(newSharesOwned)*share.ShareValue, now, req.ShareID,
		)
		if err != nil {
			return fmt.Errorf("failed to update seller's share record: %w", err)
		}

		// Create new share record for buyer
		newShareID := uuid.New().String()
		_, err = tx.Exec(
			`INSERT INTO shares (
				id, chama_id, member_id, name, share_type, shares_owned, share_value,
				total_value, purchase_date, certificate_number, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
			newShareID, share.ChamaID, req.ToMemberID, share.Name, share.ShareType,
			req.SharesCount, req.TransferPrice, req.TotalAmount, req.TransferDate,
			share.CertificateNumber, now, now,
		)
		if err != nil {
			return fmt.Errorf("failed to create new share record for buyer: %w", err)
		}
	}

	// Create share transaction record
	err = s.createShareTransactionInTx(tx, chamaID, &models.CreateShareTransactionRequest{
		FromMemberID:    &fromUserID,
		ToMemberID:      &req.ToMemberID,
		TransactionType: models.ShareTransactionTransfer,
		SharesCount:     req.SharesCount,
		ShareValue:      req.TransferPrice,
		TransactionDate: req.TransferDate,
		Description:     req.Notes,
	})
	if err != nil {
		return fmt.Errorf("failed to record share transaction: %w", err)
	}

	// Record wallet transaction
	err = s.recordShareTransferTransaction(tx, chamaID, fromUserID, req.ToMemberID, req.TotalAmount, req.Notes)
	if err != nil {
		return fmt.Errorf("failed to record wallet transaction: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully transferred %d shares from user %s to user %s in chama %s", req.SharesCount, fromUserID, req.ToMemberID, chamaID)
	return nil
}

// Helper method to add money to a user's wallet
func (s *SharesService) addToWallet(tx *sql.Tx, userID string, amount float64) error {
	// Check if wallet exists
	var exists bool
	err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM wallets WHERE owner_id = ? AND type = 'personal')", userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check wallet: %w", err)
	}

	if !exists {
		// Create wallet
		walletID := uuid.New().String()
		_, err = tx.Exec(
			"INSERT INTO wallets (id, owner_id, type, balance, created_at, updated_at) VALUES (?, ?, 'personal', ?, ?, ?)",
			walletID, userID, amount, time.Now(), time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to create wallet: %w", err)
		}
	} else {
		// Update existing wallet
		_, err = tx.Exec(
			"UPDATE wallets SET balance = balance + ?, updated_at = ? WHERE owner_id = ? AND type = 'personal'",
			amount, time.Now(), userID,
		)
		if err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}
	}

	return nil
}

// Helper method to create share transaction within a transaction
func (s *SharesService) createShareTransactionInTx(tx *sql.Tx, chamaID string, req *models.CreateShareTransactionRequest) error {
	transactionID := uuid.New().String()
	now := time.Now()

	transaction := &models.ShareTransaction{
		ID:              transactionID,
		ChamaID:         chamaID,
		FromMemberID:    req.FromMemberID,
		ToMemberID:      req.ToMemberID,
		TransactionType: req.TransactionType,
		SharesCount:     req.SharesCount,
		ShareValue:      req.ShareValue,
		TotalAmount:     float64(req.SharesCount) * req.ShareValue,
		TransactionDate: req.TransactionDate,
		Status:          models.ShareTransactionCompleted,
		Description:     req.Description,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	query := `
		INSERT INTO share_transactions (
			id, chama_id, from_member_id, to_member_id, transaction_type,
			shares_count, share_value, total_amount, transaction_date, status,
			description, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.Exec(
		query,
		transaction.ID,
		transaction.ChamaID,
		transaction.FromMemberID,
		transaction.ToMemberID,
		transaction.TransactionType,
		transaction.SharesCount,
		transaction.ShareValue,
		transaction.TotalAmount,
		transaction.TransactionDate,
		transaction.Status,
		transaction.Description,
		transaction.CreatedAt,
		transaction.UpdatedAt,
	)

	return err
}

// Helper method to record share transfer wallet transaction
func (s *SharesService) recordShareTransferTransaction(tx *sql.Tx, chamaID, fromUserID, toUserID string, amount float64, notes *string) error {
	transactionID := uuid.New().String()
	now := time.Now()

	description := "Share transfer"
	if notes != nil && *notes != "" {
		description = fmt.Sprintf("Share transfer: %s", *notes)
	}

	metadata := map[string]interface{}{
		"transferType": "share_transfer",
		"fromUserId":   fromUserID,
		"toUserId":     toUserID,
		"chamaId":      chamaID,
	}
	if notes != nil {
		metadata["notes"] = *notes
	}

	metadataJSON, _ := json.Marshal(metadata)

	_, err := tx.Exec(`
		INSERT INTO transactions (
			id, type, amount, currency, description, status, payment_method,
			initiated_by, recipient_id, metadata, created_at, updated_at
		) VALUES (?, 'share_transfer', ?, 'KES', ?, 'completed', 'wallet', ?, ?, ?, ?, ?)
	`,
		transactionID, amount, description,
		fromUserID, toUserID, string(metadataJSON), now, now,
	)

	return err
}

func stringPtr(s string) *string {
	return &s
}
