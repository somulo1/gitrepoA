package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// SharesHandlers handles shares-related HTTP requests
type SharesHandlers struct {
	sharesService *services.SharesService
}

// NewSharesHandlers creates a new shares handlers instance
func NewSharesHandlers(db *sql.DB) *SharesHandlers {
	return &SharesHandlers{
		sharesService: services.NewSharesService(db),
	}
}

// CreateShareOffering creates a new share offering for a chama
func (h *SharesHandlers) CreateShareOffering(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.CreateShareOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	offering, err := h.sharesService.CreateShareOffering(chamaID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SharesResponse{
		Success: true,
		Data:    offering,
		Message: "Share offering created successfully",
	})
}

// CreateShares creates new shares for a member (legacy method)
func (h *SharesHandlers) CreateShares(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	share, err := h.sharesService.CreateShares(chamaID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SharesResponse{
		Success: true,
		Data:    share,
		Message: "Shares created successfully",
	})
}

// GetChamaShares retrieves all share offerings for a chama
func (h *SharesHandlers) GetChamaShares(c *gin.Context) {
	log.Printf("📊 GetChamaShares called - UserID: %s, ChamaID: %s", c.GetString("userID"), c.Param("id"))

	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		log.Printf("❌ GetChamaShares: User not authenticated")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		log.Printf("❌ GetChamaShares: Chama ID is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
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

	log.Printf("📊 GetChamaShares: Fetching shares for chama %s (limit: %d, offset: %d)", chamaID, limit, offset)

	offerings, err := h.sharesService.GetChamaShares(chamaID, limit, offset)
	if err != nil {
		log.Printf("❌ GetChamaShares: Failed to retrieve share offerings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve share offerings",
		})
		return
	}

	log.Printf("✅ GetChamaShares: Successfully retrieved %d share offerings", len(offerings))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    offerings,
		"count":   len(offerings),
	})
}

// GetMemberShares retrieves shares for a specific member
func (h *SharesHandlers) GetMemberShares(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")
	memberID := c.Param("memberId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesListResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" || memberID == "" {
		c.JSON(http.StatusBadRequest, models.SharesListResponse{
			Success: false,
			Error:   "Chama ID and Member ID are required",
		})
		return
	}

	shares, err := h.sharesService.GetMemberShares(chamaID, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.SharesListResponse{
			Success: false,
			Error:   "Failed to retrieve member shares",
		})
		return
	}

	c.JSON(http.StatusOK, models.SharesListResponse{
		Success: true,
		Data:    shares,
		Count:   len(shares),
	})
}

// GetChamaSharesSummary retrieves aggregated share information for a chama
func (h *SharesHandlers) GetChamaSharesSummary(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ChamaSharesSummaryResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.ChamaSharesSummaryResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	summaries, err := h.sharesService.GetChamaSharesSummary(chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ChamaSharesSummaryResponse{
			Success: false,
			Error:   "Failed to retrieve shares summary",
		})
		return
	}

	// Calculate totals
	totalShares := 0
	totalValue := 0.0
	for _, summary := range summaries {
		totalShares += summary.TotalShares
		totalValue += summary.TotalValue
	}

	c.JSON(http.StatusOK, models.ChamaSharesSummaryResponse{
		Success:      true,
		Data:         summaries,
		TotalShares:  totalShares,
		TotalValue:   totalValue,
		TotalMembers: len(summaries),
	})
}

// UpdateShares updates existing shares
func (h *SharesHandlers) UpdateShares(c *gin.Context) {
	userID := c.GetString("userID")
	shareID := c.Param("shareId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if shareID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Share ID is required",
		})
		return
	}

	var req models.UpdateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	share, err := h.sharesService.UpdateShares(shareID, &req)
	if err != nil {
		if err.Error() == "share not found" {
			c.JSON(http.StatusNotFound, models.SharesResponse{
				Success: false,
				Error:   "Share not found",
			})
			return
		}
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SharesResponse{
		Success: true,
		Data:    share,
		Message: "Shares updated successfully",
	})
}

// GetShareTransactions retrieves share transactions for a chama
func (h *SharesHandlers) GetShareTransactions(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
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

	transactions, err := h.sharesService.GetShareTransactions(chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve share transactions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transactions,
		"count":   len(transactions),
	})
}

// BuyShares allows a member to purchase shares from an offering
func (h *SharesHandlers) BuyShares(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.BuySharesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	share, err := h.sharesService.BuyShares(chamaID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SharesResponse{
		Success: true,
		Data:    share,
		Message: "Shares purchased successfully",
	})
}

// BuyDividends allows a member to purchase dividend certificates
func (h *SharesHandlers) BuyDividends(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.BuyDividendsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// For dividends, we create a share record with dividend type
	share, err := h.sharesService.BuyDividends(chamaID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.SharesResponse{
		Success: true,
		Data:    share,
		Message: "Dividend certificates purchased successfully",
	})
}

// TransferShares allows a member to transfer shares to another member
func (h *SharesHandlers) TransferShares(c *gin.Context) {
	userID := c.GetString("userID")
	chamaID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.SharesResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	if chamaID == "" {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Chama ID is required",
		})
		return
	}

	var req models.TransferSharesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Transfer the shares
	err := h.sharesService.TransferShares(chamaID, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.SharesResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.SharesResponse{
		Success: true,
		Message: "Shares transferred successfully",
	})
}

// Helper function to convert ShareWithMemberInfo slice to Share slice
func convertToShareSlice(sharesWithInfo []models.ShareWithMemberInfo) []models.Share {
	shares := make([]models.Share, len(sharesWithInfo))
	for i, shareWithInfo := range sharesWithInfo {
		shares[i] = shareWithInfo.Share
	}
	return shares
}
