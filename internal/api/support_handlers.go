package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// SupportRequest represents a user support request
type SupportRequest struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"userId" db:"user_id"`
	Category    string     `json:"category" db:"category"`
	Subject     string     `json:"subject" db:"subject"`
	Description string     `json:"description" db:"description"`
	Priority    string     `json:"priority" db:"priority"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty" db:"resolved_at"`
	AdminNotes  *string    `json:"adminNotes,omitempty" db:"admin_notes"`

	// User information
	UserEmail     *string `json:"userEmail,omitempty" db:"user_email"`
	UserFirstName *string `json:"userFirstName,omitempty" db:"user_first_name"`
	UserLastName  *string `json:"userLastName,omitempty" db:"user_last_name"`
}

// CreateSupportRequest creates a new support request
func CreateSupportRequest(c *gin.Context) {
	// fmt.Printf("\n🎫 ===== CREATE SUPPORT REQUEST STARTED =====\n")
	// fmt.Printf("🎫 Request Method: %s\n", c.Request.Method)
	// fmt.Printf("🎫 Request URL: %s\n", c.Request.URL.String())
	// fmt.Printf("🎫 Content-Type: %s\n", c.GetHeader("Content-Type"))

	userID := c.GetString("userID")
	if userID == "" {
		fmt.Printf("❌ CREATE SUPPORT REQUEST: User not authenticated\n")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// fmt.Printf("🎫 CREATE SUPPORT REQUEST: UserID=%s\n", userID)

	var requestData struct {
		Category    string `json:"category" binding:"required"`
		Subject     string `json:"subject" binding:"required"`
		Description string `json:"description" binding:"required"`
		Priority    string `json:"priority"`
		UserInfo    struct {
			UserID    string `json:"userId"`
			Email     string `json:"email"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"userInfo"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		fmt.Printf("❌ CREATE SUPPORT REQUEST: Invalid JSON data: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	fmt.Printf("🎫 CREATE SUPPORT REQUEST: Parsed data - Category=%s, Subject=%s, Priority=%s\n",
		requestData.Category, requestData.Subject, requestData.Priority)

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create support_requests table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS support_requests (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			description TEXT NOT NULL,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			admin_notes TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.(*sql.DB).Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create support_requests table: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize support system",
		})
		return
	}

	// Set default priority if not provided
	priority := requestData.Priority
	if priority == "" {
		priority = "medium"
	}

	// Generate a unique ID for the support request
	requestID := fmt.Sprintf("sr_%d", time.Now().UnixNano())

	// Insert support request with explicit ID
	insertQuery := `
		INSERT INTO support_requests (id, user_id, category, subject, description, priority)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.(*sql.DB).Exec(insertQuery, requestID, userID, requestData.Category, requestData.Subject, requestData.Description, priority)
	if err != nil {
		fmt.Printf("❌ Failed to create support request: %v\n", err)
		fmt.Printf("   UserID: %s, Category: %s, Subject: %s\n", userID, requestData.Category, requestData.Subject)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create support request: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Support request created successfully: ID=%s, UserID=%s, Category=%s\n", requestID, userID, requestData.Category)

	// Create notification for admins about new support request
	go func() {
		// Get all admin users
		adminQuery := `SELECT id FROM users WHERE role = 'admin'`
		rows, err := db.(*sql.DB).Query(adminQuery)
		if err != nil {
			fmt.Printf("Failed to get admin users for notification: %v\n", err)
			return
		}
		defer rows.Close()

		// Create notification for each admin
		for rows.Next() {
			var adminID string
			if err := rows.Scan(&adminID); err != nil {
				continue
			}

			notificationQuery := `
				INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
				VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			`

			notificationID := fmt.Sprintf("notif_%d_%s", time.Now().UnixNano(), adminID)
			notificationTitle := "New Support Request"
			notificationMessage := fmt.Sprintf("New %s support request: %s", requestData.Category, requestData.Subject)
			notificationData := fmt.Sprintf(`{"supportRequestId": "%s", "category": "%s", "type": "new_support_request"}`, requestID, requestData.Category)

			_, err = db.(*sql.DB).Exec(notificationQuery, notificationID, adminID, "new_support_request", notificationTitle, notificationMessage, notificationData)
			if err != nil {
				fmt.Printf("Failed to create new support request notification for admin %s: %v\n", adminID, err)
			} else {
				fmt.Printf("✅ Created new support request notification for admin %s\n", adminID)
			}
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Support request created successfully",
		"data": gin.H{
			"requestId": requestID,
			"status":    "open",
			"id":        requestID,
		},
	})
}

// GetSupportRequests retrieves support requests (for admin)
func GetSupportRequests(c *gin.Context) {
	fmt.Printf("\n📋 ===== GET SUPPORT REQUESTS STARTED =====\n")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	fmt.Printf("📋 UserID: %s, UserRole: %s\n", userID, userRole)

	if userID == "" {
		fmt.Printf("❌ GET SUPPORT REQUESTS: User not authenticated\n")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create support_requests table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS support_requests (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			description TEXT NOT NULL,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			admin_notes TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.(*sql.DB).Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create support_requests table: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize support system",
		})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	status := c.DefaultQuery("status", "")
	category := c.DefaultQuery("category", "")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	var query string
	var args []interface{}

	if userRole == "admin" {
		// Admin can see all support requests
		query = `
			SELECT 
				sr.id, sr.user_id, sr.category, sr.subject, sr.description, 
				sr.priority, sr.status, sr.created_at, sr.updated_at, 
				sr.resolved_at, sr.admin_notes,
				u.email as user_email, u.first_name as user_first_name, u.last_name as user_last_name
			FROM support_requests sr
			LEFT JOIN users u ON sr.user_id = u.id
			WHERE 1=1
		`

		if status != "" {
			query += " AND sr.status = ?"
			args = append(args, status)
		}

		if category != "" {
			query += " AND sr.category = ?"
			args = append(args, category)
		}

		query += " ORDER BY sr.created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	} else {
		// Regular users can only see their own requests
		query = `
			SELECT 
				sr.id, sr.user_id, sr.category, sr.subject, sr.description, 
				sr.priority, sr.status, sr.created_at, sr.updated_at, 
				sr.resolved_at, sr.admin_notes,
				u.email as user_email, u.first_name as user_first_name, u.last_name as user_last_name
			FROM support_requests sr
			LEFT JOIN users u ON sr.user_id = u.id
			WHERE sr.user_id = ?
		`
		args = append(args, userID)

		if status != "" {
			query += " AND sr.status = ?"
			args = append(args, status)
		}

		query += " ORDER BY sr.created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	fmt.Printf("📋 Executing query: %s\n", query)
	fmt.Printf("📋 Query args: %v\n", args)

	rows, err := db.(*sql.DB).Query(query, args...)
	if err != nil {
		fmt.Printf("❌ Failed to get support requests: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve support requests",
		})
		return
	}
	defer rows.Close()

	var requests []SupportRequest
	requestCount := 0
	for rows.Next() {
		requestCount++
		var req SupportRequest
		err := rows.Scan(
			&req.ID, &req.UserID, &req.Category, &req.Subject, &req.Description,
			&req.Priority, &req.Status, &req.CreatedAt, &req.UpdatedAt,
			&req.ResolvedAt, &req.AdminNotes,
			&req.UserEmail, &req.UserFirstName, &req.UserLastName,
		)
		if err != nil {
			fmt.Printf("Failed to scan support request: %v\n", err)
			continue
		}
		requests = append(requests, req)
	}

	fmt.Printf("📋 Found %d support requests, returning %d requests\n", requestCount, len(requests))
	fmt.Printf("📋 ===== GET SUPPORT REQUESTS COMPLETED =====\n\n")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    requests,
		"meta": gin.H{
			"total":  len(requests),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// UpdateSupportRequest updates a support request (admin only)
func UpdateSupportRequest(c *gin.Context) {
	userRole := c.GetString("userRole")
	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	requestID := c.Param("id")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Request ID is required",
		})
		return
	}

	var updateData struct {
		Status     string `json:"status"`
		AdminNotes string `json:"adminNotes"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create support_requests table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS support_requests (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			description TEXT NOT NULL,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			admin_notes TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.(*sql.DB).Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create support_requests table: %v\n", err)
	}

	// Update support request
	updateQuery := `
		UPDATE support_requests 
		SET status = ?, admin_notes = ?, updated_at = CURRENT_TIMESTAMP,
		    resolved_at = CASE WHEN ? = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE id = ?
	`

	result, err := db.(*sql.DB).Exec(updateQuery, updateData.Status, updateData.AdminNotes, updateData.Status, requestID)
	if err != nil {
		fmt.Printf("Failed to update support request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update support request",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Support request not found",
		})
		return
	}

	fmt.Printf("✅ Support request updated: ID=%s, Status=%s\n", requestID, updateData.Status)

	// Create notification for the user when admin updates their support request
	go func() {
		// Get the support request details to find the user
		var supportUserID string
		var supportSubject string
		err := db.(*sql.DB).QueryRow("SELECT user_id, subject FROM support_requests WHERE id = ?", requestID).Scan(&supportUserID, &supportSubject)
		if err != nil {
			fmt.Printf("Failed to get support request details for notification: %v\n", err)
			return
		}

		// Create notification
		notificationQuery := `
			INSERT INTO notifications (id, user_id, type, title, message, data, created_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`

		notificationID := fmt.Sprintf("notif_%d", time.Now().UnixNano())
		notificationTitle := "Support Request Updated"
		notificationMessage := fmt.Sprintf("Your support request '%s' has been updated to: %s", supportSubject, updateData.Status)
		notificationData := fmt.Sprintf(`{"supportRequestId": "%s", "status": "%s", "type": "support_update"}`, requestID, updateData.Status)

		_, err = db.(*sql.DB).Exec(notificationQuery, notificationID, supportUserID, "support_update", notificationTitle, notificationMessage, notificationData)
		if err != nil {
			fmt.Printf("Failed to create support update notification: %v\n", err)
		} else {
			fmt.Printf("✅ Created support update notification for user %s\n", supportUserID)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Support request updated successfully",
	})
}

// CreateTestSupportRequest creates a test support request for debugging
func CreateTestSupportRequest(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get database from context
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Create support_requests table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS support_requests (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT NOT NULL,
			category TEXT NOT NULL,
			subject TEXT NOT NULL,
			description TEXT NOT NULL,
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'open',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME,
			admin_notes TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`

	_, err := db.(*sql.DB).Exec(createTableQuery)
	if err != nil {
		fmt.Printf("Failed to create support_requests table: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to initialize support system",
		})
		return
	}

	// Generate a unique ID for the test support request
	requestID := fmt.Sprintf("test_sr_%d", time.Now().UnixNano())

	// Insert test support request
	insertQuery := `
		INSERT INTO support_requests (id, user_id, category, subject, description, priority)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = db.(*sql.DB).Exec(insertQuery, requestID, userID, "technical", "Test Support Request", "This is a test support request to verify the system is working correctly.", "medium")
	if err != nil {
		fmt.Printf("❌ Failed to create test support request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create test support request: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ Test support request created successfully: ID=%s, UserID=%s\n", requestID, userID)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Test support request created successfully",
		"data": gin.H{
			"requestId": requestID,
			"status":    "open",
			"id":        requestID,
		},
	})
}
