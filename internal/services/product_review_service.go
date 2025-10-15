package services

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"vaultke-backend/internal/models"
// )

// // GetProductReviews retrieves reviews for a product
// func (s *MarketplaceService) GetProductReviews(productID string) ([]*models.ProductReview, error) {
// 	query := `
// 		SELECT r.id, r.reviewer_id, r.product_id, r.order_id, r.rating, r.comment, r.images, r.created_at,
// 			   u.first_name, u.last_name, u.avatar
// 		FROM product_reviews r
// 		INNER JOIN users u ON r.reviewer_id = u.id
// 		WHERE r.product_id = ?
// 		ORDER BY r.created_at DESC
// 	`

// 	rows, err := s.db.Query(query, productID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query reviews: %w", err)
// 	}
// 	defer rows.Close()

// 	var reviews []*models.ProductReview
// 	for rows.Next() {
// 		var review models.ProductReview
// 		var user models.User
// 		var userAvatar, imagesJSON sql.NullString

// 		err := rows.Scan(
// 			&review.ID, &review.ReviewerID, &review.ProductID, &review.OrderID,
// 			&review.Rating, &review.Comment, &imagesJSON, &review.CreatedAt,
// 			&user.FirstName, &user.LastName, &userAvatar,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		// Parse images JSON
// 		var images []string
// 		if imagesJSON.Valid && imagesJSON.String != "" {
// 			if err := json.Unmarshal([]byte(imagesJSON.String), &images); err == nil {
// 				review.Images = images
// 			}
// 		}

// 		// Set user info
// 		user.ID = review.ReviewerID
// 		if userAvatar.Valid {
// 			user.Avatar = &userAvatar.String
// 		}
// 		review.Reviewer = &user

// 		reviews = append(reviews, &review)
// 	}

// 	return reviews, nil
// }
