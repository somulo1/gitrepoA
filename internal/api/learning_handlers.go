package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Learning category structure
type LearningCategory struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Learning course structure
type LearningCourse struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	Description        string              `json:"description"`
	CategoryID         string              `json:"category_id"`
	CategoryName       string              `json:"category_name,omitempty"`
	Level              string              `json:"level"`
	Type               string              `json:"type"`
	Content            string              `json:"content"`
	ThumbnailURL       string              `json:"thumbnail_url"`
	DurationMinutes    int                 `json:"duration_minutes"`
	EstimatedReadTime  string              `json:"estimated_read_time"`
	Tags               []string            `json:"tags"`
	Prerequisites      []string            `json:"prerequisites"`
	LearningObjectives []string            `json:"learning_objectives"`
	Status             string              `json:"status"`
	IsFeatured         bool                `json:"is_featured"`
	ViewCount          int                 `json:"view_count"`
	Rating             float64             `json:"rating"`
	TotalRatings       int                 `json:"total_ratings"`
	CreatedBy          string              `json:"created_by"`
	CreatedByName      string              `json:"created_by_name,omitempty"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
	UserProgress       *UserCourseProgress `json:"user_progress,omitempty"`
	// Enhanced content fields
	VideoURL        string           `json:"video_url,omitempty"`
	QuizQuestions   []QuizQuestion   `json:"quiz_questions,omitempty"`
	ArticleContent  *ArticleContent  `json:"article_content,omitempty"`
	CourseStructure *CourseStructure `json:"course_structure,omitempty"`
}

// Enhanced content type structures
type QuizQuestion struct {
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"correct_answer"`
	Explanation   string   `json:"explanation,omitempty"`
}

// Enhanced ArticleContent structure matching frontend
type ArticleContent struct {
	HeadlineImage   string           `json:"headline_image"`
	Author          string           `json:"author"`
	PublicationDate string           `json:"publication_date"`
	ReadingTime     int              `json:"reading_time"`
	Excerpt         string           `json:"excerpt"`
	Tags            []string         `json:"tags"`
	Hashtags        []string         `json:"hashtags"`
	SEOKeywords     []string         `json:"seo_keywords"`
	References      []string         `json:"references"`
	TableOfContents bool             `json:"table_of_contents"`
	Sections        []ArticleSection `json:"sections"`
	Conclusion      string           `json:"conclusion"`
	CallToAction    string           `json:"call_to_action"`
	RelatedArticles []string         `json:"related_articles"`
	SocialSharing   bool             `json:"social_sharing"`
	CommentsEnabled bool             `json:"comments_enabled"`
}

// Enhanced ArticleSection structure
type ArticleSection struct {
	HeadingType  string `json:"headingType"` // h1, h2, h3, h4 - matches frontend
	Heading      string `json:"heading"`
	Content      string `json:"content"`
	ImageURL     string `json:"image_url,omitempty"`
	ImageCaption string `json:"image_caption,omitempty"`
	Quote        string `json:"quote,omitempty"`
	Order        int    `json:"order"`
}

// Enhanced CourseStructure matching frontend
type CourseStructure struct {
	Outline               string             `json:"outline"`
	Topics                []CourseTopic      `json:"topics"`
	TotalDuration         int                `json:"total_duration"`
	DifficultyProgression string             `json:"difficulty_progression"`
	Prerequisites         []string           `json:"prerequisites"`
	CompletionCriteria    CompletionCriteria `json:"completion_criteria"`
}

// Enhanced CourseTopic structure
type CourseTopic struct {
	Title              string           `json:"title"`
	Description        string           `json:"description"`
	LearningObjectives []string         `json:"learning_objectives"`
	EstimatedDuration  int              `json:"estimated_duration"`
	Subtopics          []CourseSubtopic `json:"subtopics"`
}

// Enhanced CourseSubtopic structure
type CourseSubtopic struct {
	Title              string     `json:"title"`
	Content            string     `json:"content"`
	ContentType        string     `json:"content_type"` // text, video, image, mixed
	EstimatedDuration  int        `json:"estimated_duration"`
	LearningObjectives []string   `json:"learning_objectives"`
	Resources          []string   `json:"resources"`
	Examples           []string   `json:"examples"`
	Exercises          []string   `json:"exercises"`
	VideoURL           string     `json:"video_url,omitempty"`
	ImageURL           string     `json:"image_url,omitempty"`
	Documents          []Document `json:"documents,omitempty"`
}

// Document structure for file attachments
type Document struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

// CompletionCriteria structure
type CompletionCriteria struct {
	MinTopicsCompleted  int  `json:"min_topics_completed"`
	MinScoreRequired    int  `json:"min_score_required"`
	RequireAllExercises bool `json:"require_all_exercises"`
}

// User course progress structure
type UserCourseProgress struct {
	ID                 string     `json:"id"`
	UserID             string     `json:"user_id"`
	CourseID           string     `json:"course_id"`
	Status             string     `json:"status"`
	ProgressPercentage float64    `json:"progress_percentage"`
	CurrentLessonID    *string    `json:"current_lesson_id"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	LastAccessedAt     time.Time  `json:"last_accessed_at"`
	TimeSpentMinutes   int        `json:"time_spent_minutes"`
}

// GetLearningCategories returns all learning categories
func GetLearningCategories(c *gin.Context) {
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	query := `
		SELECT id, name, description, icon, color, sort_order, is_active, created_at, updated_at
		FROM learning_categories 
		WHERE is_active = true
		ORDER BY sort_order ASC, name ASC
	`

	rows, err := db.(*sql.DB).Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch categories",
		})
		return
	}
	defer rows.Close()

	var categories []LearningCategory
	for rows.Next() {
		var category LearningCategory
		var description, icon, color sql.NullString

		err := rows.Scan(
			&category.ID, &category.Name, &description, &icon, &color,
			&category.SortOrder, &category.IsActive, &category.CreatedAt, &category.UpdatedAt,
		)
		if err != nil {
			continue
		}

		category.Description = description.String
		category.Icon = icon.String
		category.Color = color.String
		categories = append(categories, category)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"categories": categories,
	})
}

// GetLearningCategory returns a single learning category by ID
func GetLearningCategory(c *gin.Context) {
	categoryID := c.Param("id")
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	query := `
		SELECT id, name, description, icon, color, sort_order, is_active, created_at, updated_at
		FROM learning_categories
		WHERE id = ?
	`

	var category LearningCategory
	var description, icon, color sql.NullString

	err := db.(*sql.DB).QueryRow(query, categoryID).Scan(
		&category.ID, &category.Name, &description, &icon, &color,
		&category.SortOrder, &category.IsActive, &category.CreatedAt, &category.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Category not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch category",
		})
		return
	}

	category.Description = description.String
	category.Icon = icon.String
	category.Color = color.String

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    category,
	})
}

// GetLearningCourses returns courses with optional filtering
func GetLearningCourses(c *gin.Context) {
	userID := c.GetString("userID")
	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	// Parse query parameters
	categoryID := c.Query("category_id")
	level := c.Query("level")
	courseType := c.Query("type")
	featured := c.Query("featured")
	search := c.Query("search")
	limit := c.DefaultQuery("limit", "20")
	offset := c.DefaultQuery("offset", "0")

	limitInt, _ := strconv.Atoi(limit)
	offsetInt, _ := strconv.Atoi(offset)

	// Build query
	query := `
		SELECT
			lc.id, lc.title, lc.description, lc.category_id, lc.level, lc.type,
			lc.content, lc.thumbnail_url, lc.duration_minutes, lc.estimated_read_time,
			lc.tags, lc.prerequisites, lc.learning_objectives, lc.status,
			lc.is_featured, lc.view_count, lc.rating, lc.total_ratings,
			lc.created_by, lc.created_at, lc.updated_at,
			lc.video_url, lc.quiz_questions, lc.article_content, lc.course_structure,
			cat.name as category_name,
			u.first_name || ' ' || u.last_name as created_by_name
		FROM learning_courses lc
		LEFT JOIN learning_categories cat ON lc.category_id = cat.id
		LEFT JOIN users u ON lc.created_by = u.id
		WHERE lc.status = 'published'
	`

	args := []interface{}{}
	argIndex := 1

	if categoryID != "" {
		query += fmt.Sprintf(" AND lc.category_id = $%d", argIndex)
		args = append(args, categoryID)
		argIndex++
	}

	if level != "" {
		query += fmt.Sprintf(" AND lc.level = $%d", argIndex)
		args = append(args, level)
		argIndex++
	}

	if courseType != "" {
		query += fmt.Sprintf(" AND lc.type = $%d", argIndex)
		args = append(args, courseType)
		argIndex++
	}

	if featured == "true" {
		query += " AND lc.is_featured = true"
	}

	if search != "" {
		query += fmt.Sprintf(" AND (lc.title ILIKE $%d OR lc.description ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	query += " ORDER BY lc.is_featured DESC, lc.created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limitInt, offsetInt)

	rows, err := db.(*sql.DB).Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to fetch courses",
		})
		return
	}
	defer rows.Close()

	var courses []LearningCourse
	for rows.Next() {
		var course LearningCourse
		var content, thumbnailURL, estimatedReadTime sql.NullString
		var tags, prerequisites, objectives sql.NullString
		var videoURL, quizQuestions, articleContent, courseStructure sql.NullString
		var categoryName, createdByName sql.NullString

		err := rows.Scan(
			&course.ID, &course.Title, &course.Description, &course.CategoryID,
			&course.Level, &course.Type, &content, &thumbnailURL,
			&course.DurationMinutes, &estimatedReadTime, &tags, &prerequisites,
			&objectives, &course.Status, &course.IsFeatured, &course.ViewCount,
			&course.Rating, &course.TotalRatings, &course.CreatedBy,
			&course.CreatedAt, &course.UpdatedAt, &videoURL, &quizQuestions,
			&articleContent, &courseStructure, &categoryName, &createdByName,
		)
		if err != nil {
			continue
		}

		course.Content = content.String
		course.ThumbnailURL = thumbnailURL.String
		course.EstimatedReadTime = estimatedReadTime.String
		course.VideoURL = videoURL.String
		course.CategoryName = categoryName.String
		course.CreatedByName = createdByName.String

		// Parse JSON fields
		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &course.Tags)
		}
		if prerequisites.Valid {
			json.Unmarshal([]byte(prerequisites.String), &course.Prerequisites)
		}
		if objectives.Valid {
			json.Unmarshal([]byte(objectives.String), &course.LearningObjectives)
		}

		// Parse enhanced content fields
		if quizQuestions.Valid && quizQuestions.String != "" {
			json.Unmarshal([]byte(quizQuestions.String), &course.QuizQuestions)
		}
		if articleContent.Valid && articleContent.String != "" {
			var content ArticleContent
			if json.Unmarshal([]byte(articleContent.String), &content) == nil {
				course.ArticleContent = &content
			}
		}
		if courseStructure.Valid && courseStructure.String != "" {
			var structure CourseStructure
			if json.Unmarshal([]byte(courseStructure.String), &structure) == nil {
				course.CourseStructure = &structure
			}
		}

		// Parse main content field if it contains structured data
		if course.Content != "" {
			var contentData map[string]interface{}
			if json.Unmarshal([]byte(course.Content), &contentData) == nil {
				// Check if this is structured content
				if contentType, exists := contentData["type"]; exists {
					switch contentType {
					case "quiz":
						if questions, ok := contentData["questions"].([]interface{}); ok {
							questionsJSON, _ := json.Marshal(questions)
							json.Unmarshal(questionsJSON, &course.QuizQuestions)
						}
					case "article":
						if sections, ok := contentData["sections"].([]interface{}); ok {
							if headlineImage, ok := contentData["headline_image"].(string); ok {
								course.ArticleContent = &ArticleContent{
									HeadlineImage: headlineImage,
								}
								sectionsJSON, _ := json.Marshal(sections)
								json.Unmarshal(sectionsJSON, &course.ArticleContent.Sections)
							}
						}
					case "course":
						if topics, ok := contentData["topics"].([]interface{}); ok {
							if outline, ok := contentData["outline"].(string); ok {
								course.CourseStructure = &CourseStructure{
									Outline: outline,
								}
								topicsJSON, _ := json.Marshal(topics)
								json.Unmarshal(topicsJSON, &course.CourseStructure.Topics)
							}
						}
					}
				}
			}
			// For video type, if content is just a URL, set it as video_url
			if course.Type == "video" && course.VideoURL == "" && isValidURL(course.Content) {
				course.VideoURL = course.Content
			}
		}

		// Get user progress if user is authenticated
		if userID != "" {
			course.UserProgress = getUserCourseProgress(db.(*sql.DB), userID, course.ID)
		}

		courses = append(courses, course)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"courses": courses,
		"total":   len(courses),
	})
}

// Helper function to get user course progress
func getUserCourseProgress(db *sql.DB, userID, courseID string) *UserCourseProgress {
	query := `
		SELECT id, user_id, course_id, status, progress_percentage, current_lesson_id,
			   started_at, completed_at, last_accessed_at, time_spent_minutes
		FROM user_course_progress 
		WHERE user_id = ? AND course_id = ?
	`

	var progress UserCourseProgress
	var currentLessonID sql.NullString
	var startedAt, completedAt sql.NullTime

	err := db.QueryRow(query, userID, courseID).Scan(
		&progress.ID, &progress.UserID, &progress.CourseID, &progress.Status,
		&progress.ProgressPercentage, &currentLessonID, &startedAt,
		&completedAt, &progress.LastAccessedAt, &progress.TimeSpentMinutes,
	)
	if err != nil {
		return nil
	}

	if currentLessonID.Valid {
		progress.CurrentLessonID = &currentLessonID.String
	}
	if startedAt.Valid {
		progress.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		progress.CompletedAt = &completedAt.Time
	}

	return &progress
}

// GetLearningCourse returns a single course by ID
func GetLearningCourse(c *gin.Context) {
	userID := c.GetString("userID")
	courseID := c.Param("id")

	db, exists := c.Get("db")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database connection not available",
		})
		return
	}

	query := `
		SELECT
			lc.id, lc.title, lc.description, lc.category_id, lc.level, lc.type,
			lc.content, lc.thumbnail_url, lc.duration_minutes, lc.estimated_read_time,
			lc.tags, lc.prerequisites, lc.learning_objectives, lc.status,
			lc.is_featured, lc.view_count, lc.rating, lc.total_ratings,
			lc.created_by, lc.created_at, lc.updated_at,
			lc.video_url, lc.quiz_questions, lc.article_content, lc.course_structure,
			cat.name as category_name,
			u.first_name || ' ' || u.last_name as created_by_name
		FROM learning_courses lc
		LEFT JOIN learning_categories cat ON lc.category_id = cat.id
		LEFT JOIN users u ON lc.created_by = u.id
		WHERE lc.id = ?
	`

	var course LearningCourse
	var content, thumbnailURL, estimatedReadTime sql.NullString
	var tags, prerequisites, objectives sql.NullString
	var videoURL, quizQuestions, articleContent, courseStructure sql.NullString
	var categoryName, createdByName sql.NullString

	err := db.(*sql.DB).QueryRow(query, courseID).Scan(
		&course.ID, &course.Title, &course.Description, &course.CategoryID,
		&course.Level, &course.Type, &content, &thumbnailURL,
		&course.DurationMinutes, &estimatedReadTime, &tags, &prerequisites,
		&objectives, &course.Status, &course.IsFeatured, &course.ViewCount,
		&course.Rating, &course.TotalRatings, &course.CreatedBy,
		&course.CreatedAt, &course.UpdatedAt, &videoURL, &quizQuestions,
		&articleContent, &courseStructure, &categoryName, &createdByName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Course not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to fetch course",
			})
		}
		return
	}

	course.Content = content.String
	course.ThumbnailURL = thumbnailURL.String
	course.EstimatedReadTime = estimatedReadTime.String
	course.VideoURL = videoURL.String
	course.CategoryName = categoryName.String
	course.CreatedByName = createdByName.String

	// Parse JSON fields
	if tags.Valid {
		json.Unmarshal([]byte(tags.String), &course.Tags)
	}
	if prerequisites.Valid {
		json.Unmarshal([]byte(prerequisites.String), &course.Prerequisites)
	}
	if objectives.Valid {
		json.Unmarshal([]byte(objectives.String), &course.LearningObjectives)
	}

	// Parse enhanced content fields
	if quizQuestions.Valid && quizQuestions.String != "" {
		json.Unmarshal([]byte(quizQuestions.String), &course.QuizQuestions)
	}
	if articleContent.Valid && articleContent.String != "" {
		var content ArticleContent
		if json.Unmarshal([]byte(articleContent.String), &content) == nil {
			course.ArticleContent = &content
		}
	}
	if courseStructure.Valid && courseStructure.String != "" {
		var structure CourseStructure
		if json.Unmarshal([]byte(courseStructure.String), &structure) == nil {
			course.CourseStructure = &structure
		}
	}

	// Parse main content field if it contains structured data
	if course.Content != "" {
		var contentData map[string]interface{}
		if json.Unmarshal([]byte(course.Content), &contentData) == nil {
			// Check if this is structured content
			if contentType, exists := contentData["type"]; exists {
				switch contentType {
				case "quiz":
					if questions, ok := contentData["questions"].([]interface{}); ok {
						questionsJSON, _ := json.Marshal(questions)
						json.Unmarshal(questionsJSON, &course.QuizQuestions)
					}
				case "article":
					if sections, ok := contentData["sections"].([]interface{}); ok {
						if headlineImage, ok := contentData["headline_image"].(string); ok {
							course.ArticleContent = &ArticleContent{
								HeadlineImage: headlineImage,
							}
							sectionsJSON, _ := json.Marshal(sections)
							json.Unmarshal(sectionsJSON, &course.ArticleContent.Sections)
						}
					}
				case "course":
					if topics, ok := contentData["topics"].([]interface{}); ok {
						if outline, ok := contentData["outline"].(string); ok {
							course.CourseStructure = &CourseStructure{
								Outline: outline,
							}
							topicsJSON, _ := json.Marshal(topics)
							json.Unmarshal(topicsJSON, &course.CourseStructure.Topics)
						}
					}
				}
			}
		}
		// For video type, if content is just a URL, set it as video_url
		if course.Type == "video" && course.VideoURL == "" && isValidURL(course.Content) {
			course.VideoURL = course.Content
		}
	}

	// Get user progress if user is authenticated
	if userID != "" {
		course.UserProgress = getUserCourseProgress(db.(*sql.DB), userID, course.ID)

		// Update view count and last accessed time
		go updateCourseViewCount(db.(*sql.DB), courseID, userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"course":  course,
	})
}

// StartCourse starts a course for a user
func StartCourse(c *gin.Context) {
	userID := c.GetString("userID")
	courseID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
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

	// Check if course exists
	var courseExists bool
	err := db.(*sql.DB).QueryRow("SELECT EXISTS(SELECT 1 FROM learning_courses WHERE id = ? AND status = 'published')", courseID).Scan(&courseExists)
	if err != nil || !courseExists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Course not found",
		})
		return
	}

	// Check if user already has progress for this course
	var existingProgressID string
	err = db.(*sql.DB).QueryRow("SELECT id FROM user_course_progress WHERE user_id = ? AND course_id = ?", userID, courseID).Scan(&existingProgressID)

	if err == sql.ErrNoRows {
		// Create new progress record
		progressID := uuid.New().String()
		now := time.Now()

		query := `
			INSERT INTO user_course_progress
			(id, user_id, course_id, status, progress_percentage, started_at, last_accessed_at, created_at, updated_at)
			VALUES (?, ?, ?, 'in_progress', 0, ?, ?, ?, ?)
		`

		_, err = db.(*sql.DB).Exec(query, progressID, userID, courseID, now, now, now, now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to start course",
			})
			return
		}
	} else if err == nil {
		// Update existing progress
		now := time.Now()
		query := `
			UPDATE user_course_progress
			SET status = 'in_progress', last_accessed_at = ?, updated_at = ?
			WHERE user_id = ? AND course_id = ?
		`

		_, err = db.(*sql.DB).Exec(query, now, now, userID, courseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to update course progress",
			})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check course progress",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Course started successfully",
	})
}

// Helper function to update course view count
func updateCourseViewCount(db *sql.DB, courseID, userID string) {
	// Update view count
	db.Exec("UPDATE learning_courses SET view_count = view_count + 1 WHERE id = ?", courseID)

	// Update user's last accessed time if they have progress
	now := time.Now()
	db.Exec("UPDATE user_course_progress SET last_accessed_at = ?, updated_at = ? WHERE user_id = ? AND course_id = ?",
		now, now, userID, courseID)
}

// SubmitQuizResults handles quiz result submission and progress tracking
func SubmitQuizResults(c *gin.Context) {
	courseID := c.Param("id")
	userID := c.GetString("userID")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
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

	var req struct {
		Score           int                      `json:"score"`
		CorrectAnswers  int                      `json:"correct_answers"`
		TotalQuestions  int                      `json:"total_questions"`
		Passed          bool                     `json:"passed"`
		TimeTaken       *int                     `json:"time_taken"`
		DetailedResults []map[string]interface{} `json:"detailed_results"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request data",
		})
		return
	}

	now := time.Now()

	// Update or create user course progress
	progressPercentage := 100 // Quiz completion is 100%
	status := "completed"     // Mark as completed if passed
	var completedAt *time.Time
	if req.Passed {
		status = "completed"
		completedAt = &now
	} else {
		status = "in_progress" // Keep as in progress if not passed
		completedAt = nil
	}

	// Check if progress record exists
	var existingID string
	checkQuery := `SELECT id FROM user_course_progress WHERE user_id = ? AND course_id = ?`
	err := db.(*sql.DB).QueryRow(checkQuery, userID, courseID).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new progress record
		progressID := uuid.New().String()
		progressQuery := `
			INSERT INTO user_course_progress
			(id, user_id, course_id, status, progress_percentage, completed_at, last_accessed_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		_, err = db.(*sql.DB).Exec(progressQuery,
			progressID, userID, courseID, status, progressPercentage, completedAt, now, now, now)
	} else if err == nil {
		// Update existing progress record
		progressQuery := `
			UPDATE user_course_progress
			SET status = ?, progress_percentage = ?, completed_at = ?, last_accessed_at = ?, updated_at = ?
			WHERE user_id = ? AND course_id = ?
		`
		_, err = db.(*sql.DB).Exec(progressQuery,
			status, progressPercentage, completedAt, now, now, userID, courseID)
	}
	if err != nil {
		log.Printf("Failed to update course progress: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update progress",
		})
		return
	}

	// Store quiz results (optional - for detailed analytics)
	detailedResultsJSON, _ := json.Marshal(req.DetailedResults)

	resultsQuery := `
		INSERT INTO quiz_results
		(user_id, course_id, score, correct_answers, total_questions, passed, time_taken, detailed_results, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.(*sql.DB).Exec(resultsQuery,
		userID, courseID, req.Score, req.CorrectAnswers, req.TotalQuestions,
		req.Passed, req.TimeTaken, string(detailedResultsJSON), now)
	if err != nil {
		// Don't fail the request if quiz results storage fails
		log.Printf("Failed to store quiz results: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quiz results submitted successfully",
		"data": gin.H{
			"progress_updated": true,
			"completed":        req.Passed,
			"score":            req.Score,
			"status":           status,
		},
	})
}

// isValidURL checks if a string is a valid URL
func isValidURL(str string) bool {
	if str == "" {
		return false
	}
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}
