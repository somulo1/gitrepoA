package services

// import (
// 	"encoding/json"
// 	"fmt"
// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/utils"

// 	"github.com/google/uuid"
// )

// // RemoveFromWishlist removes a product from user's wishlist
// func (s *MarketplaceService) RemoveFromWishlist(userID, productID string) error {
// 	result, err := s.db.Exec(
// 		"DELETE FROM wishlist WHERE user_id = ? AND product_id = ?",
// 		userID, productID,
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to remove from wishlist: %w", err)
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return fmt.Errorf("failed to get rows affected: %w", err)
// 	}

// 	if rowsAffected == 0 {
// 		return fmt.Errorf("item not found in wishlist")
// 	}

// 	return nil
// }

// // GetWishlist retrieves user's wishlist (only in-stock products)
// func (s *MarketplaceService) GetWishlist(userID string) ([]*models.WishlistItem, error) {
// 	query := `
// 		SELECT w.id, w.user_id, w.product_id, w.added_at,
// 			   p.id, p.name, p.description, p.category, p.price, p.currency, p.images,
// 			   p.status, p.stock, p.seller_id, p.county, p.town
// 		FROM wishlist w
// 		INNER JOIN products p ON w.product_id = p.id
// 		WHERE w.user_id = ? AND p.status = ? AND p.stock > 0
// 		ORDER BY w.added_at DESC
// 	`

// 	rows, err := s.db.Query(query, userID, models.ProductStatusActive)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query wishlist: %w", err)
// 	}
// 	defer rows.Close()

// 	var wishlistItems []*models.WishlistItem
// 	for rows.Next() {
// 		var item models.WishlistItem
// 		var product models.Product
// 		var images string

// 		err := rows.Scan(
// 			&item.ID, &item.UserID, &item.ProductID, &item.AddedAt,
// 			&product.ID, &product.Name, &product.Description, &product.Category,
// 			&product.Price, &product.Currency, &images, &product.Status,
// 			&product.Stock, &product.SellerID, &product.County, &product.Town,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		// Parse images JSON
// 		if images != "" {
// 			if err := json.Unmarshal([]byte(images), &product.Images); err != nil {
// 				product.Images = []string{}
// 			}
// 		}

// 		item.Product = &product
// 		wishlistItems = append(wishlistItems, &item)
// 	}

// 	return wishlistItems, nil
// }

// // AddToWishlist adds a product to user's wishlist
// func (s *MarketplaceService) AddToWishlist(userID, productID string) error {
// 	// Check if already in wishlist
// 	var exists bool
// 	err := s.db.QueryRow(
// 		"SELECT EXISTS(SELECT 1 FROM wishlist WHERE user_id = ? AND product_id = ?)",
// 		userID, productID,
// 	).Scan(&exists)
// 	if err != nil {
// 		return fmt.Errorf("failed to check wishlist: %w", err)
// 	}

// 	if exists {
// 		return fmt.Errorf("product already in wishlist")
// 	}

// 	// Add to wishlist
// 	_, err = s.db.Exec(
// 		"INSERT INTO wishlist (id, user_id, product_id, added_at) VALUES (?, ?, ?, ?)",
// 		uuid.New().String(), userID, productID, utils.NowEAT(),
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to add to wishlist: %w", err)
// 	}

// 	return nil
// }
