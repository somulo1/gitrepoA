package models

import (
	"time"
)

// ShareType represents the type of shares
type ShareType string

const (
	ShareTypeOrdinary  ShareType = "ordinary"
	ShareTypePreferred ShareType = "preferred"
)

// ShareStatus represents the status of shares
type ShareStatus string

const (
	ShareStatusActive      ShareStatus = "active"
	ShareStatusTransferred ShareStatus = "transferred"
	ShareStatusRedeemed    ShareStatus = "redeemed"
)

// Share represents a member's share ownership in a chama
type Share struct {
	ID                string      `json:"id" db:"id"`
	ChamaID           string      `json:"chamaId" db:"chama_id"`
	MemberID          string      `json:"memberId" db:"member_id"`
	Name              string      `json:"name" db:"name"`
	ShareType         ShareType   `json:"shareType" db:"share_type"`
	SharesOwned       int         `json:"sharesOwned" db:"shares_owned"`
	ShareValue        float64     `json:"shareValue" db:"share_value"`
	TotalValue        float64     `json:"totalValue" db:"total_value"`
	PurchaseDate      time.Time   `json:"purchaseDate" db:"purchase_date"`
	CertificateNumber *string     `json:"certificateNumber,omitempty" db:"certificate_number"`
	Status            ShareStatus `json:"status" db:"status"`
	CreatedAt         time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time   `json:"updatedAt" db:"updated_at"`
}

// ShareWithMemberInfo represents share information with member details
type ShareWithMemberInfo struct {
	Share
	MemberName  string `json:"memberName"`
	MemberEmail string `json:"memberEmail"`
}

// ShareSummary represents aggregated share information for a member
type ShareSummary struct {
	MemberID       string  `json:"memberId"`
	MemberName     string  `json:"memberName"`
	TotalShares    int     `json:"totalShares"`
	TotalValue     float64 `json:"totalValue"`
	ShareTypes     []Share `json:"shareTypes"`
	LastPurchase   *time.Time `json:"lastPurchase,omitempty"`
}

// ShareOffering represents a share offering created by a chama
type ShareOffering struct {
	ID                  string    `json:"id" db:"id"`
	ChamaID             string    `json:"chamaId" db:"chama_id"`
	Name                string    `json:"name" db:"name"`
	ShareType           string    `json:"shareType" db:"share_type"`
	TotalShares         int       `json:"totalShares" db:"total_shares"`
	PricePerShare       float64   `json:"pricePerShare" db:"price_per_share"`
	MinimumPurchase     int       `json:"minimumPurchase" db:"minimum_purchase"`
	Description         *string   `json:"description,omitempty" db:"description"`
	EligibilityCriteria *string   `json:"eligibilityCriteria,omitempty" db:"eligibility_criteria"`
	ApprovalRequired    bool      `json:"approvalRequired" db:"approval_required"`
	TotalValue          float64   `json:"totalValue" db:"total_value"`
	CreatedBy           string    `json:"createdBy" db:"created_by"`
	CreatedByID         string    `json:"createdById" db:"created_by_id"`
	Timestamp           time.Time `json:"timestamp" db:"timestamp"`
	Status              string    `json:"status" db:"status"`
	TransactionID       string    `json:"transactionId" db:"transaction_id"`
	SecurityHash        string    `json:"securityHash" db:"security_hash"`
	CreatedAt           time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time `json:"updatedAt" db:"updated_at"`
}

// CreateShareRequest represents the request to create new shares
type CreateShareRequest struct {
	MemberID          string    `json:"memberId" binding:"required"`
	Name              string    `json:"name" binding:"required,min=3,max=100"`
	ShareType         ShareType `json:"shareType" binding:"required,oneof=ordinary preferred"`
	SharesCount       int       `json:"sharesCount" binding:"required,min=1"`
	ShareValue        float64   `json:"shareValue" binding:"required,min=0"`
	PurchaseDate      time.Time `json:"purchaseDate" binding:"required"`
	CertificateNumber *string   `json:"certificateNumber,omitempty"`
}

// BuySharesRequest represents the request to buy shares from an offering
type BuySharesRequest struct {
	OfferingID    string  `json:"offeringId" binding:"required"`
	ShareType     string  `json:"shareType" binding:"required,oneof=ordinary preferred founder"`
	Quantity      int     `json:"quantity" binding:"required,min=1"`
	PricePerShare float64 `json:"pricePerShare" binding:"required,min=0"`
	TotalAmount   float64 `json:"totalAmount" binding:"required,min=0"`
	PaymentMethod string  `json:"paymentMethod" binding:"required,oneof=wallet mpesa mobile_money bank_transfer cash"`
	Notes         *string `json:"notes,omitempty"`
	PurchaseDate  time.Time `json:"purchaseDate" binding:"required"`
}

// BuyDividendsRequest represents the request to buy dividend certificates
type BuyDividendsRequest struct {
	DeclarationID string  `json:"declarationId" binding:"required"`
	Quantity      int     `json:"quantity" binding:"required,min=1"`
	PricePerShare float64 `json:"pricePerShare" binding:"required,min=0"`
	TotalAmount   float64 `json:"totalAmount" binding:"required,min=0"`
	PaymentMethod string  `json:"paymentMethod" binding:"required,oneof=wallet mpesa mobile_money bank_transfer cash"`
	Notes         *string `json:"notes,omitempty"`
	PurchaseDate  time.Time `json:"purchaseDate" binding:"required"`
}

// TransferSharesRequest represents the request to transfer shares between members
type TransferSharesRequest struct {
	ShareID       string  `json:"shareId" binding:"required"`        // ID of the share record to transfer
	ToMemberID    string  `json:"toMemberId" binding:"required"`     // Member receiving the shares
	SharesCount   int     `json:"sharesCount" binding:"required,min=1"` // Number of shares to transfer
	TransferPrice float64 `json:"transferPrice" binding:"required,min=0"` // Price per share for the transfer
	TotalAmount   float64 `json:"totalAmount" binding:"required,min=0"`   // Total amount for the transfer
	Notes         *string `json:"notes,omitempty"`                    // Optional notes about the transfer
	TransferDate  time.Time `json:"transferDate" binding:"required"` // Date of the transfer
}

// UpdateShareRequest represents the request to update shares
type UpdateShareRequest struct {
	SharesOwned       *int        `json:"sharesOwned,omitempty" binding:"omitempty,min=0"`
	ShareValue        *float64    `json:"shareValue,omitempty" binding:"omitempty,min=0"`
	CertificateNumber *string     `json:"certificateNumber,omitempty"`
	Status            *ShareStatus `json:"status,omitempty" binding:"omitempty,oneof=active transferred redeemed"`
}

// CreateShareOfferingRequest represents the request to create a share offering
type CreateShareOfferingRequest struct {
	Name                string  `json:"name" binding:"required,min=3,max=100"`
	ShareType           string  `json:"shareType" binding:"required,oneof=ordinary preferred founder"`
	TotalShares         int     `json:"totalShares" binding:"required,min=1"`
	PricePerShare       float64 `json:"pricePerShare" binding:"required,min=0"`
	MinimumPurchase     int     `json:"minimumPurchase" binding:"required,min=1"`
	Description         *string `json:"description,omitempty"`
	EligibilityCriteria *string `json:"eligibilityCriteria,omitempty"`
	ApprovalRequired    bool    `json:"approvalRequired"`
}

// ShareTransactionType represents the type of share transaction
type ShareTransactionType string

const (
	ShareTransactionPurchase   ShareTransactionType = "purchase"
	ShareTransactionTransfer   ShareTransactionType = "transfer"
	ShareTransactionRedemption ShareTransactionType = "redemption"
	ShareTransactionSplit      ShareTransactionType = "split"
)

// ShareTransactionStatus represents the status of a share transaction
type ShareTransactionStatus string

const (
	ShareTransactionPending   ShareTransactionStatus = "pending"
	ShareTransactionCompleted ShareTransactionStatus = "completed"
	ShareTransactionCancelled ShareTransactionStatus = "cancelled"
)

// ShareTransaction represents a share transaction
type ShareTransaction struct {
	ID             string                  `json:"id" db:"id"`
	ChamaID        string                  `json:"chamaId" db:"chama_id"`
	FromMemberID   *string                 `json:"fromMemberId,omitempty" db:"from_member_id"`
	ToMemberID     *string                 `json:"toMemberId,omitempty" db:"to_member_id"`
	TransactionType ShareTransactionType   `json:"transactionType" db:"transaction_type"`
	SharesCount    int                     `json:"sharesCount" db:"shares_count"`
	ShareValue     float64                 `json:"shareValue" db:"share_value"`
	TotalAmount    float64                 `json:"totalAmount" db:"total_amount"`
	TransactionDate time.Time              `json:"transactionDate" db:"transaction_date"`
	Status         ShareTransactionStatus  `json:"status" db:"status"`
	ApprovedBy     *string                 `json:"approvedBy,omitempty" db:"approved_by"`
	Description    *string                 `json:"description,omitempty" db:"description"`
	CreatedAt      time.Time               `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time               `json:"updatedAt" db:"updated_at"`
}

// CreateShareTransactionRequest represents the request to create a share transaction
type CreateShareTransactionRequest struct {
	FromMemberID    *string              `json:"fromMemberId,omitempty"`
	ToMemberID      *string              `json:"toMemberId,omitempty"`
	TransactionType ShareTransactionType `json:"transactionType" binding:"required,oneof=purchase transfer redemption split"`
	SharesCount     int                  `json:"sharesCount" binding:"required,min=1"`
	ShareValue      float64              `json:"shareValue" binding:"required,min=0"`
	TransactionDate time.Time            `json:"transactionDate" binding:"required"`
	Description     *string              `json:"description,omitempty"`
}

// SharesResponse represents the response structure for share operations
type SharesResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// SharesListResponse represents the response for listing shares
type SharesListResponse struct {
	Success bool    `json:"success"`
	Data    []Share `json:"data"`
	Count   int     `json:"count"`
	Error   string  `json:"error,omitempty"`
}

// ChamaSharesSummaryResponse represents the response for chama shares summary
type ChamaSharesSummaryResponse struct {
	Success      bool           `json:"success"`
	Data         []ShareSummary `json:"data"`
	TotalShares  int            `json:"totalShares"`
	TotalValue   float64        `json:"totalValue"`
	TotalMembers int            `json:"totalMembers"`
	Error        string         `json:"error,omitempty"`
}

// IsValidShareType checks if the share type is valid
func IsValidShareType(shareType string) bool {
	switch ShareType(shareType) {
	case ShareTypeOrdinary, ShareTypePreferred:
		return true
	default:
		return false
	}
}

// IsValidShareStatus checks if the share status is valid
func IsValidShareStatus(status string) bool {
	switch ShareStatus(status) {
	case ShareStatusActive, ShareStatusTransferred, ShareStatusRedeemed:
		return true
	default:
		return false
	}
}

// IsValidShareTransactionType checks if the transaction type is valid
func IsValidShareTransactionType(transactionType string) bool {
	switch ShareTransactionType(transactionType) {
	case ShareTransactionPurchase, ShareTransactionTransfer, ShareTransactionRedemption, ShareTransactionSplit:
		return true
	default:
		return false
	}
}

// CalculateTotalValue calculates the total value of shares
func (s *Share) CalculateTotalValue() {
	s.TotalValue = float64(s.SharesOwned) * s.ShareValue
}

// CanTransfer checks if shares can be transferred
func (s *Share) CanTransfer() bool {
	return s.Status == ShareStatusActive && s.SharesOwned > 0
}

// CanRedeem checks if shares can be redeemed
func (s *Share) CanRedeem() bool {
	return s.Status == ShareStatusActive && s.SharesOwned > 0
}
