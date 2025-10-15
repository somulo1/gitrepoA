package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"vaultke-backend/internal/models"
	"vaultke-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// ReminderHandlers handles reminder-related HTTP requests
type ReminderHandlers struct {
	reminderService *services.ReminderService
}

// NewReminderHandlers creates a new reminder handlers instance
func NewReminderHandlers(db *sql.DB) *ReminderHandlers {
	return &ReminderHandlers{
		reminderService: services.NewReminderService(db),
	}
}

// CreateReminder creates a new reminder for the authenticated user
func (h *ReminderHandlers) CreateReminder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ReminderResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	var req models.CreateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	reminder, err := h.reminderService.CreateReminder(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, models.ReminderResponse{
		Success: true,
		Data:    reminder,
		Message: "Reminder created successfully",
	})
}

// GetUserReminders retrieves all reminders for the authenticated user
func (h *ReminderHandlers) GetUserReminders(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.RemindersListResponse{
			Success: false,
			Error:   "User not authenticated",
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

	reminders, err := h.reminderService.GetUserReminders(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.RemindersListResponse{
			Success: false,
			Error:   "Failed to retrieve reminders",
		})
		return
	}

	c.JSON(http.StatusOK, models.RemindersListResponse{
		Success: true,
		Data:    reminders,
		Count:   len(reminders),
	})
}

// GetReminder retrieves a specific reminder by ID
func (h *ReminderHandlers) GetReminder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ReminderResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	reminderID := c.Param("id")
	if reminderID == "" {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Reminder ID is required",
		})
		return
	}

	reminder, err := h.reminderService.GetReminderByID(reminderID, userID)
	if err != nil {
		if err.Error() == "reminder not found" {
			c.JSON(http.StatusNotFound, models.ReminderResponse{
				Success: false,
				Error:   "Reminder not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ReminderResponse{
			Success: false,
			Error:   "Failed to retrieve reminder",
		})
		return
	}

	c.JSON(http.StatusOK, models.ReminderResponse{
		Success: true,
		Data:    reminder,
	})
}

// UpdateReminder updates an existing reminder
func (h *ReminderHandlers) UpdateReminder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ReminderResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	reminderID := c.Param("id")
	if reminderID == "" {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Reminder ID is required",
		})
		return
	}

	var req models.UpdateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Invalid request data: " + err.Error(),
		})
		return
	}

	reminder, err := h.reminderService.UpdateReminder(reminderID, userID, &req)
	if err != nil {
		if err.Error() == "reminder not found" {
			c.JSON(http.StatusNotFound, models.ReminderResponse{
				Success: false,
				Error:   "Reminder not found",
			})
			return
		}
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ReminderResponse{
		Success: true,
		Data:    reminder,
		Message: "Reminder updated successfully",
	})
}

// DeleteReminder deletes a reminder
func (h *ReminderHandlers) DeleteReminder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ReminderResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	reminderID := c.Param("id")
	if reminderID == "" {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Reminder ID is required",
		})
		return
	}

	err := h.reminderService.DeleteReminder(reminderID, userID)
	if err != nil {
		if err.Error() == "reminder not found" {
			c.JSON(http.StatusNotFound, models.ReminderResponse{
				Success: false,
				Error:   "Reminder not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ReminderResponse{
			Success: false,
			Error:   "Failed to delete reminder",
		})
		return
	}

	c.JSON(http.StatusOK, models.ReminderResponse{
		Success: true,
		Message: "Reminder deleted successfully",
	})
}

// ToggleReminder toggles the enabled status of a reminder
func (h *ReminderHandlers) ToggleReminder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, models.ReminderResponse{
			Success: false,
			Error:   "User not authenticated",
		})
		return
	}

	reminderID := c.Param("id")
	if reminderID == "" {
		c.JSON(http.StatusBadRequest, models.ReminderResponse{
			Success: false,
			Error:   "Reminder ID is required",
		})
		return
	}

	// Get current reminder
	reminder, err := h.reminderService.GetReminderByID(reminderID, userID)
	if err != nil {
		if err.Error() == "reminder not found" {
			c.JSON(http.StatusNotFound, models.ReminderResponse{
				Success: false,
				Error:   "Reminder not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ReminderResponse{
			Success: false,
			Error:   "Failed to retrieve reminder",
		})
		return
	}

	// Toggle the enabled status
	newEnabled := !reminder.IsEnabled
	updateReq := models.UpdateReminderRequest{
		IsEnabled: &newEnabled,
	}

	updatedReminder, err := h.reminderService.UpdateReminder(reminderID, userID, &updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ReminderResponse{
			Success: false,
			Error:   "Failed to toggle reminder",
		})
		return
	}

	c.JSON(http.StatusOK, models.ReminderResponse{
		Success: true,
		Data:    updatedReminder,
		Message: "Reminder toggled successfully",
	})
}
