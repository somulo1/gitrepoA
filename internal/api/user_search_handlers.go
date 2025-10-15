package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UserSearchHandlers handles user search API endpoints
type UserSearchHandlers struct {
	db *sql.DB
}

// NewUserSearchHandlers creates a new instance of UserSearchHandlers
func NewUserSearchHandlers(db *sql.DB) *UserSearchHandlers {
	return &UserSearchHandlers{db: db}
}

// SearchUsers searches for users by name, email, or phone number
func (h *UserSearchHandlers) SearchUsers(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get search parameters
	query := strings.TrimSpace(c.Query("query"))
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	excludeCurrentUser := c.DefaultQuery("excludeCurrentUser", "false")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Search query is required",
		})
		return
	}

	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Search query must be at least 2 characters",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// Build search query
	searchPattern := "%" + query + "%"

	var sqlQuery string
	var args []interface{}

	if excludeCurrentUser == "true" {
		sqlQuery = `
			SELECT id, first_name, last_name, email, phone, created_at
			FROM users
			WHERE id != ? AND (
				LOWER(first_name) LIKE LOWER(?) OR
				LOWER(last_name) LIKE LOWER(?) OR
				LOWER(email) LIKE LOWER(?) OR
				phone LIKE ?
			)
			ORDER BY
				CASE
					WHEN LOWER(first_name) LIKE LOWER(?) THEN 1
					WHEN LOWER(last_name) LIKE LOWER(?) THEN 2
					WHEN LOWER(email) LIKE LOWER(?) THEN 3
					ELSE 4
				END,
				first_name, last_name
			LIMIT ? OFFSET ?
		`
		args = []interface{}{
			userID, searchPattern, searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern,
			limit, offset,
		}
	} else {
		sqlQuery = `
			SELECT id, first_name, last_name, email, phone, created_at
			FROM users
			WHERE
				LOWER(first_name) LIKE LOWER(?) OR
				LOWER(last_name) LIKE LOWER(?) OR
				LOWER(email) LIKE LOWER(?) OR
				phone LIKE ?
			ORDER BY
				CASE
					WHEN LOWER(first_name) LIKE LOWER(?) THEN 1
					WHEN LOWER(last_name) LIKE LOWER(?) THEN 2
					WHEN LOWER(email) LIKE LOWER(?) THEN 3
					ELSE 4
				END,
				first_name, last_name
			LIMIT ? OFFSET ?
		`
		args = []interface{}{
			searchPattern, searchPattern, searchPattern, searchPattern,
			searchPattern, searchPattern, searchPattern,
			limit, offset,
		}
	}

	rows, err := h.db.Query(sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to search users",
		})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, firstName, lastName, email, createdAt string
		var phoneNumber sql.NullString

		err := rows.Scan(&id, &firstName, &lastName, &email, &phoneNumber, &createdAt)
		if err != nil {
			continue
		}

		user := map[string]interface{}{
			"id":        id,
			"firstName": firstName,
			"lastName":  lastName,
			"email":     email,
			"createdAt": createdAt,
		}

		if phoneNumber.Valid {
			user["phoneNumber"] = phoneNumber.String
		}

		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"count":   len(users),
		"query":   query,
	})
}

// GetUserProfile gets a user's public profile information
func (h *UserSearchHandlers) GetUserProfile(c *gin.Context) {
	userID := c.GetString("userID")
	targetUserID := c.Param("userId")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User ID is required",
		})
		return
	}

	// Get user profile
	query := `
		SELECT id, first_name, last_name, email, phone, created_at
		FROM users
		WHERE id = ?
	`

	var id, firstName, lastName, email, createdAt string
	var phoneNumber sql.NullString

	err := h.db.QueryRow(query, targetUserID).Scan(&id, &firstName, &lastName, &email, &phoneNumber, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "User not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to get user profile",
			})
		}
		return
	}

	user := map[string]interface{}{
		"id":        id,
		"firstName": firstName,
		"lastName":  lastName,
		"email":     email,
		"createdAt": createdAt,
	}

	if phoneNumber.Valid {
		user["phoneNumber"] = phoneNumber.String
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// SearchUsersAdvanced provides advanced search with filters
func (h *UserSearchHandlers) SearchUsersAdvanced(c *gin.Context) {
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get search parameters
	query := strings.TrimSpace(c.Query("query"))
	searchType := c.DefaultQuery("type", "all") // all, name, email, phone
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	excludeCurrentUser := c.DefaultQuery("excludeCurrentUser", "false")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Search query is required",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// Build search conditions based on type
	var whereConditions []string
	var args []interface{}
	searchPattern := "%" + query + "%"

	switch searchType {
	case "name":
		whereConditions = append(whereConditions, "(LOWER(first_name) LIKE LOWER(?) OR LOWER(last_name) LIKE LOWER(?))")
		args = append(args, searchPattern, searchPattern)
	case "email":
		whereConditions = append(whereConditions, "LOWER(email) LIKE LOWER(?)")
		args = append(args, searchPattern)
	case "phone":
		whereConditions = append(whereConditions, "phone LIKE ?")
		args = append(args, searchPattern)
	default: // "all"
		whereConditions = append(whereConditions, "(LOWER(first_name) LIKE LOWER(?) OR LOWER(last_name) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?) OR phone LIKE ?)")
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if excludeCurrentUser == "true" {
		whereConditions = append(whereConditions, "id != ?")
		args = append(args, userID)
	}

	sqlQuery := `
		SELECT id, first_name, last_name, email, phone, created_at
		FROM users
		WHERE ` + strings.Join(whereConditions, " AND ") + `
		ORDER BY first_name, last_name
		LIMIT ? OFFSET ?
	`

	args = append(args, limit, offset)

	rows, err := h.db.Query(sqlQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to search users",
		})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, firstName, lastName, email, createdAt string
		var phoneNumber sql.NullString

		err := rows.Scan(&id, &firstName, &lastName, &email, &phoneNumber, &createdAt)
		if err != nil {
			continue
		}

		user := map[string]interface{}{
			"id":        id,
			"firstName": firstName,
			"lastName":  lastName,
			"email":     email,
			"createdAt": createdAt,
		}

		if phoneNumber.Valid {
			user["phoneNumber"] = phoneNumber.String
		}

		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
		"count":   len(users),
		"query":   query,
		"type":    searchType,
	})
}

// CheckMarketplaceRoles checks what marketplace roles a user has
func (h *UserSearchHandlers) CheckMarketplaceRoles(c *gin.Context) {
	userID := c.Param("userId")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "User ID is required",
		})
		return
	}

	// Query marketplace roles
	query := `
		SELECT role, auto_detected, is_active, created_at
		FROM marketplace_roles
		WHERE user_id = ? AND is_active = TRUE
	`

	rows, err := h.db.Query(query, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check marketplace roles",
		})
		return
	}
	defer rows.Close()

	roles := map[string]bool{
		"buyer":           false,
		"seller":          false,
		"delivery_person": false,
	}

	roleDetails := make(map[string]interface{})

	for rows.Next() {
		var role, createdAt string
		var autoDetected, isActive bool

		err := rows.Scan(&role, &autoDetected, &isActive, &createdAt)
		if err != nil {
			continue
		}

		roles[role] = true
		roleDetails[role] = map[string]interface{}{
			"autoDetected": autoDetected,
			"isActive":     isActive,
			"createdAt":    createdAt,
		}
	}

	// Auto-detect seller role based on products
	if !roles["seller"] {
		var productCount int
		productQuery := `SELECT COUNT(*) FROM products WHERE seller_id = ?`
		err := h.db.QueryRow(productQuery, userID).Scan(&productCount)
		if err == nil && productCount > 0 {
			roles["seller"] = true
			roleDetails["seller"] = map[string]interface{}{
				"autoDetected": true,
				"isActive":     true,
				"productCount": productCount,
				"createdAt":    nil,
			}
		}
	}

	// Auto-detect buyer role for sellers (sellers can also be buyers)
	if roles["seller"] && !roles["buyer"] {
		// Sellers are automatically buyers too (they can buy from other sellers)
		roles["buyer"] = true
		roleDetails["buyer"] = map[string]interface{}{
			"autoDetected": true,
			"isActive":     true,
			"reason":       "seller_auto_buyer",
			"createdAt":    nil,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    roles,
		"details": roleDetails,
	})
}
