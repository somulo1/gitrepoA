package models

import (
	"time"
)

// DividendDeclarationStatus represents the status of a dividend declaration
type DividendDeclarationStatus string

const (
	DividendStatusDeclared  DividendDeclarationStatus = "declared"
	DividendStatusApproved  DividendDeclarationStatus = "approved"
	DividendStatusPaid      DividendDeclarationStatus = "paid"
	DividendStatusCancelled DividendDeclarationStatus = "cancelled"
)

// DividendPaymentStatus represents the status of individual dividend payments
type DividendPaymentStatus string

const (
	DividendPaymentPending DividendPaymentStatus = "pending"
	DividendPaymentPaid    DividendPaymentStatus = "paid"
	DividendPaymentFailed  DividendPaymentStatus = "failed"
)

// DividendDeclaration represents a dividend declaration for a chama
type DividendDeclaration struct {
	ID                  string                    `json:"id" db:"id"`
	ChamaID             string                    `json:"chamaId" db:"chama_id"`
	DeclarationDate     time.Time                 `json:"declarationDate" db:"declaration_date"`
	DividendPerShare    float64                   `json:"dividendPerShare" db:"dividend_per_share"`
	TotalDividendAmount float64                   `json:"totalDividendAmount" db:"total_amount"`
	PaymentDate         *time.Time                `json:"paymentDate,omitempty" db:"payment_date"`
	Status              DividendDeclarationStatus `json:"status" db:"status"`
	DeclaredBy          string                    `json:"declaredBy" db:"declared_by"`
	ApprovedBy          *string                   `json:"approvedBy,omitempty" db:"approved_by"`
	Description         *string                   `json:"description,omitempty" db:"description"`
	DividendType        string                    `json:"dividendType,omitempty" db:"dividend_type"`
	CreatedAt           time.Time                 `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time                 `json:"updatedAt" db:"updated_at"`
}

// DividendDeclarationWithDetails represents dividend declaration with additional details
type DividendDeclarationWithDetails struct {
	DividendDeclaration
	DeclaredByName    string `json:"declaredByName"`
	ApprovedByName    *string `json:"approvedByName,omitempty"`
	TotalEligibleShares int   `json:"totalEligibleShares"`
	TotalRecipients   int    `json:"totalRecipients"`
	PaidAmount        float64 `json:"paidAmount"`
	PendingAmount     float64 `json:"pendingAmount"`
}

// DividendPayment represents an individual dividend payment to a member
type DividendPayment struct {
	ID                     string                `json:"id" db:"id"`
	DividendDeclarationID  string                `json:"dividendDeclarationId" db:"dividend_declaration_id"`
	MemberID               string                `json:"memberId" db:"member_id"`
	SharesEligible         int                   `json:"sharesEligible" db:"shares_eligible"`
	DividendAmount         float64               `json:"dividendAmount" db:"dividend_amount"`
	PaymentStatus          DividendPaymentStatus `json:"paymentStatus" db:"payment_status"`
	PaymentDate            *time.Time            `json:"paymentDate,omitempty" db:"payment_date"`
	PaymentMethod          *string               `json:"paymentMethod,omitempty" db:"payment_method"`
	TransactionReference   *string               `json:"transactionReference,omitempty" db:"transaction_reference"`
	CreatedAt              time.Time             `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time             `json:"updatedAt" db:"updated_at"`
}

// DividendPaymentWithMemberInfo represents dividend payment with member details
type DividendPaymentWithMemberInfo struct {
	DividendPayment
	MemberName  string `json:"memberName"`
	MemberEmail string `json:"memberEmail"`
	MemberPhone string `json:"memberPhone"`
}

// CreateDividendDeclarationRequest represents the request to declare dividends
type CreateDividendDeclarationRequest struct {
	DividendPerShare    float64    `json:"dividendPerShare" binding:"required,min=0"`
	TotalDividendAmount float64    `json:"totalDividendAmount" binding:"required,min=0"`
	PaymentDate         *time.Time `json:"paymentDate,omitempty"`
	Description         *string    `json:"description,omitempty" binding:"omitempty,max=500"`
	DividendType        string     `json:"DividendType,omitempty" binding:"omitempty,oneof=cash share"`
}

// UpdateDividendDeclarationRequest represents the request to update dividend declaration
type UpdateDividendDeclarationRequest struct {
	DividendPerShare    *float64                   `json:"dividendPerShare,omitempty" binding:"omitempty,min=0"`
	TotalDividendAmount *float64                   `json:"totalDividendAmount,omitempty" binding:"omitempty,min=0"`
	PaymentDate         *time.Time                 `json:"paymentDate,omitempty"`
	Status              *DividendDeclarationStatus `json:"status,omitempty" binding:"omitempty,oneof=declared approved paid cancelled"`
	Description         *string                    `json:"description,omitempty" binding:"omitempty,max=500"`
}

// ProcessDividendPaymentsRequest represents the request to process dividend payments
type ProcessDividendPaymentsRequest struct {
	PaymentMethod string `json:"paymentMethod" binding:"required,oneof=bank_transfer mobile_money cash"`
	PaymentDate   *time.Time `json:"paymentDate,omitempty"`
}

// DividendSummary represents dividend summary for a member
type DividendSummary struct {
	MemberID           string  `json:"memberId"`
	MemberName         string  `json:"memberName"`
	TotalDividendsEarned float64 `json:"totalDividendsEarned"`
	TotalDividendsPaid   float64 `json:"totalDividendsPaid"`
	PendingDividends     float64 `json:"pendingDividends"`
	LastDividendDate     *time.Time `json:"lastDividendDate,omitempty"`
	DividendHistory      []DividendPayment `json:"dividendHistory"`
}

// ChamaDividendSummary represents dividend summary for a chama
type ChamaDividendSummary struct {
	ChamaID              string    `json:"chamaId"`
	TotalDividendsDeclared float64 `json:"totalDividendsDeclared"`
	TotalDividendsPaid     float64 `json:"totalDividendsPaid"`
	PendingDividends       float64 `json:"pendingDividends"`
	LastDeclarationDate    *time.Time `json:"lastDeclarationDate,omitempty"`
	ActiveDeclarations     int       `json:"activeDeclarations"`
	TotalDeclarations      int       `json:"totalDeclarations"`
}

// DividendResponse represents the response structure for dividend operations
type DividendResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// DividendDeclarationsListResponse represents the response for listing dividend declarations
type DividendDeclarationsListResponse struct {
	Success bool                  `json:"success"`
	Data    []DividendDeclaration `json:"data"`
	Count   int                   `json:"count"`
	Error   string                `json:"error,omitempty"`
}

// DividendPaymentsListResponse represents the response for listing dividend payments
type DividendPaymentsListResponse struct {
	Success bool              `json:"success"`
	Data    []DividendPayment `json:"data"`
	Count   int               `json:"count"`
	Error   string            `json:"error,omitempty"`
}

// IsValidDividendDeclarationStatus checks if the dividend declaration status is valid
func IsValidDividendDeclarationStatus(status string) bool {
	switch DividendDeclarationStatus(status) {
	case DividendStatusDeclared, DividendStatusApproved, DividendStatusPaid, DividendStatusCancelled:
		return true
	default:
		return false
	}
}

// IsValidDividendPaymentStatus checks if the dividend payment status is valid
func IsValidDividendPaymentStatus(status string) bool {
	switch DividendPaymentStatus(status) {
	case DividendPaymentPending, DividendPaymentPaid, DividendPaymentFailed:
		return true
	default:
		return false
	}
}

// CanApprove checks if dividend declaration can be approved
func (dd *DividendDeclaration) CanApprove() bool {
	return dd.Status == DividendStatusDeclared
}

// CanProcess checks if dividend declaration can be processed for payment
func (dd *DividendDeclaration) CanProcess() bool {
	return dd.Status == DividendStatusApproved
}

// CanCancel checks if dividend declaration can be cancelled
func (dd *DividendDeclaration) CanCancel() bool {
	return dd.Status == DividendStatusDeclared || dd.Status == DividendStatusApproved
}

// CalculateDividendAmount calculates dividend amount for given shares
func (dd *DividendDeclaration) CalculateDividendAmount(shares int) float64 {
	return float64(shares) * dd.DividendPerShare
}

// IsEligibleForPayment checks if a dividend payment is eligible for processing
func (dp *DividendPayment) IsEligibleForPayment() bool {
	return dp.PaymentStatus == DividendPaymentPending && dp.DividendAmount > 0
}

// MarkAsPaid marks a dividend payment as paid
func (dp *DividendPayment) MarkAsPaid(paymentMethod, transactionRef string) {
	now := time.Now()
	dp.PaymentStatus = DividendPaymentPaid
	dp.PaymentDate = &now
	dp.PaymentMethod = &paymentMethod
	dp.TransactionReference = &transactionRef
	dp.UpdatedAt = now
}
