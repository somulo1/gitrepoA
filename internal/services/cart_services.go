package services

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/utils"

// 	"github.com/google/uuid"
// )

// // AddToCart adds a product to user's cart
// func (s *MarketplaceService) AddToCart(userID, productID string, quantity int) error {
// 	// Check if product exists and is available
// 	product, err := s.GetProductByID(productID)
// 	if err != nil {
// 		return err
// 	}

// 	if !product.IsAvailable() {
// 		return fmt.Errorf("product is not available")
// 	}

// 	if !product.IsInStock(quantity) {
// 		return fmt.Errorf("insufficient stock")
// 	}

// 	if !product.CanOrder(quantity) {
// 		return fmt.Errorf("quantity not within order limits")
// 	}

// 	// Check if item already in cart
// 	var existingID string
// 	checkQuery := "SELECT id FROM cart_items WHERE user_id = ? AND product_id = ?"
// 	err = s.db.QueryRow(checkQuery, userID, productID).Scan(&existingID)

// 	if err == sql.ErrNoRows {
// 		// Add new item to cart
// 		cartItem := &models.CartItem{
// 			ID:        uuid.New().String(),
// 			UserID:    userID,
// 			ProductID: productID,
// 			Quantity:  quantity,
// 			Price:     product.Price,
// 			AddedAt:   utils.NowEAT(),
// 		}

// 		insertQuery := `
// 			INSERT INTO cart_items (id, user_id, product_id, quantity, price, added_at)
// 			VALUES (?, ?, ?, ?, ?, ?)
// 		`
// 		_, err = s.db.Exec(insertQuery,
// 			cartItem.ID, cartItem.UserID, cartItem.ProductID,
// 			cartItem.Quantity, cartItem.Price, cartItem.AddedAt,
// 		)
// 		if err != nil {
// 			return fmt.Errorf("failed to add to cart: %w", err)
// 		}
// 	} else if err == nil {
// 		// Update existing item quantity
// 		updateQuery := "UPDATE cart_items SET quantity = quantity + ?, price = ? WHERE id = ?"
// 		_, err = s.db.Exec(updateQuery, quantity, product.Price, existingID)
// 		if err != nil {
// 			return fmt.Errorf("failed to update cart item: %w", err)
// 		}
// 	} else {
// 		return fmt.Errorf("failed to check cart: %w", err)
// 	}

// 	return nil
// }

// // GetCart retrieves user's cart items with full product and seller details (only in-stock products)
// func (s *MarketplaceService) GetCart(userID string) ([]*models.CartItem, error) {
// 	query := `
// 		SELECT ci.id, ci.user_id, ci.product_id, ci.quantity, ci.price, ci.added_at,
// 			   p.id, p.name, p.description, p.category, p.price, p.currency, p.images,
// 			   p.status, p.stock, p.seller_id, p.county, p.town,
// 			   u.first_name, u.last_name, u.avatar
// 		FROM cart_items ci
// 		INNER JOIN products p ON ci.product_id = p.id
// 		INNER JOIN users u ON p.seller_id = u.id
// 		WHERE ci.user_id = ? AND p.status = ? AND p.stock > 0
// 		ORDER BY ci.added_at DESC
// 	`

// 	rows, err := s.db.Query(query, userID, models.ProductStatusActive)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query cart items: %w", err)
// 	}
// 	defer rows.Close()

// 	var cartItems []*models.CartItem
// 	for rows.Next() {
// 		var item models.CartItem
// 		var product models.Product
// 		var seller models.User
// 		var images string
// 		var sellerAvatar sql.NullString

// 		err := rows.Scan(
// 			&item.ID, &item.UserID, &item.ProductID, &item.Quantity, &item.Price, &item.AddedAt,
// 			&product.ID, &product.Name, &product.Description, &product.Category, &product.Price,
// 			&product.Currency, &images, &product.Status, &product.Stock, &product.SellerID,
// 			&product.County, &product.Town,
// 			&seller.FirstName, &seller.LastName, &sellerAvatar,
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

// 		// Set seller info
// 		seller.ID = product.SellerID
// 		if sellerAvatar.Valid {
// 			seller.Avatar = &sellerAvatar.String
// 		}
// 		product.Seller = &seller

// 		// Set product in cart item
// 		item.Product = &product

// 		cartItems = append(cartItems, &item)
// 	}

// 	return cartItems, nil
// }

// // RemoveFromCart removes an item from cart
// func (s *MarketplaceService) RemoveFromCart(userID, cartItemID string) error {
// 	query := "DELETE FROM cart_items WHERE id = ? AND user_id = ?"
// 	result, err := s.db.Exec(query, cartItemID, userID)
// 	if err != nil {
// 		return fmt.Errorf("failed to remove from cart: %w", err)
// 	}

// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return fmt.Errorf("failed to check affected rows: %w", err)
// 	}

// 	if rowsAffected == 0 {
// 		return fmt.Errorf("cart item not found")
// 	}

// 	return nil
// }
