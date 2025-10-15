package services

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/utils"
)

// WalletService handles wallet-related business logic
type WalletService struct {
	db *sql.DB
}

// NewWalletService creates a new wallet service
func NewWalletService(db *sql.DB) *WalletService {
	return &WalletService{db: db}
}

// CreateWallet creates a new wallet
func (s *WalletService) CreateWallet(ownerID string, walletType models.WalletType) (*models.Wallet, error) {
	return s.CreateWalletWithTx(nil, ownerID, walletType)
}

// CreateWalletWithTx creates a new wallet within an existing transaction
func (s *WalletService) CreateWalletWithTx(tx *sql.Tx, ownerID string, walletType models.WalletType) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ID:        uuid.New().String(),
		Type:      walletType,
		OwnerID:   ownerID,
		Balance:   0,
		Currency:  "KES",
		IsActive:  true,
		IsLocked:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO wallets (id, type, owner_id, balance, currency, is_active, is_locked, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var err error
	if tx != nil {
		// Use the provided transaction
		_, err = tx.Exec(query,
			wallet.ID, wallet.Type, wallet.OwnerID, wallet.Balance, wallet.Currency,
			wallet.IsActive, wallet.IsLocked, wallet.CreatedAt, wallet.UpdatedAt,
		)
	} else {
		// Use the database directly
		_, err = s.db.Exec(query,
			wallet.ID, wallet.Type, wallet.OwnerID, wallet.Balance, wallet.Currency,
			wallet.IsActive, wallet.IsLocked, wallet.CreatedAt, wallet.UpdatedAt,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return wallet, nil
}

// GetWalletByID retrieves a wallet by ID
func (s *WalletService) GetWalletByID(walletID string) (*models.Wallet, error) {
	query := `
		SELECT id, type, owner_id, balance, currency, is_active, is_locked,
			   daily_limit, monthly_limit, created_at, updated_at
		FROM wallets WHERE id = ?
	`

	wallet := &models.Wallet{}
	err := s.db.QueryRow(query, walletID).Scan(
		&wallet.ID, &wallet.Type, &wallet.OwnerID, &wallet.Balance, &wallet.Currency,
		&wallet.IsActive, &wallet.IsLocked, &wallet.DailyLimit, &wallet.MonthlyLimit,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

// GetWalletsByOwner retrieves all wallets for an owner
func (s *WalletService) GetWalletsByOwner(ownerID string) ([]*models.Wallet, error) {
	query := `
		SELECT id, type, owner_id, balance, currency, is_active, is_locked,
			   daily_limit, monthly_limit, created_at, updated_at
		FROM wallets WHERE owner_id = ? ORDER BY created_at ASC
	`

	rows, err := s.db.Query(query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallets: %w", err)
	}
	defer rows.Close()

	var wallets []*models.Wallet
	for rows.Next() {
		wallet := &models.Wallet{}
		err := rows.Scan(
			&wallet.ID, &wallet.Type, &wallet.OwnerID, &wallet.Balance, &wallet.Currency,
			&wallet.IsActive, &wallet.IsLocked, &wallet.DailyLimit, &wallet.MonthlyLimit,
			&wallet.CreatedAt, &wallet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wallet: %w", err)
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

// GetWalletByOwnerAndType retrieves a wallet by owner and type
func (s *WalletService) GetWalletByOwnerAndType(ownerID string, walletType models.WalletType) (*models.Wallet, error) {
	query := `
		SELECT id, type, owner_id, balance, currency, is_active, is_locked,
			   daily_limit, monthly_limit, created_at, updated_at
		FROM wallets WHERE owner_id = ? AND type = ?
	`

	wallet := &models.Wallet{}
	err := s.db.QueryRow(query, ownerID, walletType).Scan(
		&wallet.ID, &wallet.Type, &wallet.OwnerID, &wallet.Balance, &wallet.Currency,
		&wallet.IsActive, &wallet.IsLocked, &wallet.DailyLimit, &wallet.MonthlyLimit,
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet not found")
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return wallet, nil
}

// UpdateWalletBalance updates wallet balance
func (s *WalletService) UpdateWalletBalance(walletID string, newBalance float64) error {
	query := "UPDATE wallets SET balance = ?, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, newBalance, time.Now(), walletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}
	return nil
}

// LockWallet locks a wallet
func (s *WalletService) LockWallet(walletID string) error {
	query := "UPDATE wallets SET is_locked = true, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, time.Now(), walletID)
	if err != nil {
		return fmt.Errorf("failed to lock wallet: %w", err)
	}
	return nil
}

// UnlockWallet unlocks a wallet
func (s *WalletService) UnlockWallet(walletID string) error {
	query := "UPDATE wallets SET is_locked = false, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, time.Now(), walletID)
	if err != nil {
		return fmt.Errorf("failed to unlock wallet: %w", err)
	}
	return nil
}

// CreateTransaction creates a new transaction
func (s *WalletService) CreateTransaction(transaction *models.TransactionCreation, initiatedBy string) (*models.Transaction, error) {
	// Create transaction
	tx := &models.Transaction{
		ID:               uuid.New().String(),
		FromWalletID:     transaction.FromWalletID,
		ToWalletID:       transaction.ToWalletID,
		Type:             transaction.Type,
		Status:           models.TransactionStatusPending,
		Amount:           transaction.Amount,
		Currency:         "KES",
		Description:      transaction.Description,
		PaymentMethod:    transaction.PaymentMethod,
		Metadata:         transaction.Metadata,
		Fees:             0, // Calculate fees based on transaction type
		InitiatedBy:      initiatedBy,
		RequiresApproval: false, // Set based on business rules
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Calculate fees
	tx.Fees = s.calculateTransactionFees(tx)

	// Check if transaction requires approval
	tx.RequiresApproval = s.requiresApproval(tx)

	// Get metadata JSON
	metadataJSON, err := tx.GetMetadataJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize metadata: %w", err)
	}

	query := `
		INSERT INTO transactions (
			id, from_wallet_id, to_wallet_id, type, status, amount, currency,
			description, reference, payment_method, metadata, fees, initiated_by,
			requires_approval, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.Exec(query,
		tx.ID, tx.FromWalletID, tx.ToWalletID, tx.Type, tx.Status, tx.Amount,
		tx.Currency, tx.Description, tx.Reference, tx.PaymentMethod, metadataJSON,
		tx.Fees, tx.InitiatedBy, tx.RequiresApproval, tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

// ProcessTransaction processes a transaction (updates wallet balances)
func (s *WalletService) ProcessTransaction(transactionID string) error {
	// Get transaction
	transaction, err := s.GetTransactionByID(transactionID)
	if err != nil {
		return err
	}

	// Check if transaction can be processed
	if transaction.Status != models.TransactionStatusPending {
		return fmt.Errorf("transaction is not in pending status")
	}

	// Start database transaction
	dbTx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Process based on transaction type
	switch transaction.Type {
	case models.TransactionTypeDeposit:
		err = s.processDeposit(dbTx, transaction)
	case models.TransactionTypeWithdrawal:
		err = s.processWithdrawal(dbTx, transaction)
	case models.TransactionTypeTransfer:
		err = s.processTransfer(dbTx, transaction)
	default:
		err = fmt.Errorf("unsupported transaction type: %s", transaction.Type)
	}

	if err != nil {
		return err
	}

	// Update transaction status using centralized function
	err = s.updateTransactionStatus(dbTx, transactionID, models.TransactionStatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Commit transaction
	if err = dbTx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTransactionByID retrieves a transaction by ID
func (s *WalletService) GetTransactionByID(transactionID string) (*models.Transaction, error) {
	query := `
		SELECT id, from_wallet_id, to_wallet_id, type, status, amount, currency,
			   description, reference, payment_method, metadata, fees, initiated_by,
			   approved_by, requires_approval, approval_deadline, created_at, updated_at
		FROM transactions WHERE id = ?
	`

	transaction := &models.Transaction{}
	var metadataJSON sql.NullString
	err := s.db.QueryRow(query, transactionID).Scan(
		&transaction.ID, &transaction.FromWalletID, &transaction.ToWalletID,
		&transaction.Type, &transaction.Status, &transaction.Amount, &transaction.Currency,
		&transaction.Description, &transaction.Reference, &transaction.PaymentMethod,
		&metadataJSON, &transaction.Fees, &transaction.InitiatedBy, &transaction.ApprovedBy,
		&transaction.RequiresApproval, &transaction.ApprovalDeadline, &transaction.CreatedAt,
		&transaction.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	// Parse metadata
	metadataStr := "{}"
	if metadataJSON.Valid {
		metadataStr = metadataJSON.String
	}
	if err = transaction.SetMetadataFromJSON(metadataStr); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return transaction, nil
}

// GetWalletTransactions retrieves transactions for a wallet
func (s *WalletService) GetWalletTransactions(walletID string, limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, from_wallet_id, to_wallet_id, type, status, amount, currency,
			   description, reference, payment_method, metadata, fees, initiated_by,
			   approved_by, requires_approval, approval_deadline, created_at, updated_at
		FROM transactions
		WHERE from_wallet_id = ? OR to_wallet_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, walletID, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		transaction := &models.Transaction{}
		var metadataJSON sql.NullString
		err := rows.Scan(
			&transaction.ID, &transaction.FromWalletID, &transaction.ToWalletID,
			&transaction.Type, &transaction.Status, &transaction.Amount, &transaction.Currency,
			&transaction.Description, &transaction.Reference, &transaction.PaymentMethod,
			&metadataJSON, &transaction.Fees, &transaction.InitiatedBy, &transaction.ApprovedBy,
			&transaction.RequiresApproval, &transaction.ApprovalDeadline, &transaction.CreatedAt,
			&transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Parse metadata
		metadataStr := "{}"
		if metadataJSON.Valid {
			metadataStr = metadataJSON.String
		}
		if err = transaction.SetMetadataFromJSON(metadataStr); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// updateTransactionStatus updates transaction status within a database transaction
func (s *WalletService) updateTransactionStatus(tx *sql.Tx, transactionID string, status models.TransactionStatus) error {
	updateQuery := "UPDATE transactions SET status = ?, updated_at = ? WHERE id = ?"
	result, err := tx.Exec(updateQuery, status, utils.NowEAT(), transactionID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found: %s", transactionID)
	}

	log.Printf("Successfully updated transaction %s status to %s", transactionID, status)
	return nil
}

// Helper methods

func (s *WalletService) calculateTransactionFees(transaction *models.Transaction) float64 {
	// Simple fee calculation - can be made more sophisticated
	switch transaction.Type {
	case models.TransactionTypeWithdrawal:
		if transaction.PaymentMethod == models.PaymentMethodMpesa {
			return 10.0 // Fixed M-Pesa withdrawal fee
		}
		return 5.0 // Other withdrawal fees
	case models.TransactionTypeTransfer:
		return 2.0 // Transfer fee
	default:
		return 0.0 // No fees for deposits and other types
	}
}

func (s *WalletService) requiresApproval(transaction *models.Transaction) bool {
	// Business rules for approval requirements
	if transaction.Amount > 50000 { // Large amounts require approval
		return true
	}
	if transaction.Type == models.TransactionTypeLoan {
		return true
	}
	return false
}

func (s *WalletService) processDeposit(tx *sql.Tx, transaction *models.Transaction) error {
	if transaction.ToWalletID == nil {
		return errors.New("to_wallet_id is required for deposits")
	}

	// Get current balance
	var currentBalance float64
	err := tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", *transaction.ToWalletID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Update balance
	newBalance := currentBalance + transaction.Amount
	_, err = tx.Exec("UPDATE wallets SET balance = ?, updated_at = ? WHERE id = ?",
		newBalance, time.Now(), *transaction.ToWalletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	return nil
}

func (s *WalletService) processWithdrawal(tx *sql.Tx, transaction *models.Transaction) error {
	if transaction.FromWalletID == nil {
		return errors.New("from_wallet_id is required for withdrawals")
	}

	// Get current balance
	var currentBalance float64
	err := tx.QueryRow("SELECT balance FROM wallets WHERE id = ?", *transaction.FromWalletID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get wallet balance: %w", err)
	}

	// Check sufficient balance (including fees)
	totalAmount := transaction.Amount + transaction.Fees
	if currentBalance < totalAmount {
		return errors.New("insufficient balance")
	}

	// Update balance
	newBalance := currentBalance - totalAmount
	_, err = tx.Exec("UPDATE wallets SET balance = ?, updated_at = ? WHERE id = ?",
		newBalance, time.Now(), *transaction.FromWalletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance: %w", err)
	}

	return nil
}

func (s *WalletService) processTransfer(tx *sql.Tx, transaction *models.Transaction) error {
	if transaction.FromWalletID == nil || transaction.ToWalletID == nil {
		return errors.New("both from_wallet_id and to_wallet_id are required for transfers")
	}

	// Process withdrawal from source wallet
	err := s.processWithdrawal(tx, transaction)
	if err != nil {
		return err
	}

	// Process deposit to destination wallet
	depositTransaction := *transaction
	depositTransaction.Type = models.TransactionTypeDeposit
	depositTransaction.Fees = 0 // Fees already deducted from source
	err = s.processDeposit(tx, &depositTransaction)
	if err != nil {
		return err
	}

	return nil
}
