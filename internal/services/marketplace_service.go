package services

// import (
// 	"context"
// 	"crypto/md5"
// 	"database/sql"
// 	"encoding/json"
// 	"fmt"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/google/uuid"

// 	"vaultke-backend/internal/models"
// 	"vaultke-backend/internal/utils"
// )

// // Cache structure for products
// type ProductCache struct {
// 	Products  []*models.Product `json:"products"`
// 	Timestamp time.Time         `json:"timestamp"`
// 	Filters   string            `json:"filters"`
// }

// // Cache storage
// type CacheStore struct {
// 	mu    sync.RWMutex
// 	cache map[string]*ProductCache
// }

// func NewCacheStore() *CacheStore {
// 	return &CacheStore{
// 		cache: make(map[string]*ProductCache),
// 	}
// }

// func (cs *CacheStore) Get(key string) (*ProductCache, bool) {
// 	cs.mu.RLock()
// 	defer cs.mu.RUnlock()

// 	cached, exists := cs.cache[key]
// 	if !exists {
// 		return nil, false
// 	}

// 	// Check if cache is still valid (5 minutes)
// 	if time.Since(cached.Timestamp) > 5*time.Minute {
// 		return nil, false
// 	}

// 	return cached, true
// }

// func (cs *CacheStore) Set(key string, products []*models.Product, filters string) {
// 	cs.mu.Lock()
// 	defer cs.mu.Unlock()

// 	cs.cache[key] = &ProductCache{
// 		Products:  products,
// 		Timestamp: time.Now(),
// 		Filters:   filters,
// 	}
// }

// func (cs *CacheStore) Clear() {
// 	cs.mu.Lock()
// 	defer cs.mu.Unlock()
// 	cs.cache = make(map[string]*ProductCache)
// }

// // MarketplaceService handles marketplace-related business logic
// type MarketplaceService struct {
// 	db    *sql.DB
// 	cache *CacheStore
// }

// // Global cache instance
// var globalCache = NewCacheStore()

// // NewMarketplaceService creates a new marketplace service
// func NewMarketplaceService(db *sql.DB) *MarketplaceService {
// 	service := &MarketplaceService{
// 		db:    db,
// 		cache: globalCache,
// 	}

// 	// Create performance indexes on first initialization
// 	service.createPerformanceIndexes()

// 	return service
// }

// // createPerformanceIndexes creates database indexes for fast product queries
// func (s *MarketplaceService) createPerformanceIndexes() {
// 	indexes := []string{
// 		"CREATE INDEX IF NOT EXISTS idx_products_status_stock ON products(status, stock)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_category ON products(category)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_county ON products(county)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_chama_id ON products(chama_id)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_seller_id ON products(seller_id)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_search ON products(name, description)",
// 		"CREATE INDEX IF NOT EXISTS idx_products_promoted ON products(is_promoted, created_at DESC)",
// 	}

// 	for _, indexSQL := range indexes {
// 		if _, err := s.db.Exec(indexSQL); err != nil {
// 			fmt.Printf("⚠️ Failed to create index: %v\n", err)
// 		}
// 	}
// 	fmt.Println("✅ Database indexes created for marketplace performance")
// }

// // CreateProduct creates a new product
// func (s *MarketplaceService) CreateProduct(creation *models.ProductCreation, sellerID string, chamaID *string) (*models.Product, error) {
// 	// Validate input
// 	if err := utils.ValidateStruct(creation); err != nil {
// 		return nil, fmt.Errorf("validation error: %w", err)
// 	}

// 	product := &models.Product{
// 		ID:           uuid.New().String(),
// 		Name:         creation.Name,
// 		Description:  creation.Description,
// 		Category:     creation.Category,
// 		Price:        creation.Price,
// 		Currency:     "KES",
// 		Images:       creation.Images,
// 		Status:       models.ProductStatusActive,
// 		Stock:        creation.Stock,
// 		MinOrder:     creation.MinOrder,
// 		MaxOrder:     creation.MaxOrder,
// 		SellerID:     sellerID,
// 		ChamaID:      chamaID,
// 		County:       creation.County,
// 		Town:         creation.Town,
// 		Address:      creation.Address,
// 		Tags:         creation.Tags,
// 		Rating:       0,
// 		TotalRatings: 0,
// 		TotalSales:   0,
// 		IsPromoted:   false,
// 		CreatedAt:    utils.NowEAT(),
// 		UpdatedAt:    utils.NowEAT(),
// 	}

// 	// Serialize JSON fields
// 	imagesJSON, err := product.GetImagesJSON()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to serialize images: %w", err)
// 	}

// 	tagsJSON, err := product.GetTagsJSON()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to serialize tags: %w", err)
// 	}

// 	query := `
// 		INSERT INTO products (
// 			id, name, description, category, price, currency, images, status,
// 			stock, min_order, max_order, seller_id, chama_id, county, town,
// 			address, tags, rating, total_ratings, total_sales, is_promoted,
// 			created_at, updated_at
// 		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
// 	`

// 	_, err = s.db.Exec(query,
// 		product.ID, product.Name, product.Description, product.Category,
// 		product.Price, product.Currency, imagesJSON, product.Status,
// 		product.Stock, product.MinOrder, product.MaxOrder, product.SellerID,
// 		product.ChamaID, product.County, product.Town, product.Address,
// 		tagsJSON, product.Rating, product.TotalRatings, product.TotalSales,
// 		product.IsPromoted, product.CreatedAt, product.UpdatedAt,
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create product: %w", err)
// 	}

// 	return product, nil
// }

// // GetProductByID retrieves a product by ID with seller information
// func (s *MarketplaceService) GetProductByID(productID string) (*models.Product, error) {
// 	query := `
// 		SELECT p.id, p.name, p.description, p.category, p.price, p.currency, p.images, p.status,
// 			   p.stock, p.min_order, p.max_order, p.seller_id, p.chama_id, p.county, p.town,
// 			   p.address, p.tags, p.rating, p.total_ratings, p.total_sales, p.is_promoted,
// 			   p.created_at, p.updated_at,
// 			   u.first_name, u.last_name, u.avatar
// 		FROM products p
// 		INNER JOIN users u ON p.seller_id = u.id
// 		WHERE p.id = ?
// 	`

// 	product := &models.Product{}
// 	seller := &models.User{}
// 	var imagesJSON, tagsJSON string

// 	err := s.db.QueryRow(query, productID).Scan(
// 		&product.ID, &product.Name, &product.Description, &product.Category,
// 		&product.Price, &product.Currency, &imagesJSON, &product.Status,
// 		&product.Stock, &product.MinOrder, &product.MaxOrder, &product.SellerID,
// 		&product.ChamaID, &product.County, &product.Town, &product.Address,
// 		&tagsJSON, &product.Rating, &product.TotalRatings, &product.TotalSales,
// 		&product.IsPromoted, &product.CreatedAt, &product.UpdatedAt,
// 		&seller.FirstName, &seller.LastName, &seller.Avatar,
// 	)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, fmt.Errorf("product not found")
// 		}
// 		return nil, fmt.Errorf("failed to get product: %w", err)
// 	}

// 	// Parse JSON fields
// 	if err = product.SetImagesFromJSON(imagesJSON); err != nil {
// 		return nil, fmt.Errorf("failed to parse images: %w", err)
// 	}
// 	if err = product.SetTagsFromJSON(tagsJSON); err != nil {
// 		return nil, fmt.Errorf("failed to parse tags: %w", err)
// 	}

// 	// Set seller information
// 	seller.ID = product.SellerID
// 	product.Seller = seller

// 	return product, nil
// }

// // generateCacheKey creates a unique cache key for the given filters
// func (s *MarketplaceService) generateCacheKey(filters map[string]interface{}, limit, offset int) string {
// 	// Convert filters to JSON for consistent key generation
// 	filtersJSON, _ := json.Marshal(filters)
// 	keyData := fmt.Sprintf("%s_%d_%d", string(filtersJSON), limit, offset)
// 	return fmt.Sprintf("%x", md5.Sum([]byte(keyData)))
// }

// // GetProducts retrieves products with filters (only in-stock products) with caching
// func (s *MarketplaceService) GetProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, error) {
// 	// Generate cache key
// 	cacheKey := s.generateCacheKey(filters, limit, offset)

// 	// Try to get from cache first
// 	if cached, found := s.cache.Get(cacheKey); found {
// 		fmt.Printf("🚀 Cache HIT for products query: %s\n", cacheKey[:8])
// 		return cached.Products, nil
// 	}

// 	fmt.Printf("💾 Cache MISS for products query: %s\n", cacheKey[:8])

// 	// Ultra-optimized query for billion-scale database
// 	query := `
// 		SELECT p.id, p.name, p.category, p.price, p.currency,
// 			   COALESCE(p.images, '[]') as images, p.stock, p.seller_id,
// 			   u.first_name, u.last_name
// 		FROM products p
// 		INNER JOIN users u ON p.seller_id = u.id
// 		WHERE p.status = ? AND p.stock > 0
// 	`
// 	args := []interface{}{models.ProductStatusActive}

// 	// Add filters
// 	if category, ok := filters["category"]; ok {
// 		query += " AND p.category = ?"
// 		args = append(args, category)
// 	}
// 	if county, ok := filters["county"]; ok {
// 		query += " AND p.county = ?"
// 		args = append(args, county)
// 	}
// 	if chamaID, ok := filters["chamaId"]; ok {
// 		query += " AND p.chama_id = ?"
// 		args = append(args, chamaID)
// 	}
// 	if search, ok := filters["search"]; ok {
// 		query += " AND (p.name LIKE ? OR p.description LIKE ?)"
// 		searchTerm := "%" + search.(string) + "%"
// 		args = append(args, searchTerm, searchTerm)
// 	}

// 	query += " ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
// 	args = append(args, limit, offset)

// 	// Add query timeout to prevent hanging (increased for tunnel latency)
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// Log query performance
// 	startTime := time.Now()
// 	rows, err := s.db.QueryContext(ctx, query, args...)
// 	queryTime := time.Since(startTime)

// 	if err != nil {
// 		if err == context.DeadlineExceeded {
// 			fmt.Printf("⚠️ Query timeout after 2s - query too slow for billion-scale\n")
// 			return nil, fmt.Errorf("query timeout: products query took too long")
// 		}
// 		return nil, fmt.Errorf("failed to get products: %w", err)
// 	}
// 	defer rows.Close()

// 	fmt.Printf("⚡ Query executed in %v\n", queryTime)

// 	var products []*models.Product
// 	for rows.Next() {
// 		product := &models.Product{}
// 		seller := &models.User{}
// 		var imagesJSON, tagsJSON string

// 		err := rows.Scan(
// 			&product.ID, &product.Name, &product.Category, &product.Price,
// 			&product.Currency, &imagesJSON, &product.Stock, &product.SellerID,
// 			&seller.FirstName, &seller.LastName,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to scan product: %w", err)
// 		}

// 		// Parse JSON fields
// 		if err = product.SetImagesFromJSON(imagesJSON); err != nil {
// 			return nil, fmt.Errorf("failed to parse images: %w", err)
// 		}
// 		if err = product.SetTagsFromJSON(tagsJSON); err != nil {
// 			return nil, fmt.Errorf("failed to parse tags: %w", err)
// 		}

// 		seller.ID = product.SellerID
// 		product.Seller = seller
// 		products = append(products, product)
// 	}

// 	// Store in cache for future requests
// 	filtersJSON, _ := json.Marshal(filters)
// 	s.cache.Set(cacheKey, products, string(filtersJSON))
// 	fmt.Printf("💾 Cached %d products for key: %s\n", len(products), cacheKey[:8])

// 	return products, nil
// }

// // GetAllProducts retrieves all products including out of stock (for admin/seller use)
// func (s *MarketplaceService) GetAllProducts(filters map[string]interface{}, limit, offset int) ([]*models.Product, error) {
// 	query := `
// 		SELECT p.id, p.name, p.description, p.category, p.price, p.currency, p.images, p.status,
// 			   p.stock, p.min_order, p.max_order, p.seller_id, p.chama_id, p.county, p.town,
// 			   p.address, p.tags, p.rating, p.total_ratings, p.total_sales, p.is_promoted,
// 			   p.created_at, p.updated_at,
// 			   u.first_name, u.last_name, u.avatar
// 		FROM products p
// 		INNER JOIN users u ON p.seller_id = u.id
// 		WHERE p.status = ?
// 	`
// 	args := []interface{}{models.ProductStatusActive}

// 	// Add filters (same as GetProducts but without stock filter)
// 	if category, ok := filters["category"]; ok {
// 		query += " AND p.category = ?"
// 		args = append(args, category)
// 	}
// 	if county, ok := filters["county"]; ok {
// 		query += " AND p.county = ?"
// 		args = append(args, county)
// 	}
// 	if chamaID, ok := filters["chamaId"]; ok {
// 		query += " AND p.chama_id = ?"
// 		args = append(args, chamaID)
// 	}
// 	if sellerID, ok := filters["sellerId"]; ok {
// 		query += " AND p.seller_id = ?"
// 		args = append(args, sellerID)
// 	}
// 	if search, ok := filters["search"]; ok {
// 		query += " AND (p.name LIKE ? OR p.description LIKE ?)"
// 		searchTerm := "%" + search.(string) + "%"
// 		args = append(args, searchTerm, searchTerm)
// 	}

// 	query += " ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
// 	args = append(args, limit, offset)

// 	rows, err := s.db.Query(query, args...)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get all products: %w", err)
// 	}
// 	defer rows.Close()

// 	var products []*models.Product
// 	for rows.Next() {
// 		product := &models.Product{}
// 		seller := &models.User{}
// 		var imagesJSON, tagsJSON string

// 		err := rows.Scan(
// 			&product.ID, &product.Name, &product.Description, &product.Category,
// 			&product.Price, &product.Currency, &imagesJSON, &product.Status,
// 			&product.Stock, &product.MinOrder, &product.MaxOrder, &product.SellerID,
// 			&product.ChamaID, &product.County, &product.Town, &product.Address,
// 			&tagsJSON, &product.Rating, &product.TotalRatings, &product.TotalSales,
// 			&product.IsPromoted, &product.CreatedAt, &product.UpdatedAt,
// 			&seller.FirstName, &seller.LastName, &seller.Avatar,
// 		)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to scan product: %w", err)
// 		}

// 		// Parse JSON fields
// 		if err := product.SetImagesFromJSON(imagesJSON); err != nil {
// 			return nil, fmt.Errorf("failed to parse images: %w", err)
// 		}
// 		if err := product.SetTagsFromJSON(tagsJSON); err != nil {
// 			return nil, fmt.Errorf("failed to parse tags: %w", err)
// 		}

// 		// Set seller information
// 		seller.ID = product.SellerID
// 		product.Seller = seller

// 		products = append(products, product)
// 	}

// 	return products, nil
// }

// // Category represents a product category with count
// type Category struct {
// 	ID           string `json:"id"`
// 	Name         string `json:"name"`
// 	Icon         string `json:"icon"`
// 	ProductCount int    `json:"productCount"`
// }

// // GetCategoriesWithCounts returns all product categories with their product counts
// func (s *MarketplaceService) GetCategoriesWithCounts() ([]Category, error) {
// 	// Query to get categories with product counts
// 	query := `
// 		SELECT
// 			category,
// 			COUNT(*) as product_count
// 		FROM products
// 		WHERE status = 'active' AND stock > 0
// 		GROUP BY category
// 		ORDER BY category ASC
// 	`

// 	rows, err := s.db.Query(query)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query categories: %w", err)
// 	}
// 	defer rows.Close()

// 	var categories []Category

// 	// Map category IDs to display names and icons
// 	categoryMap := map[string]struct {
// 		name string
// 		icon string
// 	}{
// 		"agriculture":    {"Agriculture", "leaf"},
// 		"food_beverage":  {"Food & Beverage", "restaurant"},
// 		"clothing":       {"Clothing", "shirt"},
// 		"electronics":    {"Electronics", "phone-portrait"},
// 		"home_garden":    {"Home & Garden", "home"},
// 		"health_beauty":  {"Health & Beauty", "heart"},
// 		"sports_outdoors": {"Sports & Outdoors", "football"},
// 		"books_media":    {"Books & Media", "book"},
// 		"automotive":     {"Automotive", "car"},
// 		"services":       {"Services", "construct"},
// 		"crafts":         {"Crafts", "color-palette"},
// 		"beauty":         {"Beauty", "flower"},
// 		"other":          {"Other", "ellipsis-horizontal"},
// 	}

// 	for rows.Next() {
// 		var categoryID string
// 		var productCount int

// 		if err := rows.Scan(&categoryID, &productCount); err != nil {
// 			return nil, fmt.Errorf("failed to scan category row: %w", err)
// 		}

// 		// Get category display info
// 		categoryInfo, exists := categoryMap[categoryID]
// 		if !exists {
// 			// Default for unknown categories
// 			categoryInfo = struct {
// 				name string
// 				icon string
// 			}{
// 				name: strings.Title(strings.ReplaceAll(categoryID, "_", " ")),
// 				icon: "pricetag",
// 			}
// 		}

// 		category := Category{
// 			ID:           categoryID,
// 			Name:         categoryInfo.name,
// 			Icon:         categoryInfo.icon,
// 			ProductCount: productCount,
// 		}

// 		categories = append(categories, category)
// 	}

// 	if err := rows.Err(); err != nil {
// 		return nil, fmt.Errorf("error iterating category rows: %w", err)
// 	}

// 	// If no categories found, return default categories with 0 counts
// 	if len(categories) == 0 {
// 		defaultCategories := []Category{
// 			{"agriculture", "Agriculture", "leaf", 0},
// 			{"food_beverage", "Food & Beverage", "restaurant", 0},
// 			{"clothing", "Clothing", "shirt", 0},
// 			{"electronics", "Electronics", "phone-portrait", 0},
// 			{"home_garden", "Home & Garden", "home", 0},
// 			{"health_beauty", "Health & Beauty", "heart", 0},
// 			{"automotive", "Automotive", "car", 0},
// 			{"services", "Services", "construct", 0},
// 			{"other", "Other", "ellipsis-horizontal", 0},
// 		}
// 		return defaultCategories, nil
// 	}

// 	return categories, nil
// }
