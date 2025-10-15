package services

// import (
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"strings"
// 	"time"
// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/utils"

// 	"github.com/google/uuid"
// )

// // GetOrders retrieves orders for a user (as buyer or seller)
// func (s *MarketplaceService) GetOrders(userID string, filters map[string]interface{}, limit, offset int) ([]*models.Order, error) {
// 	role := filters["role"].(string)

// 	var query string
// 	var args []interface{}

// 	if role == "seller" {
// 		query = `
// 			SELECT o.id, o.buyer_id, o.seller_id, o.chama_id, o.total_amount, o.currency,
// 				   o.status, o.payment_method, o.payment_status, o.delivery_county, o.delivery_town,
// 				   o.delivery_address, o.delivery_phone, o.delivery_fee, o.delivery_status,
// 				   o.delivery_person_id, o.notes, o.created_at, o.updated_at,
// 				   buyer.first_name as buyer_first_name, buyer.last_name as buyer_last_name,
// 				   buyer.avatar as buyer_avatar
// 			FROM orders o
// 			INNER JOIN users buyer ON o.buyer_id = buyer.id
// 			WHERE o.seller_id = ?
// 		`
// 		args = append(args, userID)
// 	} else {
// 		query = `
// 			SELECT o.id, o.buyer_id, o.seller_id, o.chama_id, o.total_amount, o.currency,
// 				   o.status, o.payment_method, o.payment_status, o.delivery_county, o.delivery_town,
// 				   o.delivery_address, o.delivery_phone, o.delivery_fee, o.delivery_status,
// 				   o.delivery_person_id, o.notes, o.created_at, o.updated_at,
// 				   seller.first_name as seller_first_name, seller.last_name as seller_last_name,
// 				   seller.avatar as seller_avatar
// 			FROM orders o
// 			INNER JOIN users seller ON o.seller_id = seller.id
// 			WHERE o.buyer_id = ?
// 		`
// 		args = append(args, userID)
// 	}

// 	// Add status filter
// 	if status, ok := filters["status"]; ok {
// 		query += " AND o.status = ?"
// 		args = append(args, status)
// 	}

// 	query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
// 	args = append(args, limit, offset)

// 	rows, err := s.db.Query(query, args...)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query orders: %w", err)
// 	}
// 	defer rows.Close()

// 	var orders []*models.Order
// 	for rows.Next() {
// 		var order models.Order
// 		var otherUser models.User
// 		var chamaID sql.NullString
// 		var deliveryPersonID sql.NullString
// 		var otherUserAvatar sql.NullString

// 		err := rows.Scan(
// 			&order.ID, &order.BuyerID, &order.SellerID, &chamaID, &order.TotalAmount,
// 			&order.Currency, &order.Status, &order.PaymentMethod, &order.PaymentStatus,
// 			&order.DeliveryCounty, &order.DeliveryTown, &order.DeliveryAddress,
// 			&order.DeliveryPhone, &order.DeliveryFee, &order.DeliveryStatus,
// 			&deliveryPersonID, &order.Notes, &order.CreatedAt, &order.UpdatedAt,
// 			&otherUser.FirstName, &otherUser.LastName, &otherUserAvatar,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		// Set optional fields
// 		if chamaID.Valid {
// 			order.ChamaID = &chamaID.String
// 		}
// 		if deliveryPersonID.Valid {
// 			order.DeliveryPersonID = &deliveryPersonID.String
// 		}
// 		if otherUserAvatar.Valid {
// 			otherUser.Avatar = &otherUserAvatar.String
// 		}

// 		// Set the other user (buyer for seller, seller for buyer)
// 		if role == "seller" {
// 			otherUser.ID = order.BuyerID
// 			order.Buyer = &otherUser
// 		} else {
// 			otherUser.ID = order.SellerID
// 			order.Seller = &otherUser
// 		}

// 		orders = append(orders, &order)
// 	}

// 	return orders, nil
// }

// // GetOrderByID retrieves a specific order by ID with full details
// func (s *MarketplaceService) GetOrderByID(orderID string, userID string) (*models.Order, error) {
// 	// Query to get order with buyer, seller, and delivery person details
// 	query := `
// 		SELECT o.id, o.buyer_id, o.seller_id, o.chama_id, o.total_amount, o.currency,
// 			   o.status, o.payment_method, o.payment_status, o.delivery_county,
// 			   o.delivery_town, o.delivery_address, o.delivery_phone, o.delivery_fee,
// 			   o.delivery_status, o.delivery_person_id, o.estimated_delivery,
// 			   o.actual_delivery, o.notes, o.created_at, o.updated_at,
// 			   buyer.first_name as buyer_first_name, buyer.last_name as buyer_last_name,
// 			   buyer.avatar as buyer_avatar, buyer.phone as buyer_phone,
// 			   seller.first_name as seller_first_name, seller.last_name as seller_last_name,
// 			   seller.avatar as seller_avatar, seller.phone as seller_phone,
// 			   dp.first_name as dp_first_name, dp.last_name as dp_last_name,
// 			   dp.avatar as dp_avatar, dp.phone as dp_phone
// 		FROM orders o
// 		INNER JOIN users buyer ON o.buyer_id = buyer.id
// 		INNER JOIN users seller ON o.seller_id = seller.id
// 		LEFT JOIN users dp ON o.delivery_person_id = dp.id
// 		WHERE o.id = ? AND (o.buyer_id = ? OR o.seller_id = ?)
// 	`

// 	var order models.Order
// 	var buyer, seller, deliveryPerson models.User
// 	var chamaID, deliveryPersonID sql.NullString
// 	var estimatedDelivery, actualDelivery sql.NullTime
// 	var buyerAvatar, buyerPhone, sellerAvatar, sellerPhone sql.NullString
// 	var dpFirstName, dpLastName, dpAvatar, dpPhone sql.NullString

// 	err := s.db.QueryRow(query, orderID, userID, userID).Scan(
// 		&order.ID, &order.BuyerID, &order.SellerID, &chamaID, &order.TotalAmount,
// 		&order.Currency, &order.Status, &order.PaymentMethod, &order.PaymentStatus,
// 		&order.DeliveryCounty, &order.DeliveryTown, &order.DeliveryAddress,
// 		&order.DeliveryPhone, &order.DeliveryFee, &order.DeliveryStatus,
// 		&deliveryPersonID, &estimatedDelivery, &actualDelivery, &order.Notes,
// 		&order.CreatedAt, &order.UpdatedAt,
// 		&buyer.FirstName, &buyer.LastName, &buyerAvatar, &buyerPhone,
// 		&seller.FirstName, &seller.LastName, &sellerAvatar, &sellerPhone,
// 		&dpFirstName, &dpLastName, &dpAvatar, &dpPhone,
// 	)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, fmt.Errorf("order not found")
// 		}
// 		return nil, fmt.Errorf("failed to get order: %w", err)
// 	}

// 	// Set optional fields
// 	if chamaID.Valid {
// 		order.ChamaID = &chamaID.String
// 	}
// 	if deliveryPersonID.Valid {
// 		order.DeliveryPersonID = &deliveryPersonID.String
// 	}
// 	if estimatedDelivery.Valid {
// 		order.EstimatedDelivery = &estimatedDelivery.Time
// 	}
// 	if actualDelivery.Valid {
// 		order.ActualDelivery = &actualDelivery.Time
// 	}

// 	// Set buyer details
// 	buyer.ID = order.BuyerID
// 	if buyerAvatar.Valid {
// 		buyer.Avatar = &buyerAvatar.String
// 	}
// 	if buyerPhone.Valid {
// 		buyer.Phone = buyerPhone.String
// 	}
// 	order.Buyer = &buyer

// 	// Set seller details
// 	seller.ID = order.SellerID
// 	if sellerAvatar.Valid {
// 		seller.Avatar = &sellerAvatar.String
// 	}
// 	if sellerPhone.Valid {
// 		seller.Phone = sellerPhone.String
// 	}
// 	order.Seller = &seller

// 	// Set delivery person details if exists
// 	if deliveryPersonID.Valid {
// 		deliveryPerson.ID = deliveryPersonID.String
// 		if dpFirstName.Valid {
// 			deliveryPerson.FirstName = dpFirstName.String
// 		}
// 		if dpLastName.Valid {
// 			deliveryPerson.LastName = dpLastName.String
// 		}
// 		if dpAvatar.Valid {
// 			deliveryPerson.Avatar = &dpAvatar.String
// 		}
// 		if dpPhone.Valid {
// 			deliveryPerson.Phone = dpPhone.String
// 		}
// 		order.DeliveryPerson = &deliveryPerson
// 	}

// 	// Get order items
// 	itemsQuery := `
// 		SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price, oi.name,
// 			   p.images, p.category, p.seller_id
// 		FROM order_items oi
// 		LEFT JOIN products p ON oi.product_id = p.id
// 		WHERE oi.order_id = ?
// 	`

// 	rows, err := s.db.Query(itemsQuery, orderID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get order items: %w", err)
// 	}
// 	defer rows.Close()

// 	var items []models.OrderItem
// 	for rows.Next() {
// 		var item models.OrderItem
// 		var product models.Product
// 		var imagesJSON sql.NullString

// 		err := rows.Scan(
// 			&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
// 			&item.Price, &item.Name, &imagesJSON, &product.Category, &product.SellerID,
// 		)
// 		if err != nil {
// 			continue
// 		}

// 		// Parse images
// 		if imagesJSON.Valid && imagesJSON.String != "" {
// 			if err := json.Unmarshal([]byte(imagesJSON.String), &product.Images); err == nil {
// 				item.Product = &product
// 			}
// 		}

// 		items = append(items, item)
// 	}

// 	order.Items = items
// 	return &order, nil
// }

// // CreateOrderFromCart creates an order from selected cart items
// func (s *MarketplaceService) CreateOrderFromCart(userID string, cartItemIDs []string, orderData *models.OrderCreation) (*models.Order, error) {
// 	// Start transaction
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to start transaction: %w", err)
// 	}
// 	defer tx.Rollback()

// 	// Get cart items
// 	if len(cartItemIDs) == 0 {
// 		return nil, fmt.Errorf("no cart items selected")
// 	}

// 	// Build query to get cart items with product details
// 	placeholders := make([]string, len(cartItemIDs))
// 	args := make([]interface{}, len(cartItemIDs)+1)
// 	args[0] = userID
// 	for i, id := range cartItemIDs {
// 		placeholders[i] = "?"
// 		args[i+1] = id
// 	}

// 	query := fmt.Sprintf(`
// 		SELECT ci.id, ci.product_id, ci.quantity, ci.price,
// 			   p.name, p.seller_id, p.chama_id, p.stock
// 		FROM cart_items ci
// 		INNER JOIN products p ON ci.product_id = p.id
// 		WHERE ci.user_id = ? AND ci.id IN (%s)
// 	`, strings.Join(placeholders, ","))

// 	rows, err := tx.Query(query, args...)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get cart items: %w", err)
// 	}
// 	defer rows.Close()

// 	var cartItems []struct {
// 		ID        string
// 		ProductID string
// 		Quantity  int
// 		Price     float64
// 		Name      string
// 		SellerID  string
// 		ChamaID   sql.NullString
// 		Stock     int
// 	}

// 	var totalAmount float64
// 	var sellerID string
// 	var chamaID *string

// 	for rows.Next() {
// 		var item struct {
// 			ID        string
// 			ProductID string
// 			Quantity  int
// 			Price     float64
// 			Name      string
// 			SellerID  string
// 			ChamaID   sql.NullString
// 			Stock     int
// 		}

// 		err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity, &item.Price,
// 			&item.Name, &item.SellerID, &item.ChamaID, &item.Stock)
// 		if err != nil {
// 			continue
// 		}

// 		// Check stock
// 		if item.Stock < item.Quantity {
// 			return nil, fmt.Errorf("insufficient stock for product %s", item.Name)
// 		}

// 		// Ensure all items are from the same seller
// 		if sellerID == "" {
// 			sellerID = item.SellerID
// 			if item.ChamaID.Valid {
// 				chamaID = &item.ChamaID.String
// 			}
// 		} else if sellerID != item.SellerID {
// 			return nil, fmt.Errorf("all items must be from the same seller")
// 		}

// 		totalAmount += item.Price * float64(item.Quantity)
// 		cartItems = append(cartItems, item)
// 	}

// 	if len(cartItems) == 0 {
// 		return nil, fmt.Errorf("no valid cart items found")
// 	}

// 	// Create order
// 	order := &models.Order{
// 		ID:              uuid.New().String(),
// 		BuyerID:         userID,
// 		SellerID:        sellerID,
// 		ChamaID:         chamaID,
// 		TotalAmount:     totalAmount,
// 		Currency:        "KES",
// 		Status:          models.OrderStatusPending,
// 		PaymentMethod:   orderData.PaymentMethod,
// 		PaymentStatus:   "pending",
// 		DeliveryCounty:  orderData.DeliveryCounty,
// 		DeliveryTown:    orderData.DeliveryTown,
// 		DeliveryAddress: orderData.DeliveryAddress,
// 		DeliveryPhone:   orderData.DeliveryPhone,
// 		DeliveryFee:     0, // Calculate based on location
// 		DeliveryStatus:  models.DeliveryStatusPending,
// 		Notes:           orderData.Notes,
// 		CreatedAt:       time.Now(),
// 		UpdatedAt:       time.Now(),
// 	}

// 	// Insert order
// 	orderQuery := `
// 		INSERT INTO orders (
// 			id, buyer_id, seller_id, chama_id, total_amount, currency, status,
// 			payment_method, payment_status, delivery_county, delivery_town,
// 			delivery_address, delivery_phone, delivery_fee, delivery_status,
// 			notes, created_at, updated_at
// 		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
// 	`

// 	_, err = tx.Exec(orderQuery,
// 		order.ID, order.BuyerID, order.SellerID, order.ChamaID,
// 		order.TotalAmount, order.Currency, order.Status, order.PaymentMethod,
// 		order.PaymentStatus, order.DeliveryCounty, order.DeliveryTown,
// 		order.DeliveryAddress, order.DeliveryPhone, order.DeliveryFee,
// 		order.DeliveryStatus, order.Notes, order.CreatedAt, order.UpdatedAt,
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create order: %w", err)
// 	}

// 	// Insert order items and update stock
// 	for _, item := range cartItems {
// 		orderItem := &models.OrderItem{
// 			ID:        uuid.New().String(),
// 			OrderID:   order.ID,
// 			ProductID: item.ProductID,
// 			Quantity:  item.Quantity,
// 			Price:     item.Price,
// 			Name:      item.Name,
// 		}

// 		itemQuery := `
// 			INSERT INTO order_items (id, order_id, product_id, quantity, price, name)
// 			VALUES (?, ?, ?, ?, ?, ?)
// 		`

// 		_, err = tx.Exec(itemQuery,
// 			orderItem.ID, orderItem.OrderID, orderItem.ProductID,
// 			orderItem.Quantity, orderItem.Price, orderItem.Name,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create order item: %w", err)
// 		}

// 		// Update product stock
// 		_, err = tx.Exec(
// 			"UPDATE products SET stock = stock - ? WHERE id = ?",
// 			item.Quantity, item.ProductID,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to update product stock: %w", err)
// 		}

// 		// Remove from cart
// 		_, err = tx.Exec("DELETE FROM cart_items WHERE id = ?", item.ID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to clear cart: %w", err)
// 		}
// 	}

// 	// Commit transaction
// 	if err = tx.Commit(); err != nil {
// 		return nil, fmt.Errorf("failed to commit transaction: %w", err)
// 	}

// 	return order, nil
// }

// // CreateOrder creates a new order from cart items
// func (s *MarketplaceService) CreateOrder(creation *models.OrderCreation, buyerID string) (*models.Order, error) {
// 	// Validate input
// 	if err := utils.ValidateStruct(creation); err != nil {
// 		return nil, fmt.Errorf("validation error: %w", err)
// 	}

// 	// Start transaction
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to start transaction: %w", err)
// 	}
// 	defer tx.Rollback()

// 	// Calculate total amount
// 	var totalAmount float64
// 	var sellerID string
// 	var chamaID *string

// 	for i, item := range creation.Items {
// 		if i == 0 {
// 			sellerID = item.Product.SellerID
// 			chamaID = item.Product.ChamaID
// 		} else {
// 			// Ensure all items are from the same seller
// 			if item.Product.SellerID != sellerID {
// 				return nil, fmt.Errorf("all items must be from the same seller")
// 			}
// 		}
// 		totalAmount += item.GetTotalPrice()
// 	}

// 	// Create order
// 	order := &models.Order{
// 		ID:              uuid.New().String(),
// 		BuyerID:         buyerID,
// 		SellerID:        sellerID,
// 		ChamaID:         chamaID,
// 		TotalAmount:     totalAmount,
// 		Currency:        "KES",
// 		Status:          models.OrderStatusPending,
// 		PaymentMethod:   creation.PaymentMethod,
// 		PaymentStatus:   "pending",
// 		DeliveryCounty:  creation.DeliveryCounty,
// 		DeliveryTown:    creation.DeliveryTown,
// 		DeliveryAddress: creation.DeliveryAddress,
// 		DeliveryPhone:   creation.DeliveryPhone,
// 		DeliveryFee:     0, // Calculate based on location
// 		DeliveryStatus:  models.DeliveryStatusPending,
// 		Notes:           creation.Notes,
// 		CreatedAt:       utils.NowEAT(),
// 		UpdatedAt:       utils.NowEAT(),
// 	}

// 	// Insert order
// 	orderQuery := `
// 		INSERT INTO orders (
// 			id, buyer_id, seller_id, chama_id, total_amount, currency, status,
// 			payment_method, payment_status, delivery_county, delivery_town,
// 			delivery_address, delivery_phone, delivery_fee, delivery_status,
// 			notes, created_at, updated_at
// 		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
// 	`

// 	_, err = tx.Exec(orderQuery,
// 		order.ID, order.BuyerID, order.SellerID, order.ChamaID,
// 		order.TotalAmount, order.Currency, order.Status, order.PaymentMethod,
// 		order.PaymentStatus, order.DeliveryCounty, order.DeliveryTown,
// 		order.DeliveryAddress, order.DeliveryPhone, order.DeliveryFee,
// 		order.DeliveryStatus, order.Notes, order.CreatedAt, order.UpdatedAt,
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create order: %w", err)
// 	}

// 	// Insert order items
// 	for _, item := range creation.Items {
// 		orderItem := &models.OrderItem{
// 			ID:        uuid.New().String(),
// 			OrderID:   order.ID,
// 			ProductID: item.ProductID,
// 			Quantity:  item.Quantity,
// 			Price:     item.Price,
// 			Name:      item.Product.Name,
// 		}

// 		itemQuery := `
// 			INSERT INTO order_items (id, order_id, product_id, quantity, price, name)
// 			VALUES (?, ?, ?, ?, ?, ?)
// 		`

// 		_, err = tx.Exec(itemQuery,
// 			orderItem.ID, orderItem.OrderID, orderItem.ProductID,
// 			orderItem.Quantity, orderItem.Price, orderItem.Name,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create order item: %w", err)
// 		}

// 		// Update product stock
// 		_, err = tx.Exec(
// 			"UPDATE products SET stock = stock - ? WHERE id = ?",
// 			item.Quantity, item.ProductID,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to update product stock: %w", err)
// 		}

// 		// Remove from cart
// 		_, err = tx.Exec(
// 			"DELETE FROM cart_items WHERE user_id = ? AND product_id = ?",
// 			buyerID, item.ProductID,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to clear cart: %w", err)
// 		}
// 	}

// 	// Commit transaction
// 	if err = tx.Commit(); err != nil {
// 		return nil, fmt.Errorf("failed to commit transaction: %w", err)
// 	}

// 	return order, nil
// }
