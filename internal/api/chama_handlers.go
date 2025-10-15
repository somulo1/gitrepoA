package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// Chama handlers
func GetChamas(c *gin.Context) {
	// Get query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get all public chamas
	chamas, err := chamaService.GetChamas(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chamas: " + err.Error(),
		})
		return
	}

	// Return chamas
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    chamas,
		"count":   len(chamas),
	})
}

// GetAllChamasForAdmin - Admin endpoint to get all chamas (no filters)
func GetAllChamasForAdmin(c *gin.Context) {
	// Check if user is admin
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only admins can access this endpoint",
		})
		return
	}

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get all chamas for admin
	chamas, err := chamaService.GetAllChamasForAdmin(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chamas for admin: " + err.Error(),
		})
		return
	}

	// Return chamas
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    chamas,
		"count":   len(chamas),
	})
}

func GetUserChamas(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get user's chamas
	chamas, err := chamaService.GetChamasByUser(userID.(string), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get user chamas: " + err.Error(),
		})
		return
	}

	// Return chamas
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    chamas,
		"count":   len(chamas),
	})
}

func CreateChama(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Parse request body with enhanced validation tags
	var req struct {
		Name                  string  `json:"name" binding:"required" validate:"required,min=3,max=100,safe_text,no_sql_injection,no_xss"`
		Description           string  `json:"description" binding:"required" validate:"required,min=10,max=500,safe_text,no_sql_injection,no_xss"`
		Category              string  `json:"category" binding:"required" validate:"required,oneof=chama contribution"`
		Type                  string  `json:"type" binding:"required" validate:"required,alphanumeric"`
		County                string  `json:"county" binding:"required" validate:"required,min=2,max=50,alpha,no_sql_injection,no_xss"`
		Town                  string  `json:"town" binding:"required" validate:"required,min=2,max=50,alpha,no_sql_injection,no_xss"`
		ContributionAmount    float64 `json:"contribution_amount,omitempty"`
		ContributionFrequency string  `json:"contribution_frequency,omitempty"`
		TargetAmount          float64 `json:"target_amount,omitempty"`
		TargetDeadline        string  `json:"target_deadline,omitempty"`
		PaymentMethod         string  `json:"payment_method,omitempty" validate:"omitempty,oneof=till paybill"`
		TillNumber            string  `json:"till_number,omitempty"`
		PaybillBusinessNumber string  `json:"paybill_business_number,omitempty"`
		PaybillAccountNumber  string  `json:"paybill_account_number,omitempty"`
		PaymentRecipientName  string  `json:"payment_recipient_name,omitempty"`
		MaxMembers            int     `json:"max_members" binding:"required" validate:"required,min=2,max=1000"`
		IsPublic              bool    `json:"is_public"`
		RequiresApproval      bool    `json:"requires_approval"`
		Rules                 string  `json:"rules" validate:"max=1000,safe_text,no_sql_injection,no_xss"`
		MeetingSchedule       string  `json:"meeting_schedule" validate:"max=200,safe_text,no_sql_injection,no_xss"`
		Members               []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"members,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSON binding failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Debug logging
	log.Printf("🔍 Received chama creation request: %+v", req)
	log.Printf("📋 Members count: %d", len(req.Members))

	// Additional validation - basic security checks
	if len(req.Name) < 3 || len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama name must be between 3 and 100 characters",
		})
		return
	}

	if len(req.Description) < 10 || len(req.Description) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Description must be between 10 and 500 characters",
		})
		return
	}

	// Validate category
	if req.Category != "chama" && req.Category != "contribution" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Category must be either 'chama' or 'contribution'",
		})
		return
	}

	// Validate type based on category
	chamaTypes := []string{"investment", "savings", "business", "welfare", "merry-go-round"}
	contributionTypes := []string{"emergency", "medical", "education", "community", "personal"}

	if req.Category == "chama" {
		validType := false
		for _, t := range chamaTypes {
			if req.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid type for chama. Valid types: investment, savings, business, welfare, merry-go-round",
			})
			return
		}
	} else if req.Category == "contribution" {
		validType := false
		for _, t := range contributionTypes {
			if req.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid type for contribution group. Valid types: emergency, medical, education, community, personal",
			})
			return
		}

		// For contribution groups, target amount is required
		if req.TargetAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Target amount is required for contribution groups",
			})
			return
		}
	} else if req.Category == "chama" {
		// For chamas, contribution amount and frequency are required
		if req.ContributionAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Contribution amount is required for chamas",
			})
			return
		}

		validFrequencies := []string{"weekly", "monthly", "quarterly"}
		validFreq := false
		for _, freq := range validFrequencies {
			if req.ContributionFrequency == freq {
				validFreq = true
				break
			}
		}
		if !validFreq {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid contribution frequency. Valid options: weekly, monthly, quarterly",
			})
			return
		}
	}

	// Validate payment method if provided
	if req.PaymentMethod != "" {
		if req.PaymentMethod != "till" && req.PaymentMethod != "paybill" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Payment method must be either 'till' or 'paybill'",
			})
			return
		}

		if req.PaymentMethod == "till" {
			if req.TillNumber == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Till number is required when payment method is 'till'",
				})
				return
			}
			if req.PaymentRecipientName == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Payment recipient name is required for till payments",
				})
				return
			}
		}

		if req.PaymentMethod == "paybill" {
			if req.PaybillBusinessNumber == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Paybill business number is required when payment method is 'paybill'",
				})
				return
			}
			if req.PaybillAccountNumber == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Paybill account number is required when payment method is 'paybill'",
				})
				return
			}
			if req.PaymentRecipientName == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Payment recipient name is required for paybill payments",
				})
				return
			}
		}
	}



	if req.MaxMembers < 2 || req.MaxMembers > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Max members must be between 2 and 1000",
		})
		return
	}

	// Basic sanitization - remove dangerous characters
	req.Name = sanitizeInput(req.Name)
	req.Description = sanitizeInput(req.Description)
	req.Type = sanitizeInput(req.Type)
	req.County = sanitizeInput(req.County)
	req.Town = sanitizeInput(req.Town)
	req.ContributionFrequency = sanitizeInput(req.ContributionFrequency)
	req.Rules = sanitizeInput(req.Rules)
	req.MeetingSchedule = sanitizeInput(req.MeetingSchedule)

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Convert string fields to proper types
	var description *string
	if req.Description != "" {
		description = &req.Description
	}

	var maxMembers *int
	if req.MaxMembers > 0 {
		maxMembers = &req.MaxMembers
	}

	// Parse rules from string to []string (simple split by newlines)
	var rules []string
	if req.Rules != "" {
		// For now, treat the entire string as one rule
		// In the future, you might want to parse JSON or split by delimiters
		rules = []string{req.Rules}
	}

	// Parse meeting schedule from string (for now, just store as simple schedule)
	var meetingSchedule *models.MeetingSchedule
	if req.MeetingSchedule != "" {
		meetingSchedule = &models.MeetingSchedule{
			Frequency: "monthly", // Default
			Time:      "18:00",   // Default
		}
	}

	// Handle target amount and deadline for contribution groups
	var targetAmount *float64
	var targetDeadline *time.Time

	if req.Category == "contribution" {
		if req.TargetAmount > 0 {
			targetAmount = &req.TargetAmount
		}

		if req.TargetDeadline != "" {
			if deadline, err := time.Parse("2006-01-02", req.TargetDeadline); err == nil {
				targetDeadline = &deadline
			}
		}
	}

	// Handle payment method fields
	var paymentMethod, tillNumber, paybillBusinessNumber, paybillAccountNumber, paymentRecipientName *string

	if req.PaymentMethod != "" {
		paymentMethod = &req.PaymentMethod

		if req.PaymentMethod == "till" && req.TillNumber != "" {
			tillNumber = &req.TillNumber
		}

		if req.PaymentMethod == "paybill" {
			if req.PaybillBusinessNumber != "" {
				paybillBusinessNumber = &req.PaybillBusinessNumber
			}
			if req.PaybillAccountNumber != "" {
				paybillAccountNumber = &req.PaybillAccountNumber
			}
		}

		if req.PaymentRecipientName != "" {
			paymentRecipientName = &req.PaymentRecipientName
		}
	}

	// Create chama creation model
	creation := &models.ChamaCreation{
		Name:                  req.Name,
		Description:           description,
		Category:              models.ChamaCategory(req.Category),
		Type:                  models.ChamaType(req.Type),
		County:                req.County,
		Town:                  req.Town,
		ContributionAmount:    req.ContributionAmount,
		ContributionFrequency: models.ContributionFrequency(req.ContributionFrequency),
		TargetAmount:          targetAmount,
		TargetDeadline:        targetDeadline,
		PaymentMethod:         paymentMethod,
		TillNumber:            tillNumber,
		PaybillBusinessNumber: paybillBusinessNumber,
		PaybillAccountNumber:  paybillAccountNumber,
		PaymentRecipientName:  paymentRecipientName,
		MaxMembers:            maxMembers,
		IsPublic:              req.IsPublic,
		RequiresApproval:      req.RequiresApproval,
		Rules:                 rules,
		MeetingSchedule:       meetingSchedule,
	}

	// Create the chama
	chama, err := chamaService.CreateChama(creation, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create chama: " + err.Error(),
		})
		return
	}

	// Add additional members to the chama
	if len(req.Members) > 0 {
		for _, member := range req.Members {
			// Skip the creator (already added as chairperson)
			if member.UserID == userID.(string) {
				continue
			}

			// Add member to chama
			err := chamaService.AddMemberToChama(chama.ID, member.UserID, member.Role)
			if err != nil {
				// Log error but don't fail the entire operation
				// The chama is already created, so we continue with other members
				fmt.Printf("Warning: Failed to add member %s to chama %s: %v\n", member.UserID, chama.ID, err)
				continue
			}

			// Update current members count
			chama.CurrentMembers++
		}

		// Update the chama's current members count in database
		err = chamaService.UpdateChamaMemberCount(chama.ID, chama.CurrentMembers)
		if err != nil {
			fmt.Printf("Warning: Failed to update member count for chama %s: %v\n", chama.ID, err)
		}
	}

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Chama created successfully",
		"data": map[string]interface{}{
			"id":                     chama.ID,
			"name":                   chama.Name,
			"description":            chama.Description,
			"category":               chama.Category,
			"type":                   chama.Type,
			"status":                 chama.Status,
			"county":                 chama.County,
			"town":                   chama.Town,
			"contribution_amount":    chama.ContributionAmount,
			"contribution_frequency":    chama.ContributionFrequency,
			"target_amount":             chama.TargetAmount,
			"target_deadline":           chama.TargetDeadline,
			"payment_method":            chama.PaymentMethod,
			"till_number":               chama.TillNumber,
			"paybill_business_number":   chama.PaybillBusinessNumber,
			"paybill_account_number":    chama.PaybillAccountNumber,
			"payment_recipient_name":    chama.PaymentRecipientName,
			"max_members":               chama.MaxMembers,
			"current_members":        chama.CurrentMembers,
			"is_public":              chama.IsPublic,
			"requires_approval":      chama.RequiresApproval,
			"created_by":             chama.CreatedBy,
			"created_at":             chama.CreatedAt,
		},
	})
}

func generateMockMonthlyData(average float64) []float64 {
	data := make([]float64, 12)
	for i := 0; i < 12; i++ {
		// Add some variation around the average
		variation := (float64(i%3) - 1) * 500 // -500, 0, +500 variation
		data[i] = average + variation
		if data[i] < 0 {
			data[i] = 0
		}
	}
	return data
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func GetChama(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get chama details
	chama, err := chamaService.GetChamaByID(chamaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Chama not found: " + err.Error(),
		})
		return
	}

	// Return chama details
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    chama,
		"message": "Chama details retrieved successfully",
	})
}

func UpdateChama(c *gin.Context) {
	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user is chairperson of this chama
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to verify user role: " + err.Error(),
		})
		return
	}

	if userRole != "chairperson" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson can update chama settings",
		})
		return
	}

	// Parse request body
	var req struct {
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Update chama settings
	err = chamaService.UpdateChamaSettings(chamaID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update chama settings: " + err.Error(),
		})
		return
	}

	// Get updated chama details
	updatedChama, err := chamaService.GetChamaByID(chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get updated chama details: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    updatedChama,
		"message": "Chama settings updated successfully",
	})
}

func DeleteChama(c *gin.Context) {
	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user is chairperson of this chama
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	if userRole != "chairperson" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson can delete the chama",
		})
		return
	}

	// Delete the chama (this will cascade delete all related data)
	err = chamaService.DeleteChama(chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete chama: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Chama deleted successfully",
	})
}

func LeaveChama(c *gin.Context) {
	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user is a member of this chama
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	// Check if user is the chairperson - chairperson cannot leave without transferring role
	if userRole == "chairperson" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chairperson cannot leave chama. Please transfer chairperson role first or delete the chama.",
		})
		return
	}

	// Remove user from chama
	err = chamaService.RemoveUserFromChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to leave chama: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Successfully left the chama",
	})
}

func GetChamaMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	dbInterface, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}
	db := dbInterface.(*sql.DB)

	// Get real chama members from database with comprehensive information
	query := `
		SELECT
			cm.id, cm.chama_id, cm.user_id, cm.role, cm.joined_at, cm.is_active,
			cm.total_contributions, cm.last_contribution, cm.rating, cm.total_ratings,
			u.first_name, u.last_name, u.email, u.phone, u.avatar, u.status,
			u.is_email_verified, u.is_phone_verified, u.business_type, u.county, u.town,
			u.bio, u.occupation, u.created_at as user_created_at,
			COALESCE(w.balance, 0) as savings_balance,
			COALESCE(loan_balance.balance, 0) as loan_balance,
			COALESCE(contrib_stats.monthly_average, 0) as monthly_average,
			COALESCE(contrib_stats.consistency_rate, 0) as consistency_rate,
			COALESCE(meeting_stats.meetings_attended, 0) as meetings_attended,
			COALESCE(meeting_stats.total_meetings, 0) as total_meetings,
			COALESCE(activity_stats.contributions_made, 0) as contributions_made,
			COALESCE(activity_stats.loans_taken, 0) as loans_taken,
			COALESCE(activity_stats.guarantor_requests, 0) as guarantor_requests
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN wallets w ON u.id = w.owner_id AND w.type = 'personal'
		LEFT JOIN (
			SELECT
				borrower_id,
				SUM(CASE WHEN status IN ('approved', 'disbursed', 'active') THEN remaining_amount ELSE 0 END) as balance
			FROM loans
			WHERE chama_id = ?
			GROUP BY borrower_id
		) loan_balance ON u.id = loan_balance.borrower_id
		LEFT JOIN (
			SELECT
				t.initiated_by,
				AVG(t.amount) as monthly_average,
				(COUNT(*) * 100.0 / 12) as consistency_rate,
				COUNT(*) as contributions_made
			FROM transactions t
			WHERE t.type = 'contribution'
			AND t.created_at >= datetime('now', '-12 months')
			GROUP BY t.initiated_by
		) contrib_stats ON u.id = contrib_stats.initiated_by
		LEFT JOIN (
			SELECT
				t.initiated_by,
				COUNT(*) as contributions_made,
				0 as loans_taken,
				0 as guarantor_requests
			FROM transactions t
			WHERE t.type = 'contribution'
			GROUP BY t.initiated_by
		) activity_stats ON u.id = activity_stats.initiated_by
		LEFT JOIN (
			SELECT
				cm.user_id,
				COUNT(*) as meetings_attended,
				(SELECT COUNT(*) FROM meetings WHERE chama_id = ?) as total_meetings
			FROM chama_members cm
			WHERE cm.chama_id = ?
			GROUP BY cm.user_id
		) meeting_stats ON u.id = meeting_stats.user_id
		WHERE cm.chama_id = ? AND cm.is_active = true
		ORDER BY cm.joined_at ASC
	`

	rows, err := db.Query(query, chamaID, chamaID, chamaID, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch chama members: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	activeMembers := 0
	pendingMembers := 0

	for rows.Next() {
		var (
			id, chamaID, userID, role, firstName, lastName, email, phone, userStatus                        string
			joinedAt, userCreatedAt                                                                         string
			isActive, isEmailVerified, isPhoneVerified                                                      bool
			totalContributions, rating, savingsBalance, loanBalance, monthlyAverage, consistencyRate        float64
			totalRatings, meetingsAttended, totalMeetings, contributionsMade, loansTaken, guarantorRequests int
			avatar, lastContribution, businessType, county, town, bio, occupation                           *string
		)

		err := rows.Scan(
			&id, &chamaID, &userID, &role, &joinedAt, &isActive,
			&totalContributions, &lastContribution, &rating, &totalRatings,
			&firstName, &lastName, &email, &phone, &avatar, &userStatus,
			&isEmailVerified, &isPhoneVerified, &businessType, &county, &town,
			&bio, &occupation, &userCreatedAt,
			&savingsBalance, &loanBalance, &monthlyAverage, &consistencyRate,
			&meetingsAttended, &totalMeetings, &contributionsMade, &loansTaken, &guarantorRequests,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		// Calculate attendance rate
		attendanceRate := 0.0
		if totalMeetings > 0 {
			attendanceRate = (float64(meetingsAttended) / float64(totalMeetings)) * 100
		}

		// Determine online status (mock for now - would need real-time tracking)
		isOnline := userStatus == "active" && (id == "user-1" || id == "user-3" || id == "user-4")

		// Calculate last contribution amount (mock for now)
		lastContributionAmount := 5000.0
		if totalContributions > 0 {
			lastContributionAmount = monthlyAverage
		}

		// Build member object with real data
		member := map[string]interface{}{
			"id":                       id,
			"user_id":                  userID,
			"chama_id":                 chamaID,
			"role":                     role,
			"joined_at":                joinedAt,
			"status":                   userStatus,
			"total_contributions":      totalContributions,
			"last_contribution_date":   lastContribution,
			"last_contribution_amount": lastContributionAmount,
			"attendance_rate":          attendanceRate,
			"loan_balance":             loanBalance,
			"savings_balance":          savingsBalance,
			"reputation_score":         rating,
			"business_type":            businessType,
			"location":                 fmt.Sprintf("%s, %s", getStringValue(town), getStringValue(county)),
			"phone_verified":           isPhoneVerified,
			"email_verified":           isEmailVerified,
			"user": map[string]interface{}{
				"id":         userID,
				"first_name": firstName,
				"last_name":  lastName,
				"email":      email,
				"phone":      phone,
				"avatar_url": avatar,
				"bio":        bio,
				"occupation": occupation,
				"created_at": userCreatedAt,
				"last_seen":  joinedAt, // Mock - would need real tracking
				"is_online":  isOnline,
			},
			"contributions_summary": map[string]interface{}{
				"total_amount":     totalContributions,
				"monthly_average":  monthlyAverage,
				"consistency_rate": consistencyRate,
				"last_12_months":   generateMockMonthlyData(monthlyAverage), // Mock historical data
			},
			"activity_summary": map[string]interface{}{
				"meetings_attended":  meetingsAttended,
				"total_meetings":     totalMeetings,
				"last_activity":      joinedAt,
				"contributions_made": contributionsMade,
				"loans_taken":        loansTaken,
				"guarantor_requests": guarantorRequests,
			},
		}

		members = append(members, member)

		// Count member statuses
		if userStatus == "active" {
			activeMembers++
		} else {
			pendingMembers++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
		"message": "Chama members retrieved successfully",
		"meta": map[string]interface{}{
			"total_members":   len(members),
			"active_members":  activeMembers,
			"pending_members": pendingMembers,
			"last_updated":    time.Now().Format(time.RFC3339),
		},
	})
}

// SendChamaInvitation sends an invitation to join a chama
func SendChamaInvitation(c *gin.Context) {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ [CHAMA INVITATION] Panic in SendChamaInvitation: %v\n", r)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Internal server error during invitation sending",
			})
		}
	}()

	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if the chama exists
	_, err := chamaService.GetChamaByID(chamaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Chama not found",
		})
		return
	}

	// Check if user has permission to invite (chairperson, secretary, treasurer)
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	if !isLeadershipRole(userRole) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson, secretary, and treasurer can send invitations",
		})
		return
	}

	// Parse request body
	var req struct {
		Email           string `json:"email" binding:"required,email"`
		PhoneNumber     string `json:"phone_number,omitempty"`
		Message         string `json:"message,omitempty"`
		Role            string `json:"role,omitempty"`
		RoleName        string `json:"role_name,omitempty"`
		RoleDescription string `json:"role_description,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Send invitation
	fmt.Printf("🔍 Sending chama invitation:\n")
	fmt.Printf("  - Chama ID: %s\n", chamaID)
	fmt.Printf("  - User ID: %s\n", userID.(string))
	fmt.Printf("  - Email: %s\n", req.Email)
	fmt.Printf("  - Phone: %s\n", req.PhoneNumber)
	fmt.Printf("  - Message: %s\n", req.Message)

	invitationID, err := chamaService.SendInvitation(chamaID, userID.(string), req.Email, req.PhoneNumber, req.Message, req.Role, req.RoleName, req.RoleDescription)
	if err != nil {
		fmt.Printf("❌ Chama invitation failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to send invitation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation sent successfully",
		"data": gin.H{
			"invitation_id": invitationID,
		},
	})
}

// RespondToInvitation handles accepting or rejecting a chama invitation
func RespondToInvitation(c *gin.Context) {
	// Get invitation ID from URL (support both old and new route formats)
	invitationID := c.Param("invitationId")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation ID is required",
		})
		return
	}

	// Get chama ID from URL (using :id parameter to match route pattern)
	chamaID := c.Param("id")
	if chamaID != "" {
		fmt.Printf("📋 RespondToInvitation: Chama ID provided: %s\n", chamaID)
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Parse request body
	var req struct {
		Response string `json:"response" binding:"required"` // "accept" or "reject"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	if req.Response != "accept" && req.Response != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Response must be 'accept' or 'reject'",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Respond to invitation
	err := chamaService.RespondToInvitation(invitationID, userID.(string), req.Response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to respond to invitation: " + err.Error(),
		})
		return
	}

	message := "Invitation rejected"
	if req.Response == "accept" {
		message = "Invitation accepted! You are now a member of the chama"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
	})
}

// GetUserInvitations gets pending invitations for a user
func GetUserInvitations(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get user invitations
	invitations, err := chamaService.GetUserInvitations(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get invitations: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    invitations,
		"count":   len(invitations),
	})
}

// GetChamaSentInvitations gets all invitations sent for a specific chama
func GetChamaSentInvitations(c *gin.Context) {
	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user has permission to view invitations (chairperson, secretary, treasurer)
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	if !isLeadershipRole(userRole) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson, secretary, and treasurer can view sent invitations",
		})
		return
	}

	// Get sent invitations for this chama
	invitations, err := chamaService.GetChamaSentInvitations(chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get sent invitations: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    invitations,
		"count":   len(invitations),
	})
}

// CancelInvitation cancels a pending invitation
func CancelInvitation(c *gin.Context) {
	// Get invitation ID from URL
	invitationID := c.Param("invitationId")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation ID is required",
		})
		return
	}

	// Get chama ID from URL (using :id parameter to match route pattern)
	chamaID := c.Param("id")
	if chamaID != "" {
		fmt.Printf("📋 CancelInvitation: Chama ID provided: %s\n", chamaID)
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Update invitation status to cancelled
	result, err := db.(*sql.DB).Exec(`
		UPDATE chama_invitations
		SET status = 'cancelled', responded_at = ?
		WHERE id = ? AND inviter_id = ? AND status = 'pending'
	`, time.Now(), invitationID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to cancel invitation: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Invitation not found or cannot be cancelled",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation cancelled successfully",
	})
}

// ResendInvitation resends a pending invitation
func ResendInvitation(c *gin.Context) {
	fmt.Printf("🔄 [RESEND INVITATION] ResendInvitation handler called\n")
	fmt.Printf("📝 [RESEND INVITATION] Request method: %s, URL: %s\n", c.Request.Method, c.Request.URL.Path)

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("❌ [RESEND INVITATION] Panic in ResendInvitation: %v\n", r)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Internal server error during invitation resending",
			})
		}
	}()

	// Get invitation ID from URL
	invitationID := c.Param("invitationId")
	fmt.Printf("📋 [RESEND INVITATION] Invitation ID: %s\n", invitationID)
	if invitationID == "" {
		fmt.Printf("❌ [RESEND INVITATION] Invitation ID is missing\n")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invitation ID is required",
		})
		return
	}

	// Get chama ID from URL (using :id parameter to match route pattern)
	chamaID := c.Param("id")
	if chamaID != "" {
		fmt.Printf("📋 [RESEND INVITATION] Chama ID provided: %s\n", chamaID)
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Get invitation details first
	var invitation struct {
		Email           string
		ChamaID         string
		Message         string
		InvitationToken string
	}

	err := db.(*sql.DB).QueryRow(`
		SELECT email, chama_id, message, invitation_token
		FROM chama_invitations
		WHERE id = ? AND inviter_id = ? AND status = 'pending'
	`, invitationID, userID.(string)).Scan(&invitation.Email, &invitation.ChamaID, &invitation.Message, &invitation.InvitationToken)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Invitation not found or cannot be resent",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to get invitation details: " + err.Error(),
			})
		}
		return
	}

	// Update invitation with new expiry date
	result, err := db.(*sql.DB).Exec(`
		UPDATE chama_invitations
		SET expires_at = ?
		WHERE id = ? AND inviter_id = ? AND status = 'pending'
	`, time.Now().Add(7*24*time.Hour), invitationID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to resend invitation: " + err.Error(),
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Invitation not found or cannot be resent",
		})
		return
	}

	// Create chama service and resend the email
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Get chama and inviter details for email
	var chamaName, inviterFirstName, inviterLastName string
	err = db.(*sql.DB).QueryRow(`
		SELECT c.name, u.first_name, u.last_name
		FROM chamas c
		INNER JOIN users u ON u.id = ?
		WHERE c.id = ?
	`, userID.(string), invitation.ChamaID).Scan(&chamaName, &inviterFirstName, &inviterLastName)

	if err != nil {
		fmt.Printf("❌ Failed to get chama/inviter details for resend: %v\n", err)
		// Continue anyway, email sending is not critical
	} else {
		// Resend the email
		fmt.Printf("📧 Resending chama invitation email to: %s\n", invitation.Email)
		inviterFullName := fmt.Sprintf("%s %s", inviterFirstName, inviterLastName)

		// Safely attempt to send email
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("❌ Panic during email resend: %v\n", r)
				}
			}()

			emailService := chamaService.GetEmailService()
			if emailService != nil {
				err = emailService.SendChamaInvitationEmail(
					invitation.Email,
					chamaName,
					inviterFullName,
					invitation.Message,
					invitation.InvitationToken,
				)
				if err != nil {
					fmt.Printf("❌ Failed to resend invitation email: %v\n", err)
					// Don't fail the API call if email fails
				} else {
					fmt.Printf("✅ Invitation email resent successfully to: %s\n", invitation.Email)
				}
			} else {
				fmt.Printf("❌ Email service not available for resend\n")
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Invitation resent successfully",
	})
}

// GetMemberRole gets a member's role in a chama
func GetMemberRole(c *gin.Context) {
	chamaID := c.Param("id")
	userID := c.Param("userId")

	if chamaID == "" || userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID and User ID are required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query member role
	query := `SELECT role FROM chama_members WHERE chama_id = ? AND user_id = ? AND is_active = TRUE`
	var role string
	err := db.(*sql.DB).QueryRow(query, chamaID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Member not found in this chama",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to get member role",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"role": role,
		},
	})
}

// Helper function to check if role can send invitations
func isLeadershipRole(role string) bool {
	return role == "chairperson" || role == "secretary" || role == "treasurer"
}

// InviteToChama is an alias for SendChamaInvitation for test compatibility
func InviteToChama(c *gin.Context) {
	SendChamaInvitation(c)
}

// UpdateChamaMember updates a chama member's role or status
func UpdateChamaMember(c *gin.Context) {
	// Get chama ID and member ID from URL
	chamaID := c.Param("id")
	memberID := c.Param("memberId")
	if chamaID == "" || memberID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID and Member ID are required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Parse request body
	var req struct {
		Role   string `json:"role"`
		Status string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user has permission to update members (chairperson only)
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	if userRole != "chairperson" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson can update member details",
		})
		return
	}

	// Update member role if provided
	if req.Role != "" {
		err = chamaService.UpdateMemberRoleSimple(chamaID, memberID, req.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update member role: " + err.Error(),
			})
			return
		}
	}

	// Update member status if provided
	if req.Status != "" {
		err = chamaService.UpdateMemberStatus(chamaID, memberID, req.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update member status: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member updated successfully",
	})
}

// RemoveChamaMember removes a member from a chama
func RemoveChamaMember(c *gin.Context) {
	// Get chama ID and member ID from URL
	chamaID := c.Param("id")
	memberID := c.Param("memberId")
	if chamaID == "" || memberID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID and Member ID are required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user has permission to remove members (chairperson only)
	userRole, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	if userRole != "chairperson" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Only chairperson can remove members",
		})
		return
	}

	// Remove member from chama
	err = chamaService.RemoveUserFromChama(chamaID, memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to remove member: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Member removed successfully",
	})
}

// GetChamaStatistics returns statistics for a chama
func GetChamaStatistics(c *gin.Context) {
	// Get chama ID from URL
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user is a member of this chama
	_, err := chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	// Get chama statistics with user-specific data
	stats, err := chamaService.GetChamaStatistics(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chama statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetChamaTransactions retrieves all transactions for a chama
func GetChamaTransactions(c *gin.Context) {
	// Get chama ID from URL parameter
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create chama service
	chamaService := services.NewChamaService(db.(*sql.DB))

	// Check if user is a member of this chama
	_, err = chamaService.GetUserRoleInChama(chamaID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "You are not a member of this chama",
		})
		return
	}

	// Get chama transactions
	transactions, err := chamaService.GetChamaTransactions(chamaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get chama transactions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    transactions,
		"count":   len(transactions),
	})
}

// GetEligibleLoanMembers retrieves members eligible for loan disbursements
func GetEligibleLoanMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible loan members (approved but not yet disbursed)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   l.amount as approved_amount, l.status, l.created_at
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		INNER JOIN loans l ON cm.user_id = l.borrower_id AND l.chama_id = cm.chama_id
		WHERE cm.chama_id = ? AND cm.is_active = true
		AND l.status = 'approved'
		AND NOT EXISTS (
			SELECT 1 FROM disbursements d
			WHERE d.chama_id = cm.chama_id AND d.member_id = cm.user_id
			AND d.disbursement_type = 'loan_disbursement' AND d.status IN ('completed', 'processing')
		)
		ORDER BY l.created_at ASC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible loan members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName, status string
		var approvedAmount float64
		var createdAt string

		err := rows.Scan(&userID, &firstName, &lastName, &approvedAmount, &status, &createdAt)
		if err != nil {
			continue
		}

		member := map[string]interface{}{
			"id":             userID,
			"name":           firstName + " " + lastName,
			"approvedAmount": approvedAmount,
			"status":         status,
			"applicationDate": createdAt,
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// GetEligibleWelfareMembers retrieves members eligible for welfare disbursements
func GetEligibleWelfareMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible welfare members (completed contribution period)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   COALESCE(SUM(t.amount), 0) as contribution_amount,
			   COUNT(t.id) as contribution_count,
			   MAX(t.created_at) as last_contribution
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN transactions t ON cm.user_id = t.initiated_by AND t.type = 'contribution'
		WHERE cm.chama_id = ? AND cm.is_active = true
		GROUP BY cm.user_id, u.first_name, u.last_name
		HAVING contribution_count >= 6  -- At least 6 months of contributions
		ORDER BY contribution_amount DESC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible welfare members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName string
		var contributionAmount float64
		var contributionCount int
		var lastContribution sql.NullString

		err := rows.Scan(&userID, &firstName, &lastName, &contributionAmount, &contributionCount, &lastContribution)
		if err != nil {
			continue
		}

		member := map[string]interface{}{
			"id":                  userID,
			"name":                firstName + " " + lastName,
			"contributionAmount": contributionAmount,
			"contributionCount":  contributionCount,
		}
		if lastContribution.Valid {
			member["lastContribution"] = lastContribution.String
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// GetEligibleDividendMembers retrieves members eligible for dividend disbursements
func GetEligibleDividendMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible dividend members (shareholders)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   COALESCE(SUM(s.shares_owned), 0) as shares_owned,
			   COALESCE(SUM(s.total_value), 0) as total_value
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN shares s ON cm.user_id = s.member_id AND s.chama_id = cm.chama_id AND s.status = 'active'
		WHERE cm.chama_id = ? AND cm.is_active = true
		GROUP BY cm.user_id, u.first_name, u.last_name
		HAVING shares_owned > 0
		ORDER BY shares_owned DESC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible dividend members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName string
		var sharesOwned int
		var totalValue float64

		err := rows.Scan(&userID, &firstName, &lastName, &sharesOwned, &totalValue)
		if err != nil {
			continue
		}

		member := map[string]interface{}{
			"id":           userID,
			"name":         firstName + " " + lastName,
			"sharesOwned":  sharesOwned,
			"totalValue":   totalValue,
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// GetEligibleSharesMembers retrieves members eligible for share allocations
func GetEligibleSharesMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible share members (active members with good standing)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   COALESCE(SUM(s.shares_owned), 0) as current_shares,
			   COALESCE(SUM(t.amount), 0) as total_contributions,
			   COUNT(t.id) as contribution_count
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN shares s ON cm.user_id = s.member_id AND s.chama_id = cm.chama_id AND s.status = 'active'
		LEFT JOIN transactions t ON cm.user_id = t.initiated_by AND t.type = 'contribution'
		WHERE cm.chama_id = ? AND cm.is_active = true
		GROUP BY cm.user_id, u.first_name, u.last_name
		HAVING contribution_count >= 3  -- At least 3 contributions
		ORDER BY total_contributions DESC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible share members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName string
		var currentShares int
		var totalContributions float64
		var contributionCount int

		err := rows.Scan(&userID, &firstName, &lastName, &currentShares, &totalContributions, &contributionCount)
		if err != nil {
			continue
		}

		// Calculate eligible shares based on contributions (1 share per 1000 KES contributed)
		eligibleShares := int(totalContributions / 1000)
		if eligibleShares > currentShares {
			eligibleShares = eligibleShares - currentShares
		} else {
			eligibleShares = 0
		}

		member := map[string]interface{}{
			"id":                userID,
			"name":              firstName + " " + lastName,
			"currentShares":     currentShares,
			"eligibleShares":    eligibleShares,
			"totalContributions": totalContributions,
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// GetEligibleSavingsMembers retrieves members eligible for savings withdrawals
func GetEligibleSavingsMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible savings members (members with savings balance)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   COALESCE(w.balance, 0) as available_savings,
			   COALESCE(SUM(t.amount), 0) as total_deposits
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN wallets w ON u.id = w.owner_id AND w.type = 'personal'
		LEFT JOIN transactions t ON cm.user_id = t.initiated_by AND t.type = 'savings_deposit'
		WHERE cm.chama_id = ? AND cm.is_active = true
		GROUP BY cm.user_id, u.first_name, u.last_name, w.balance
		HAVING available_savings > 0
		ORDER BY available_savings DESC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible savings members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName string
		var availableSavings, totalDeposits float64

		err := rows.Scan(&userID, &firstName, &lastName, &availableSavings, &totalDeposits)
		if err != nil {
			continue
		}

		member := map[string]interface{}{
			"id":               userID,
			"name":             firstName + " " + lastName,
			"availableSavings": availableSavings,
			"totalDeposits":    totalDeposits,
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// GetEligibleOtherMembers retrieves members eligible for other disbursements
func GetEligibleOtherMembers(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Query eligible other members (active members in good standing)
	query := `
		SELECT DISTINCT cm.user_id, u.first_name, u.last_name,
			   COALESCE(SUM(t.amount), 0) as total_contributions,
			   COUNT(t.id) as contribution_count,
			   cm.joined_at
		FROM chama_members cm
		INNER JOIN users u ON cm.user_id = u.id
		LEFT JOIN transactions t ON cm.user_id = t.initiated_by AND t.type = 'contribution'
		WHERE cm.chama_id = ? AND cm.is_active = true
		GROUP BY cm.user_id, u.first_name, u.last_name, cm.joined_at
		HAVING contribution_count >= 1  -- At least 1 contribution
		ORDER BY total_contributions DESC
	`

	rows, err := db.(*sql.DB).Query(query, chamaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch eligible other members",
		})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var userID, firstName, lastName, joinedAt string
		var totalContributions float64
		var contributionCount int

		err := rows.Scan(&userID, &firstName, &lastName, &totalContributions, &contributionCount, &joinedAt)
		if err != nil {
			continue
		}

		// Calculate eligible amount (10% of total contributions)
		eligibleAmount := totalContributions * 0.1

		member := map[string]interface{}{
			"id":                  userID,
			"name":                firstName + " " + lastName,
			"eligibleAmount":      eligibleAmount,
			"totalContributions":  totalContributions,
			"contributionCount":   contributionCount,
			"joinedAt":            joinedAt,
		}
		members = append(members, member)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    members,
	})
}

// CreateIndividualDisbursement creates an individual disbursement
func CreateIndividualDisbursement(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Parse request body
	var req struct {
		Type            string  `json:"type" binding:"required"`
		Category        string  `json:"category" binding:"required"`
		MemberID        string  `json:"memberId" binding:"required"`
		MemberName      string  `json:"memberName" binding:"required"`
		Amount          float64 `json:"amount" binding:"required"`
		Purpose         string  `json:"purpose" binding:"required"`
		PrivateNote     string  `json:"privateNote"`
		FromAccount     string  `json:"fromAccount" binding:"required"`
		ToAccount       string  `json:"toAccount" binding:"required"`
		InitiatedBy     string  `json:"initiatedBy" binding:"required"`
		InitiatedByID   string  `json:"initiatedById" binding:"required"`
		Timestamp       string  `json:"timestamp" binding:"required"`
		Status          string  `json:"status" binding:"required"`
		TransactionID   string  `json:"transactionId" binding:"required"`
		SecurityHash    string  `json:"securityHash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Insert disbursement record
	query := `
		INSERT INTO disbursements (
			id, chama_id, type, category, member_id, member_name, amount, purpose,
			private_note, from_account, to_account, initiated_by, initiated_by_id,
			timestamp, status, transaction_id, security_hash, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	disburseID := fmt.Sprintf("DISB_%d", time.Now().Unix())
	now := time.Now()

	_, err := db.(*sql.DB).Exec(
		query,
		disburseID, chamaID, req.Type, req.Category, req.MemberID, req.MemberName,
		req.Amount, req.Purpose, req.PrivateNote, req.FromAccount, req.ToAccount,
		req.InitiatedBy, req.InitiatedByID, req.Timestamp, req.Status,
		req.TransactionID, req.SecurityHash, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create disbursement: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Individual disbursement created successfully",
		"data": gin.H{
			"id": disburseID,
		},
	})
}

// CreateBulkDisbursement creates a bulk disbursement (dividends)
func CreateBulkDisbursement(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Parse request body
	var req struct {
		Type             string                   `json:"type" binding:"required"`
		Category         string                   `json:"category" binding:"required"`
		DividendPerShare float64                  `json:"dividendPerShare" binding:"required"`
		TotalAmount      float64                  `json:"totalAmount" binding:"required"`
		Description      string                   `json:"description"`
		EligibleMembers  []map[string]interface{} `json:"eligibleMembers" binding:"required"`
		FromAccount      string                   `json:"fromAccount" binding:"required"`
		InitiatedBy      string                   `json:"initiatedBy" binding:"required"`
		InitiatedByID    string                   `json:"initiatedById" binding:"required"`
		Timestamp        string                   `json:"timestamp" binding:"required"`
		Status           string                   `json:"status" binding:"required"`
		TransactionID    string                   `json:"transactionId" binding:"required"`
		SecurityHash     string                   `json:"securityHash" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Start transaction
	tx, err := db.(*sql.DB).Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// Insert bulk disbursement record
	bulkID := fmt.Sprintf("BULK_%d", time.Now().Unix())
	now := time.Now()

	bulkQuery := `
		INSERT INTO bulk_disbursements (
			id, chama_id, type, category, dividend_per_share, total_amount, description,
			from_account, initiated_by, initiated_by_id, timestamp, status,
			transaction_id, security_hash, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(
		bulkQuery,
		bulkID, chamaID, req.Type, req.Category, req.DividendPerShare, req.TotalAmount,
		req.Description, req.FromAccount, req.InitiatedBy, req.InitiatedByID,
		req.Timestamp, req.Status, req.TransactionID, req.SecurityHash, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create bulk disbursement: " + err.Error(),
		})
		return
	}

	// Insert individual dividend records
	dividendQuery := `
		INSERT INTO dividends (
			id, bulk_disbursement_id, chama_id, member_id, member_name, shares_owned,
			dividend_per_share, amount, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, member := range req.EligibleMembers {
		memberID, ok := member["id"].(string)
		if !ok {
			continue
		}
		memberName, _ := member["name"].(string)
		sharesOwned, _ := member["sharesOwned"].(float64)

		dividendID := fmt.Sprintf("DIV_%d_%s", time.Now().Unix(), memberID)
		amount := sharesOwned * req.DividendPerShare

		_, err = tx.Exec(
			dividendQuery,
			dividendID, bulkID, chamaID, memberID, memberName, int(sharesOwned),
			req.DividendPerShare, amount, "pending", now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to create dividend record: " + err.Error(),
			})
			return
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to commit transaction",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Bulk disbursement created successfully",
		"data": gin.H{
			"id": bulkID,
		},
	})
}

// CreateChamaShares creates new shares for a chama
func CreateChamaShares(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Parse request body
	var req struct {
		ShareType         string  `json:"shareType" binding:"required"`
		TotalShares       int     `json:"totalShares" binding:"required"`
		PricePerShare     float64 `json:"pricePerShare" binding:"required"`
		MinimumPurchase   int     `json:"minimumPurchase"`
		Description       string  `json:"description"`
		EligibilityCriteria string `json:"eligibilityCriteria"`
		ApprovalRequired  bool    `json:"approvalRequired"`
		TotalValue        float64 `json:"totalValue"`
		CreatedBy         string  `json:"createdBy"`
		CreatedByID       string  `json:"createdById"`
		Timestamp         string  `json:"timestamp"`
		Status            string  `json:"status"`
		TransactionID     string  `json:"transactionId"`
		SecurityHash      string  `json:"securityHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Insert share offering record
	query := `
		INSERT INTO share_offerings (
			id, chama_id, share_type, total_shares, price_per_share, minimum_purchase,
			description, eligibility_criteria, approval_required, total_value,
			created_by, created_by_id, timestamp, status, transaction_id, security_hash,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	offeringID := fmt.Sprintf("OFFER_%d", time.Now().Unix())
	now := time.Now()

	_, err := db.(*sql.DB).Exec(
		query,
		offeringID, chamaID, req.ShareType, req.TotalShares, req.PricePerShare,
		req.MinimumPurchase, req.Description, req.EligibilityCriteria, req.ApprovalRequired,
		req.TotalValue, req.CreatedBy, req.CreatedByID, req.Timestamp, req.Status,
		req.TransactionID, req.SecurityHash, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create share offering: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Share offering created successfully",
		"data": gin.H{
			"id": offeringID,
		},
	})
}

// DeclareChamaDividends declares dividends for a chama
func DeclareChamaDividends(c *gin.Context) {
	chamaID := c.Param("id")
	if chamaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Chama ID is required",
		})
		return
	}

	// Parse request body
	var req struct {
		DividendType       string  `json:"dividendType" binding:"required"`
		TotalAmount        float64 `json:"totalAmount" binding:"required"`
		DividendPerShare   float64 `json:"dividendPerShare"`
		PaymentDate        string  `json:"paymentDate" binding:"required"`
		Description        string  `json:"description"`
		EligibilityCriteria string `json:"eligibilityCriteria"`
		ApprovalRequired   bool    `json:"approvalRequired"`
		CreatedBy          string  `json:"createdBy"`
		CreatedByID        string  `json:"createdById"`
		Timestamp          string  `json:"timestamp"`
		Status             string  `json:"status"`
		TransactionID      string  `json:"transactionId"`
		SecurityHash       string  `json:"securityHash"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database connection
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Insert dividend declaration record
	query := `
		INSERT INTO dividend_declarations (
			id, chama_id, dividend_type, total_amount, dividend_per_share, payment_date,
			description, eligibility_criteria, approval_required, created_by, created_by_id,
			timestamp, status, transaction_id, security_hash, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	declarationID := fmt.Sprintf("DECL_%d", time.Now().Unix())
	now := time.Now()

	_, err := db.(*sql.DB).Exec(
		query,
		declarationID, chamaID, req.DividendType, req.TotalAmount, req.DividendPerShare,
		req.PaymentDate, req.Description, req.EligibilityCriteria, req.ApprovalRequired,
		req.CreatedBy, req.CreatedByID, req.Timestamp, req.Status, req.TransactionID,
		req.SecurityHash, now, now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to declare dividends: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Dividend declaration created successfully",
		"data": gin.H{
			"id": declarationID,
		},
	})
}
