package api

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"path/filepath"
// 	"strconv"
// 	"time"
// 	"vaultke-backend/internal/services"

// 	"github.com/gin-gonic/gin"
// )

// func GetReviews(c *gin.Context) {
// 	productID := c.Param("productId")
// 	if productID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Product ID is required",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Create marketplace service
// 	marketplaceService := services.NewMarketplaceService(db.(*sql.DB))

// 	// Get reviews for product
// 	reviews, err := marketplaceService.GetProductReviews(productID)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get reviews: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    reviews,
// 	})
// }

// func CreateReview(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Parse multipart form
// 	err := c.Request.ParseMultipartForm(10 << 20) // 10MB max
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Failed to parse form data: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Get form values
// 	productID := c.PostForm("productId")
// 	orderID := c.PostForm("orderId")
// 	ratingStr := c.PostForm("rating")
// 	comment := c.PostForm("comment")

// 	// Validate required fields
// 	if productID == "" || orderID == "" || ratingStr == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Product ID, Order ID, and Rating are required",
// 		})
// 		return
// 	}

// 	// Parse rating
// 	rating, err := strconv.Atoi(ratingStr)
// 	if err != nil || rating < 1 || rating > 5 {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Rating must be between 1 and 5",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Handle image uploads
// 	var imageURLs []string
// 	form := c.Request.MultipartForm
// 	if files, ok := form.File["images"]; ok {
// 		for _, file := range files {
// 			// Validate file type
// 			allowedTypes := map[string]bool{
// 				"image/jpeg": true,
// 				"image/jpg":  true,
// 				"image/png":  true,
// 				"image/webp": true,
// 			}

// 			if !allowedTypes[file.Header.Get("Content-Type")] {
// 				c.JSON(http.StatusBadRequest, gin.H{
// 					"success": false,
// 					"error":   "Invalid file type. Only JPEG, PNG, and WebP images are allowed",
// 				})
// 				return
// 			}

// 			// Validate file size (5MB max per image)
// 			if file.Size > 5*1024*1024 {
// 				c.JSON(http.StatusBadRequest, gin.H{
// 					"success": false,
// 					"error":   "Image too large. Maximum size is 5MB per image",
// 				})
// 				return
// 			}

// 			// Create uploads directory
// 			uploadDir := "./uploads/reviews"
// 			if err := os.MkdirAll(uploadDir, 0755); err != nil {
// 				c.JSON(http.StatusInternalServerError, gin.H{
// 					"success": false,
// 					"error":   "Failed to create upload directory",
// 				})
// 				return
// 			}

// 			// Generate unique filename
// 			ext := filepath.Ext(file.Filename)
// 			filename := fmt.Sprintf("review_%d_%s%s", time.Now().UnixNano(), userID.(string), ext)
// 			filePath := filepath.Join(uploadDir, filename)

// 			// Save file
// 			if err := c.SaveUploadedFile(file, filePath); err != nil {
// 				c.JSON(http.StatusInternalServerError, gin.H{
// 					"success": false,
// 					"error":   "Failed to save image",
// 				})
// 				return
// 			}

// 			// Add to image URLs (store relative path)
// 			imageURLs = append(imageURLs, "/uploads/reviews/"+filename)
// 		}
// 	}

// 	// Convert image URLs to JSON
// 	var imagesJSON string
// 	if len(imageURLs) > 0 {
// 		imagesBytes, _ := json.Marshal(imageURLs)
// 		imagesJSON = string(imagesBytes)
// 	}

// 	// Create review
// 	reviewID := fmt.Sprintf("REV_%d", time.Now().UnixNano())

// 	insertQuery := `
// 		INSERT INTO product_reviews (
// 			id, reviewer_id, product_id, order_id, rating, comment, images, created_at, updated_at
// 		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
// 	`

// 	_, err = db.(*sql.DB).Exec(insertQuery,
// 		reviewID, userID.(string), productID, orderID,
// 		rating, comment, imagesJSON,
// 	)
// 	if err != nil {
// 		// Clean up uploaded files on error
// 		for _, imageURL := range imageURLs {
// 			os.Remove("." + imageURL)
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to create review: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Return success response
// 	c.JSON(http.StatusCreated, gin.H{
// 		"success": true,
// 		"message": "Review created successfully",
// 		"data": map[string]interface{}{
// 			"id":        reviewID,
// 			"productId": productID,
// 			"orderId":   orderID,
// 			"rating":    rating,
// 			"comment":   comment,
// 			"images":    imageURLs,
// 		},
// 	})
// }

// func GetProductReviewStats(c *gin.Context) {
// 	productID := c.Param("productId")
// 	if productID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"success": false,
// 			"error":   "Product ID is required",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Get review statistics
// 	query := `
// 		SELECT
// 			COUNT(*) as total_reviews,
// 			AVG(rating) as average_rating,
// 			COUNT(CASE WHEN rating = 5 THEN 1 END) as five_star,
// 			COUNT(CASE WHEN rating = 4 THEN 1 END) as four_star,
// 			COUNT(CASE WHEN rating = 3 THEN 1 END) as three_star,
// 			COUNT(CASE WHEN rating = 2 THEN 1 END) as two_star,
// 			COUNT(CASE WHEN rating = 1 THEN 1 END) as one_star
// 		FROM product_reviews
// 		WHERE product_id = ?
// 	`

// 	var stats struct {
// 		TotalReviews  int     `json:"totalReviews"`
// 		AverageRating float64 `json:"averageRating"`
// 		FiveStar      int     `json:"fiveStar"`
// 		FourStar      int     `json:"fourStar"`
// 		ThreeStar     int     `json:"threeStar"`
// 		TwoStar       int     `json:"twoStar"`
// 		OneStar       int     `json:"oneStar"`
// 	}

// 	var averageRating sql.NullFloat64

// 	err := db.(*sql.DB).QueryRow(query, productID).Scan(
// 		&stats.TotalReviews, &averageRating,
// 		&stats.FiveStar, &stats.FourStar, &stats.ThreeStar,
// 		&stats.TwoStar, &stats.OneStar,
// 	)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to get review statistics: " + err.Error(),
// 		})
// 		return
// 	}

// 	// Handle NULL average rating (when no reviews exist)
// 	if averageRating.Valid {
// 		// Round average rating to 1 decimal place
// 		stats.AverageRating = float64(int(averageRating.Float64*10)) / 10
// 	} else {
// 		stats.AverageRating = 0.0
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    stats,
// 	})
// }

// func GetMyReviews(c *gin.Context) {
// 	// Get user ID from context (set by auth middleware)
// 	userID, exists := c.Get("userID")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{
// 			"success": false,
// 			"error":   "User not authenticated",
// 		})
// 		return
// 	}

// 	// Get database connection
// 	db, exists := c.Get("db")
// 	if !exists {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Database connection not available",
// 		})
// 		return
// 	}

// 	// Get user's reviews with product information
// 	query := `
// 		SELECT r.id, r.reviewer_id, r.product_id, r.order_id, r.rating, r.comment, r.images, r.created_at,
// 			   p.name as product_name, p.price as product_price, p.images as product_images
// 		FROM product_reviews r
// 		INNER JOIN products p ON r.product_id = p.id
// 		WHERE r.reviewer_id = ?
// 		ORDER BY r.created_at DESC
// 	`

// 	rows, err := db.(*sql.DB).Query(query, userID.(string))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{
// 			"success": false,
// 			"error":   "Failed to query reviews: " + err.Error(),
// 		})
// 		return
// 	}
// 	defer rows.Close()

// 	var reviews []map[string]interface{}
// 	for rows.Next() {
// 		var reviewID, reviewerID, productID, orderID, comment string
// 		var rating int
// 		var createdAt string
// 		var reviewImages, productName sql.NullString
// 		var productPrice float64
// 		var productImages sql.NullString

// 		err := rows.Scan(
// 			&reviewID, &reviewerID, &productID, &orderID, &rating, &comment, &reviewImages, &createdAt,
// 			&productName, &productPrice, &productImages,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		// Parse review images
// 		var reviewImagesList []string
// 		if reviewImages.Valid && reviewImages.String != "" {
// 			if err := json.Unmarshal([]byte(reviewImages.String), &reviewImagesList); err == nil {
// 				// Images parsed successfully
// 			}
// 		}

// 		// Parse product images
// 		var productImagesList []string
// 		if productImages.Valid && productImages.String != "" {
// 			if err := json.Unmarshal([]byte(productImages.String), &productImagesList); err == nil {
// 				// Images parsed successfully
// 			}
// 		}

// 		review := map[string]interface{}{
// 			"id":        reviewID,
// 			"productId": productID,
// 			"orderId":   orderID,
// 			"rating":    rating,
// 			"comment":   comment,
// 			"images":    reviewImagesList,
// 			"createdAt": createdAt,
// 			"product": map[string]interface{}{
// 				"name":   productName.String,
// 				"price":  productPrice,
// 				"images": productImagesList,
// 			},
// 		}

// 		reviews = append(reviews, review)
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"success": true,
// 		"data":    reviews,
// 	})
// }
