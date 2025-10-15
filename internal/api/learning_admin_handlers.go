package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateLearningCategory creates a new learning category (admin only)
func CreateLearningCategory(c *gin.Context) {
	userRole := c.GetString("userRole")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		SortOrder   int    `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	categoryID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO learning_categories 
		(id, name, description, icon, color, sort_order, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, true, ?, ?)
	`

	_, err := db.(*sql.DB).Exec(query, categoryID, req.Name, req.Description,
		req.Icon, req.Color, req.SortOrder, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create category",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":     true,
		"message":     "Category created successfully",
		"category_id": categoryID,
	})
}

// UpdateLearningCategory updates a learning category (admin only)
func UpdateLearningCategory(c *gin.Context) {
	userRole := c.GetString("userRole")
	categoryID := c.Param("id")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		SortOrder   int    `json:"sort_order"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.Icon != "" {
		updates = append(updates, "icon = ?")
		args = append(args, req.Icon)
	}
	if req.Color != "" {
		updates = append(updates, "color = ?")
		args = append(args, req.Color)
	}
	if req.SortOrder > 0 {
		updates = append(updates, "sort_order = ?")
		args = append(args, req.SortOrder)
	}
	if req.IsActive != nil {
		updates = append(updates, "is_active = ?")
		args = append(args, *req.IsActive)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No fields to update",
		})
		return
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, categoryID)

	query := "UPDATE learning_categories SET " +
		updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	result, err := db.(*sql.DB).Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update category",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Category not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category updated successfully",
	})
}

// CreateLearningCourse creates a new learning course (admin only)
func CreateLearningCourse(c *gin.Context) {
	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var req struct {
		Title              string   `json:"title" binding:"required"`
		Description        string   `json:"description" binding:"required"`
		CategoryID         string   `json:"category_id" binding:"required"`
		Level              string   `json:"level" binding:"required"`
		Type               string   `json:"type" binding:"required"`
		Content            string   `json:"content"`
		ThumbnailURL       string   `json:"thumbnail_url"`
		DurationMinutes    int      `json:"duration_minutes"`
		EstimatedReadTime  string   `json:"estimated_read_time"`
		Tags               []string `json:"tags"`
		Prerequisites      []string `json:"prerequisites"`
		LearningObjectives []string `json:"learning_objectives"`
		Status             string   `json:"status"`
		IsFeatured         bool     `json:"is_featured"`
		// Enhanced content fields
		VideoURL        string           `json:"video_url"`
		QuizQuestions   []QuizQuestion   `json:"quiz_questions"`
		ArticleContent  *ArticleContent  `json:"article_content"`
		CourseStructure *CourseStructure `json:"course_structure"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Failed to bind JSON for course creation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data: " + err.Error(),
		})
		return
	}

	// Debug logging for enhanced content structures
	log.Printf("📝 Creating course with type: %s", req.Type)
	if req.ArticleContent != nil {
		log.Printf("📰 Article content received with %d sections", len(req.ArticleContent.Sections))
	}
	if req.CourseStructure != nil {
		log.Printf("📚 Course structure received with %d topics", len(req.CourseStructure.Topics))
	}
	if len(req.QuizQuestions) > 0 {
		log.Printf("🧪 Quiz questions received: %d questions", len(req.QuizQuestions))
	}

	// Validate level
	if req.Level != "beginner" && req.Level != "intermediate" && req.Level != "advanced" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid level. Must be: beginner, intermediate, or advanced",
		})
		return
	}

	// Validate type
	if req.Type != "article" && req.Type != "video" && req.Type != "course" && req.Type != "quiz" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid type. Must be: article, video, course, or quiz",
		})
		return
	}

	// Validate status
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Status != "draft" && req.Status != "published" && req.Status != "archived" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid status. Must be: draft, published, or archived",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Check if category exists
	var categoryExists bool
	err := db.(*sql.DB).QueryRow("SELECT EXISTS(SELECT 1 FROM learning_categories WHERE id = ? AND is_active = true)", req.CategoryID).Scan(&categoryExists)
	if err != nil || !categoryExists {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid category ID",
		})
		return
	}

	courseID := uuid.New().String()
	now := time.Now()

	// Convert arrays and objects to JSON
	tagsJSON, _ := json.Marshal(req.Tags)
	prerequisitesJSON, _ := json.Marshal(req.Prerequisites)
	objectivesJSON, _ := json.Marshal(req.LearningObjectives)

	// Convert enhanced content to JSON
	var quizQuestionsJSON, articleContentJSON, courseStructureJSON []byte
	if req.QuizQuestions != nil && len(req.QuizQuestions) > 0 {
		quizQuestionsJSON, _ = json.Marshal(req.QuizQuestions)
	} else {
		quizQuestionsJSON = []byte("[]") // Empty array
	}
	if req.ArticleContent != nil {
		articleContentJSON, _ = json.Marshal(req.ArticleContent)
	} else {
		articleContentJSON = []byte("null") // Null value
	}
	if req.CourseStructure != nil {
		courseStructureJSON, _ = json.Marshal(req.CourseStructure)
	} else {
		courseStructureJSON = []byte("null") // Null value
	}

	// Consolidate content based on course type
	var mainContent string
	switch req.Type {
	case "video":
		if req.VideoURL != "" {
			mainContent = req.VideoURL
		} else {
			mainContent = req.Content // Fallback to provided content
		}
	case "quiz":
		if len(req.QuizQuestions) > 0 {
			contentData := map[string]interface{}{
				"type":        "quiz",
				"questions":   req.QuizQuestions,
				"description": req.Content, // Use content as quiz description
			}
			contentJSON, _ := json.Marshal(contentData)
			mainContent = string(contentJSON)
		} else {
			mainContent = req.Content
		}
	case "article":
		if req.ArticleContent != nil && len(req.ArticleContent.Sections) > 0 {
			contentData := map[string]interface{}{
				"type":           "article",
				"headline_image": req.ArticleContent.HeadlineImage,
				"sections":       req.ArticleContent.Sections,
				"description":    req.Content, // Use content as article description
			}
			contentJSON, _ := json.Marshal(contentData)
			mainContent = string(contentJSON)
		} else {
			mainContent = req.Content
		}
	case "course":
		if req.CourseStructure != nil && len(req.CourseStructure.Topics) > 0 {
			contentData := map[string]interface{}{
				"type":        "course",
				"outline":     req.CourseStructure.Outline,
				"topics":      req.CourseStructure.Topics,
				"description": req.Content, // Use content as course description
			}
			contentJSON, _ := json.Marshal(contentData)
			mainContent = string(contentJSON)
		} else {
			mainContent = req.Content
		}
	default:
		mainContent = req.Content // For any other type, use the provided content
	}

	// Check if enhanced content columns exist, if not, use basic query
	var hasEnhancedColumns bool
	checkQuery := `SELECT COUNT(*) FROM pragma_table_info('learning_courses') WHERE name IN ('video_url', 'quiz_questions', 'article_content', 'course_structure')`
	err = db.(*sql.DB).QueryRow(checkQuery).Scan(&hasEnhancedColumns)
	if err != nil {
		hasEnhancedColumns = false
	}

	var query string
	var args []interface{}

	if hasEnhancedColumns {
		// Use enhanced query with new columns
		query = `
			INSERT INTO learning_courses
			(id, title, description, category_id, level, type, content, thumbnail_url,
			 duration_minutes, estimated_read_time, tags, prerequisites, learning_objectives,
			 status, is_featured, video_url, quiz_questions, article_content, course_structure,
			 created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		args = []interface{}{
			courseID, req.Title, req.Description, req.CategoryID,
			req.Level, req.Type, mainContent, req.ThumbnailURL, req.DurationMinutes,
			req.EstimatedReadTime, string(tagsJSON), string(prerequisitesJSON),
			string(objectivesJSON), req.Status, req.IsFeatured, req.VideoURL,
			string(quizQuestionsJSON), string(articleContentJSON), string(courseStructureJSON),
			userID, now, now,
		}
	} else {
		// Use basic query without enhanced columns
		query = `
			INSERT INTO learning_courses
			(id, title, description, category_id, level, type, content, thumbnail_url,
			 duration_minutes, estimated_read_time, tags, prerequisites, learning_objectives,
			 status, is_featured, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		args = []interface{}{
			courseID, req.Title, req.Description, req.CategoryID,
			req.Level, req.Type, mainContent, req.ThumbnailURL, req.DurationMinutes,
			req.EstimatedReadTime, string(tagsJSON), string(prerequisitesJSON),
			string(objectivesJSON), req.Status, req.IsFeatured, userID, now, now,
		}
	}

	_, err = db.(*sql.DB).Exec(query, args...)
	if err != nil {
		log.Printf("Failed to create course: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to create course: %v", err),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success":   true,
		"message":   "Course created successfully",
		"course_id": courseID,
	})
}

// UpdateLearningCourse updates an existing learning course (admin only)
func UpdateLearningCourse(c *gin.Context) {
	userRole := c.GetString("userRole")
	courseID := c.Param("id")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	var req struct {
		Title              string   `json:"title"`
		Description        string   `json:"description"`
		CategoryID         string   `json:"category_id"`
		Level              string   `json:"level"`
		Type               string   `json:"type"`
		Content            string   `json:"content"`
		ThumbnailURL       string   `json:"thumbnail_url"`
		DurationMinutes    int      `json:"duration_minutes"`
		EstimatedReadTime  string   `json:"estimated_read_time"`
		Tags               []string `json:"tags"`
		Prerequisites      []string `json:"prerequisites"`
		LearningObjectives []string `json:"learning_objectives"`
		Status             string   `json:"status"`
		IsFeatured         *bool    `json:"is_featured"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Build dynamic update query
	updates := []string{}
	args := []interface{}{}

	if req.Title != "" {
		updates = append(updates, "title = ?")
		args = append(args, req.Title)
	}
	if req.Description != "" {
		updates = append(updates, "description = ?")
		args = append(args, req.Description)
	}
	if req.CategoryID != "" {
		// Verify category exists
		var categoryExists bool
		err := db.(*sql.DB).QueryRow("SELECT EXISTS(SELECT 1 FROM learning_categories WHERE id = ? AND is_active = true)", req.CategoryID).Scan(&categoryExists)
		if err != nil || !categoryExists {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid category ID",
			})
			return
		}
		updates = append(updates, "category_id = ?")
		args = append(args, req.CategoryID)
	}
	if req.Level != "" {
		if req.Level != "beginner" && req.Level != "intermediate" && req.Level != "advanced" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid level. Must be: beginner, intermediate, or advanced",
			})
			return
		}
		updates = append(updates, "level = ?")
		args = append(args, req.Level)
	}
	if req.Type != "" {
		if req.Type != "article" && req.Type != "video" && req.Type != "course" && req.Type != "quiz" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid type. Must be: article, video, course, or quiz",
			})
			return
		}
		updates = append(updates, "type = ?")
		args = append(args, req.Type)
	}
	if req.Content != "" {
		updates = append(updates, "content = ?")
		args = append(args, req.Content)
	}
	if req.ThumbnailURL != "" {
		updates = append(updates, "thumbnail_url = ?")
		args = append(args, req.ThumbnailURL)
	}
	if req.DurationMinutes > 0 {
		updates = append(updates, "duration_minutes = ?")
		args = append(args, req.DurationMinutes)
	}
	if req.EstimatedReadTime != "" {
		updates = append(updates, "estimated_read_time = ?")
		args = append(args, req.EstimatedReadTime)
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		updates = append(updates, "tags = ?")
		args = append(args, string(tagsJSON))
	}
	if req.Prerequisites != nil {
		prerequisitesJSON, _ := json.Marshal(req.Prerequisites)
		updates = append(updates, "prerequisites = ?")
		args = append(args, string(prerequisitesJSON))
	}
	if req.LearningObjectives != nil {
		objectivesJSON, _ := json.Marshal(req.LearningObjectives)
		updates = append(updates, "learning_objectives = ?")
		args = append(args, string(objectivesJSON))
	}
	if req.Status != "" {
		if req.Status != "draft" && req.Status != "published" && req.Status != "archived" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid status. Must be: draft, published, or archived",
			})
			return
		}
		updates = append(updates, "status = ?")
		args = append(args, req.Status)
	}
	if req.IsFeatured != nil {
		updates = append(updates, "is_featured = ?")
		args = append(args, *req.IsFeatured)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "No fields to update",
		})
		return
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, courseID)

	query := "UPDATE learning_courses SET " +
		updates[0]
	for i := 1; i < len(updates); i++ {
		query += ", " + updates[i]
	}
	query += " WHERE id = ?"

	result, err := db.(*sql.DB).Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update course",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Course not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Course updated successfully",
	})
}

// DeleteLearningCourse deletes a learning course (admin only)
func DeleteLearningCourse(c *gin.Context) {
	userRole := c.GetString("userRole")
	courseID := c.Param("id")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Delete the course (this will cascade delete lessons, progress, etc.)
	result, err := db.(*sql.DB).Exec("DELETE FROM learning_courses WHERE id = ?", courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete course",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Course not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Course deleted successfully",
	})
}

// DeleteLearningCategory deletes a learning category (admin only)
func DeleteLearningCategory(c *gin.Context) {
	userRole := c.GetString("userRole")
	categoryID := c.Param("id")

	if userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Admin access required",
		})
		return
	}

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Check if category has courses
	var courseCount int
	err := db.(*sql.DB).QueryRow("SELECT COUNT(*) FROM learning_courses WHERE category_id = ?", categoryID).Scan(&courseCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check category usage",
		})
		return
	}

	if courseCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Cannot delete category. It has %d courses. Please move or delete the courses first.", courseCount),
		})
		return
	}

	// Delete the category
	result, err := db.(*sql.DB).Exec("DELETE FROM learning_categories WHERE id = ?", categoryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete category",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Category not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Category deleted successfully",
	})
}
