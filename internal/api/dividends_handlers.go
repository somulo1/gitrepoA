package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// DividendsHandlers handles dividend-related HTTP requests
type DividendsHandlers struct {
	dividendsService *services.DividendsService
}

// NewDividendsHandlers creates a new dividends handlers instance
func NewDividendsHandlers(db *sql.DB) *DividendsHandlers {
	return &DividendsHandlers{
		dividendsService: services.NewDividendsService(db),
	}
}

// DeclareDividend creates a new dividend declaration
func (h *DividendsHandlers) DeclareDividend(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.CreateDividendDeclarationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	declaration, err := h.dividendsService.DeclareDividend(chamaID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.DividendResponse{
		Success: true,
		Data:    declaration,
		Message: "Dividend declared successfully",
	})
}

// GetChamaDividendDeclarations retrieves dividend declarations for a chama
func (h *DividendsHandlers) GetChamaDividendDeclarations(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendDeclarationsListResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.DividendDeclarationsListResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	declarations, err := h.dividendsService.GetChamaDividendDeclarations(chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.DividendDeclarationsListResponse{
			Success: false,
			Error:   "Failed to retrieve dividend declarations",
		})
		return
	}

	// Convert to basic declarations for response
	basicDeclarations := make([]models.DividendDeclaration, len(declarations))
	for i, decl := range declarations {
		basicDeclarations[i] = decl.DividendDeclaration
	}

	c.JSON(http.StatusOK, models.DividendDeclarationsListResponse{
		Success: true,
		Data:    basicDeclarations,
		Count:   len(basicDeclarations),
	})
}

// GetDividendDeclarationDetails retrieves detailed information about a dividend declaration
func (h *DividendsHandlers) GetDividendDeclarationDetails(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	declarationID := c.Param("declarationId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" || declarationID == "" {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Chama ID and Declaration ID are required",
		})
		return
	}

	// Get declarations and find the specific one
	declarations, err := h.dividendsService.GetChamaDividendDeclarations(chamaID, 100, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.DividendResponse{
			Success: false,
			Error:   "Failed to retrieve dividend declaration",
		})
		return
	}

	var targetDeclaration *models.DividendDeclarationWithDetails
	for _, decl := range declarations {
		if decl.ID == declarationID {
			targetDeclaration = &decl
			break
		}
	}

	if targetDeclaration == nil {
		c.JSON(http.StatusNotFound, models.DividendResponse{
			Success: false,
			Error:   "Dividend declaration not found",
		})
		return
	}

	c.JSON(http.StatusOK, models.DividendResponse{
		Success: true,
		Data:    targetDeclaration,
	})
}

// ApproveDividend approves a dividend declaration
func (h *DividendsHandlers) ApproveDividend(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	declarationID := c.Param("declarationId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" || declarationID == "" {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Chama ID and Declaration ID are required",
		})
		return
	}

	declaration, err := h.dividendsService.ApproveDividend(declarationID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.DividendResponse{
		Success: true,
		Data:    declaration,
		Message: "Dividend approved successfully",
	})
}

// ProcessDividendPayments processes dividend payments for a declaration
func (h *DividendsHandlers) ProcessDividendPayments(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	declarationID := c.Param("declarationId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" || declarationID == "" {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Chama ID and Declaration ID are required",
		})
		return
	}

	var req models.ProcessDividendPaymentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	err := h.dividendsService.ProcessDividendPayments(declarationID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.DividendResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.DividendResponse{
		Success: true,
		Message: "Dividend payments processed successfully",
	})
}

// GetMemberDividendHistory retrieves dividend history for a member
func (h *DividendsHandlers) GetMemberDividendHistory(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	memberID := c.Param("memberId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" || memberID == "" {
		c.JSON(http.StatusBadRequest, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "Chama ID and Member ID are required",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	payments, err := h.dividendsService.GetMemberDividendHistory(chamaID, memberID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "Failed to retrieve dividend history",
		})
		return
	}

	// Convert to basic payments for response
	basicPayments := make([]models.DividendPayment, len(payments))
	for i, payment := range payments {
		basicPayments[i] = payment.DividendPayment
	}

	c.JSON(http.StatusOK, models.DividendPaymentsListResponse{
		Success: true,
		Data:    basicPayments,
		Count:   len(basicPayments),
	})
}

// GetMyDividendHistory retrieves dividend history for the authenticated user
func (h *DividendsHandlers) GetMyDividendHistory(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	payments, err := h.dividendsService.GetMemberDividendHistory(chamaID, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.DividendPaymentsListResponse{
			Success: false,
			Error:   "Failed to retrieve dividend history",
		})
		return
	}

	// Convert to basic payments for response
	basicPayments := make([]models.DividendPayment, len(payments))
	for i, payment := range payments {
		basicPayments[i] = payment.DividendPayment
	}

	c.JSON(http.StatusOK, models.DividendPaymentsListResponse{
		Success: true,
		Data:    basicPayments,
		Count:   len(basicPayments),
	})
}
