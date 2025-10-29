package models

import (
	"time"
)

// LoanStatus represents loan status
type LoanStatus string

const (
	LoanStatusPending   LoanStatus = "pending"
	LoanStatusApproved  LoanStatus = "approved"
	LoanStatusRejected  LoanStatus = "rejected"
	LoanStatusActive    LoanStatus = "active"
	LoanStatusCompleted LoanStatus = "completed"
	LoanStatusDefaulted LoanStatus = "defaulted"
)

// LoanType represents loan type
type LoanType string

const (
	LoanTypePersonal   LoanType = "personal"
	LoanTypeBusiness   LoanType = "business"
	LoanTypeEmergency  LoanType = "emergency"
	LoanTypeEducation  LoanType = "education"
)

// GuarantorStatus represents guarantor status
type GuarantorStatus string

const (
	GuarantorStatusPending  GuarantorStatus = "pending"
	GuarantorStatusAccepted GuarantorStatus = "accepted"
	GuarantorStatusRejected GuarantorStatus = "rejected"
)

// Loan represents a loan in the system
type Loan struct {
	ID                string      `json:"id" db:"id"`
	BorrowerID        string      `json:"borrowerId" db:"borrower_id"`
	ChamaID           string      `json:"chamaId" db:"chama_id"`
	Type              LoanType    `json:"type" db:"type"`
	Amount            float64     `json:"amount" db:"amount"`
	InterestRate      float64     `json:"interestRate" db:"interest_rate"`
	Duration          int         `json:"duration" db:"duration"` // in months
	Purpose           string      `json:"purpose" db:"purpose"`
	Status            LoanStatus  `json:"status" db:"status"`
	ApprovedBy        *string     `json:"approvedBy,omitempty" db:"approved_by"`
	ApprovedAt        *time.Time  `json:"approvedAt,omitempty" db:"approved_at"`
	DisbursedAt       *time.Time  `json:"disbursedAt,omitempty" db:"disbursed_at"`
	DueDate           *time.Time  `json:"dueDate,omitempty" db:"due_date"`
	TotalAmount       float64     `json:"totalAmount" db:"total_amount"`
	PaidAmount        float64     `json:"paidAmount" db:"paid_amount"`
	RemainingAmount   float64     `json:"remainingAmount" db:"remaining_amount"`
	RequiredGuarantors int        `json:"requiredGuarantors" db:"required_guarantors"`
	ApprovedGuarantors int        `json:"approvedGuarantors" db:"approved_guarantors"`
	CreatedAt         time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time   `json:"updatedAt" db:"updated_at"`
	
	// Joined data
	Borrower   *User        `json:"borrower,omitempty"`
	Chama      *Chama       `json:"chama,omitempty"`
	Guarantors []Guarantor  `json:"guarantors,omitempty"`
	Payments   []LoanPayment `json:"payments,omitempty"`
}

// Guarantor represents a loan guarantor
type Guarantor struct {
	ID         string          `json:"id" db:"id"`
	LoanID     string          `json:"loanId" db:"loan_id"`
	UserID     string          `json:"userId" db:"user_id"`
	Amount     float64         `json:"amount" db:"amount"`
	Status     GuarantorStatus `json:"status" db:"status"`
	Message    *string         `json:"message,omitempty" db:"message"`
	RespondedAt *time.Time     `json:"respondedAt,omitempty" db:"responded_at"`
	CreatedAt  time.Time       `json:"createdAt" db:"created_at"`
	
	// Joined data
	User *User `json:"user,omitempty"`
}

// LoanPayment represents a loan payment
type LoanPayment struct {
	ID            string    `json:"id" db:"id"`
	LoanID        string    `json:"loanId" db:"loan_id"`
	Amount        float64   `json:"amount" db:"amount"`
	PrincipalAmount float64 `json:"principalAmount" db:"principal_amount"`
	InterestAmount  float64 `json:"interestAmount" db:"interest_amount"`
	PaymentMethod string    `json:"paymentMethod" db:"payment_method"`
	Reference     *string   `json:"reference,omitempty" db:"reference"`
	PaidAt        time.Time `json:"paidAt" db:"paid_at"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}

// LoanApplication represents loan application data
type LoanApplication struct {
	Type              LoanType `json:"type" validate:"required"`
	Amount            float64  `json:"amount" validate:"required,gt=0"`
	Duration          int      `json:"duration" validate:"required,gt=0"`
	Purpose           string   `json:"purpose" validate:"required"`
	RequiredGuarantors int     `json:"requiredGuarantors" validate:"required,gt=0"`
	GuarantorUserIDs  []string `json:"guarantorUserIds" validate:"required,min=1"`
}

// LoanApproval represents loan approval data
type LoanApproval struct {
	Approved     bool    `json:"approved"`
	InterestRate float64 `json:"interestRate,omitempty"`
	Message      *string `json:"message,omitempty"`
}

// GuarantorResponse represents guarantor response data
type GuarantorResponse struct {
	Accept  bool    `json:"accept"`
	Message *string `json:"message,omitempty"`
}

// IsActive checks if the loan is active
func (l *Loan) IsActive() bool {
	return l.Status == LoanStatusActive
}

// IsCompleted checks if the loan is completed
func (l *Loan) IsCompleted() bool {
	return l.Status == LoanStatusCompleted
}

// IsPending checks if the loan is pending approval
func (l *Loan) IsPending() bool {
	return l.Status == LoanStatusPending
}

// IsApproved checks if the loan is approved
func (l *Loan) IsApproved() bool {
	return l.Status == LoanStatusApproved
}

// IsOverdue checks if the loan is overdue
func (l *Loan) IsOverdue() bool {
	return l.IsActive() && l.DueDate != nil && time.Now().After(*l.DueDate)
}

// GetProgress returns the loan payment progress as percentage
func (l *Loan) GetProgress() float64 {
	if l.TotalAmount == 0 {
		return 0
	}
	return (l.PaidAmount / l.TotalAmount) * 100
}

// GetMonthlyPayment calculates the monthly payment amount
func (l *Loan) GetMonthlyPayment() float64 {
	if l.Duration == 0 {
		return l.TotalAmount
	}
	return l.TotalAmount / float64(l.Duration)
}

// CalculateTotalAmount calculates total amount including interest
func (l *Loan) CalculateTotalAmount() float64 {
	interest := l.Amount * (l.InterestRate / 100) * (float64(l.Duration) / 12)
	return l.Amount + interest
}

// HasSufficientGuarantors checks if loan has enough approved guarantors
func (l *Loan) HasSufficientGuarantors() bool {
	return l.ApprovedGuarantors >= l.RequiredGuarantors
}

// CanBeApproved checks if loan can be approved
func (l *Loan) CanBeApproved() bool {
	return l.IsPending() && l.HasSufficientGuarantors()
}

// GetRemainingDays returns remaining days until due date
func (l *Loan) GetRemainingDays() int {
	if l.DueDate == nil {
		return 0
	}
	
	remaining := time.Until(*l.DueDate)
	return int(remaining.Hours() / 24)
}

// IsAccepted checks if guarantor has accepted
func (g *Guarantor) IsAccepted() bool {
	return g.Status == GuarantorStatusAccepted
}

// IsRejected checks if guarantor has rejected
func (g *Guarantor) IsRejected() bool {
	return g.Status == GuarantorStatusRejected
}

// IsPending checks if guarantor response is pending
func (g *Guarantor) IsPending() bool {
	return g.Status == GuarantorStatusPending
}

// HasResponded checks if guarantor has responded
func (g *Guarantor) HasResponded() bool {
	return g.Status != GuarantorStatusPending
}

// GetTotalAmount returns total payment amount
func (lp *LoanPayment) GetTotalAmount() float64 {
	return lp.PrincipalAmount + lp.InterestAmount
}

// LoanSummary represents loan summary statistics
type LoanSummary struct {
	TotalLoans       int     `json:"totalLoans"`
	ActiveLoans      int     `json:"activeLoans"`
	CompletedLoans   int     `json:"completedLoans"`
	DefaultedLoans   int     `json:"defaultedLoans"`
	TotalBorrowed    float64 `json:"totalBorrowed"`
	TotalRepaid      float64 `json:"totalRepaid"`
	TotalOutstanding float64 `json:"totalOutstanding"`
	AverageInterestRate float64 `json:"averageInterestRate"`
}

// LoanSchedule represents loan repayment schedule
type LoanSchedule struct {
	PaymentNumber   int       `json:"paymentNumber"`
	DueDate         time.Time `json:"dueDate"`
	PrincipalAmount float64   `json:"principalAmount"`
	InterestAmount  float64   `json:"interestAmount"`
	TotalAmount     float64   `json:"totalAmount"`
	RemainingBalance float64  `json:"remainingBalance"`
	IsPaid          bool      `json:"isPaid"`
}

// GenerateSchedule generates loan repayment schedule
func (l *Loan) GenerateSchedule() []LoanSchedule {
	if l.Duration == 0 || l.DisbursedAt == nil {
		return []LoanSchedule{}
	}

	var schedule []LoanSchedule
	monthlyPayment := l.GetMonthlyPayment()
	monthlyPrincipal := l.Amount / float64(l.Duration)
	monthlyInterest := (l.TotalAmount - l.Amount) / float64(l.Duration)
	remainingBalance := l.Amount

	for i := 1; i <= l.Duration; i++ {
		dueDate := l.DisbursedAt.AddDate(0, i, 0)
		
		// Adjust last payment to account for rounding
		if i == l.Duration {
			monthlyPrincipal = remainingBalance
			monthlyPayment = monthlyPrincipal + monthlyInterest
		}

		schedule = append(schedule, LoanSchedule{
			PaymentNumber:    i,
			DueDate:          dueDate,
			PrincipalAmount:  monthlyPrincipal,
			InterestAmount:   monthlyInterest,
			TotalAmount:      monthlyPayment,
			RemainingBalance: remainingBalance - monthlyPrincipal,
			IsPaid:          false, // This would be determined by checking actual payments
		})

		remainingBalance -= monthlyPrincipal
	}

	return schedule
}
