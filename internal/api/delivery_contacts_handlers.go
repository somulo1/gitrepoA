package api

// import (
// 	"database/sql"
// 	"net/http"
// 	"strconv"

// 	"github.com/gin-gonic/gin"
// 	"github.com/google/uuid"
// )

// // DeliveryContactsHandlers handles delivery contacts API endpoints
// type DeliveryContactsHandlers struct {
// 	db *sql.DB
// }

// // NewDeliveryContactsHandlers creates a new instance of DeliveryContactsHandlers
// func NewDeliveryContactsHandlers(db *sql.DB) *DeliveryContactsHandlers {
// 	return &DeliveryContactsHandlers{db: db}
// }

// // GetDeliveryContacts retrieves delivery contacts for the current seller
// func (h *DeliveryContactsHandlers) GetDeliveryContacts(c *gin.Context) {
// 	userID := c.GetString("userID")

// 	if userID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get pagination parameters
// 	limitStr := c.DefaultQuery("limit", "50")
// 	offsetStr := c.DefaultQuery("offset", "0")

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil {
// 		limit = 50
// 	}

// 	offset, err := strconv.Atoi(offsetStr)
// 	if err != nil {
// 		offset = 0
// 	}

// 	// Query delivery contacts
// 	query := `
// 		SELECT dc.id, dc.seller_id, dc.user_id, dc.name, dc.phone, dc.email,
// 			   dc.address, dc.notes, dc.is_active, dc.created_at, dc.updated_at,
// 			   u.first_name, u.last_name, u.email as user_email, u.phone as user_phone
// 		FROM delivery_contacts dc
// 		LEFT JOIN users u ON dc.user_id = u.id
// 		WHERE dc.seller_id = ?
// 		ORDER BY dc.created_at DESC
// 		LIMIT ? OFFSET ?
// 	`

// 	rows, err := h.db.Query(query, userID, limit, offset)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to retrieve delivery contacts",
// 		})
// 		return
// 	}
// 	defer rows.Close()

// 	var contacts []map[string]interface{}
// 	for rows.Next() {
// 		var id, sellerId, name, phone, email, address, notes, createdAt, updatedAt string
// 		var userId, userFirstName, userLastName, userEmail, userPhone sql.NullString
// 		var isActive bool

// 		err := rows.Scan(
// 			&id, &sellerId, &userId, &name, &phone, &email,
// 			&address, &notes, &isActive, &createdAt, &updatedAt,
// 			&userFirstName, &userLastName, &userEmail, &userPhone,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		contact := map[string]interface{}{
// 			"id":        id,
// 			"sellerId":  sellerId,
// 			"name":      name,
// 			"phone":     phone,
// 			"email":     email,
// 			"address":   address,
// 			"notes":     notes,
// 			"isActive":  isActive,
// 			"createdAt": createdAt,
// 			"updatedAt": updatedAt,
// 		}

// 		if userId.Valid {
// 			contact["userId"] = userId.String
// 			if userFirstName.Valid && userLastName.Valid {
// 				contact["userFullName"] = userFirstName.String + " " + userLastName.String
// 			}
// 			if userEmail.Valid {
// 				contact["userEmail"] = userEmail.String
// 			}
// 			if userPhone.Valid {
// 				contact["userPhone"] = userPhone.String
// 			}
// 		}

// 		contacts = append(contacts, contact)
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    contacts,
// 		"count":   len(contacts),
// 	})
// }

// // CreateDeliveryContact creates a new delivery contact
// func (h *DeliveryContactsHandlers) CreateDeliveryContact(c *gin.Context) {
// 	userID := c.GetString("userID")

// 	if userID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	var req struct {
// 		UserID   string `json:"userId"`
// 		Name     string `json:"name" binding:"required"`
// 		Phone    string `json:"phone"`
// 		Email    string `json:"email"`
// 		Address  string `json:"address"`
// 		Notes    string `json:"notes"`
// 		IsActive bool   `json:"isActive"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid request data: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Check if contact already exists for this seller and user
// 	if req.UserID != "" {
// 		var existingID string
// 		checkQuery := `SELECT id FROM delivery_contacts WHERE seller_id = ? AND user_id = ?`
// 		err := h.db.QueryRow(checkQuery, userID, req.UserID).Scan(&existingID)
// 		if err == nil {
// 			c.JSON(http.StatusConflict, gin.H{
// 				"success": false,
// 				"error":   "This user is already in your delivery contacts",
// 			})
// 			return
// 		}
// 	}

// 	// Create delivery contact
// 	contactID := uuid.New().String()
// 	query := `
// 		INSERT INTO delivery_contacts (
// 			id, seller_id, user_id, name, phone, email, address, notes, is_active, created_at, updated_at
// 		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
// 	`

// 	_, err := h.db.Exec(query, contactID, userID, req.UserID, req.Name, req.Phone, req.Email, req.Address, req.Notes, req.IsActive)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to create delivery contact",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data": gin.H{
// 			"id":       contactID,
// 			"name":     req.Name,
// 			"phone":    req.Phone,
// 			"email":    req.Email,
// 			"isActive": req.IsActive,
// 		},
// 		"message": "Delivery contact created successfully",
// 	})
// }

// // UpdateDeliveryContact updates an existing delivery contact
// func (h *DeliveryContactsHandlers) UpdateDeliveryContact(c *gin.Context) {
// 	userID := c.GetString("userID")
// 	contactID := c.Param("contactId")

// 	if userID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	var req struct {
// 		Name     string `json:"name"`
// 		Phone    string `json:"phone"`
// 		Email    string `json:"email"`
// 		Address  string `json:"address"`
// 		Notes    string `json:"notes"`
// 		IsActive *bool  `json:"isActive"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Invalid request data: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Check if contact exists and belongs to the seller
// 	var existingSellerId string
// 	checkQuery := `SELECT seller_id FROM delivery_contacts WHERE id = ?`
// 	err := h.db.QueryRow(checkQuery, contactID).Scan(&existingSellerId)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Delivery contact not found",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to check contact ownership",
// 			})
// 		}
// 		return
// 	}

// 	if existingSellerId != userID {
// 		c.JSON(http.StatusForbidden, gin.H{
// 			"success": false,
// 			"error":   "You can only update your own delivery contacts",
// 		})
// 		return
// 	}

// 	// Update delivery contact
// 	query := `
// 		UPDATE delivery_contacts
// 		SET name = ?, phone = ?, email = ?, address = ?, notes = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
// 		WHERE id = ?
// 	`

// 	isActive := true
// 	if req.IsActive != nil {
// 		isActive = *req.IsActive
// 	}

// 	_, err = h.db.Exec(query, req.Name, req.Phone, req.Email, req.Address, req.Notes, isActive, contactID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to update delivery contact",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery contact updated successfully",
// 	})
// }

// // DeleteDeliveryContact deletes a delivery contact
// func (h *DeliveryContactsHandlers) DeleteDeliveryContact(c *gin.Context) {
// 	userID := c.GetString("userID")
// 	contactID := c.Param("contactId")

// 	if userID == "" {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Check if contact exists and belongs to the seller
// 	var existingSellerId string
// 	checkQuery := `SELECT seller_id FROM delivery_contacts WHERE id = ?`
// 	err := h.db.QueryRow(checkQuery, contactID).Scan(&existingSellerId)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			c.JSON(http.StatusNotFound, gin.H{
// 				"success": false,
// 				"error":   "Delivery contact not found",
// 			})
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{
// 				"success": false,
// 				"error":   "Failed to check contact ownership",
// 			})
// 		}
// 		return
// 	}

// 	if existingSellerId != userID {
// 		c.JSON(http.StatusForbidden, gin.H{
// 			"success": false,
// 			"error":   "You can only delete your own delivery contacts",
// 		})
// 		return
// 	}

// 	// Delete delivery contact
// 	query := `DELETE FROM delivery_contacts WHERE id = ?`
// 	_, err = h.db.Exec(query, contactID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to delete delivery contact",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"message": "Delivery contact deleted successfully",
// 	})
// }
