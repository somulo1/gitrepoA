package models

// import (
// 	"encoding/json"
// 	"time"
// )

// // ProductCategory represents product categories
// type ProductCategory string

// const (
// 	ProductCategoryAgriculture  ProductCategory = "agriculture"
// 	ProductCategoryFoodBeverage ProductCategory = "food_beverage"
// 	ProductCategoryClothing     ProductCategory = "clothing"
// 	ProductCategoryElectronics  ProductCategory = "electronics"
// 	ProductCategoryServices     ProductCategory = "services"
// 	ProductCategoryCrafts       ProductCategory = "crafts"
// 	ProductCategoryBeauty       ProductCategory = "beauty"
// 	ProductCategoryHomeGarden   ProductCategory = "home_garden"
// 	ProductCategoryAutomotive   ProductCategory = "automotive"
// 	ProductCategoryOther        ProductCategory = "other"
// )

// // ProductStatus represents product status
// type ProductStatus string

// const (
// 	ProductStatusActive       ProductStatus = "active"
// 	ProductStatusInactive     ProductStatus = "inactive"
// 	ProductStatusOutOfStock   ProductStatus = "out_of_stock"
// 	ProductStatusDiscontinued ProductStatus = "discontinued"
// )

// // OrderStatus represents order status
// type OrderStatus string

// const (
// 	OrderStatusPending    OrderStatus = "pending"
// 	OrderStatusConfirmed  OrderStatus = "confirmed"
// 	OrderStatusProcessing OrderStatus = "processing"
// 	OrderStatusShipped    OrderStatus = "shipped"
// 	OrderStatusDelivered  OrderStatus = "delivered"
// 	OrderStatusCancelled  OrderStatus = "cancelled"
// 	OrderStatusRefunded   OrderStatus = "refunded"
// )

// // DeliveryStatus represents delivery status
// type DeliveryStatus string

// const (
// 	DeliveryStatusPending   DeliveryStatus = "pending"
// 	DeliveryStatusAssigned  DeliveryStatus = "assigned"
// 	DeliveryStatusInTransit DeliveryStatus = "in_transit"
// 	DeliveryStatusDelivered DeliveryStatus = "delivered"
// 	DeliveryStatusFailed    DeliveryStatus = "failed"
// )

// // Product represents a product in the marketplace
// type Product struct {
// 	ID           string          `json:"id" db:"id"`
// 	Name         string          `json:"name" db:"name"`
// 	Description  string          `json:"description" db:"description"`
// 	Category     ProductCategory `json:"category" db:"category"`
// 	Price        float64         `json:"price" db:"price"`
// 	Currency     string          `json:"currency" db:"currency"`
// 	Images       []string        `json:"images" db:"images"`
// 	Status       ProductStatus   `json:"status" db:"status"`
// 	Stock        int             `json:"stock" db:"stock"`
// 	MinOrder     int             `json:"minOrder" db:"min_order"`
// 	MaxOrder     *int            `json:"maxOrder,omitempty" db:"max_order"`
// 	SellerID     string          `json:"sellerId" db:"seller_id"`
// 	ChamaID      *string         `json:"chamaId,omitempty" db:"chama_id"`
// 	County       string          `json:"county" db:"county"`
// 	Town         string          `json:"town" db:"town"`
// 	Address      *string         `json:"address,omitempty" db:"address"`
// 	Tags         []string        `json:"tags" db:"tags"`
// 	Rating       float64         `json:"rating" db:"rating"`
// 	TotalRatings int             `json:"totalRatings" db:"total_ratings"`
// 	TotalSales   int             `json:"totalSales" db:"total_sales"`
// 	IsPromoted   bool            `json:"isPromoted" db:"is_promoted"`
// 	CreatedAt    time.Time       `json:"createdAt" db:"created_at"`
// 	UpdatedAt    time.Time       `json:"updatedAt" db:"updated_at"`

// 	// Joined data (populated when needed)
// 	Seller *User  `json:"seller,omitempty"`
// 	Chama  *Chama `json:"chama,omitempty"`
// }

// // CartItem represents an item in a user's cart
// type CartItem struct {
// 	ID        string    `json:"id" db:"id"`
// 	UserID    string    `json:"userId" db:"user_id"`
// 	ProductID string    `json:"productId" db:"product_id"`
// 	Quantity  int       `json:"quantity" db:"quantity"`
// 	Price     float64   `json:"price" db:"price"`
// 	AddedAt   time.Time `json:"addedAt" db:"added_at"`

// 	// Joined data (populated when needed)
// 	Product *Product `json:"product,omitempty"`
// }

// // OrderItem represents an item within an order
// type OrderItem struct {
// 	ID        string  `json:"id" db:"id"`
// 	OrderID   string  `json:"orderId" db:"order_id"`
// 	ProductID string  `json:"productId" db:"product_id"`
// 	Quantity  int     `json:"quantity" db:"quantity"`
// 	Price     float64 `json:"price" db:"price"`
// 	Name      string  `json:"name" db:"name"`

// 	// Joined data (populated when needed)
// 	Product *Product `json:"product,omitempty"`
// }

// // Order represents an order in the marketplace
// type Order struct {
// 	ID                string         `json:"id" db:"id"`
// 	BuyerID           string         `json:"buyerId" db:"buyer_id"`
// 	SellerID          string         `json:"sellerId" db:"seller_id"`
// 	ChamaID           *string        `json:"chamaId,omitempty" db:"chama_id"`
// 	TotalAmount       float64        `json:"totalAmount" db:"total_amount"`
// 	Currency          string         `json:"currency" db:"currency"`
// 	Status            OrderStatus    `json:"status" db:"status"`
// 	PaymentMethod     string         `json:"paymentMethod" db:"payment_method"`
// 	PaymentStatus     string         `json:"paymentStatus" db:"payment_status"`
// 	DeliveryCounty    string         `json:"deliveryCounty" db:"delivery_county"`
// 	DeliveryTown      string         `json:"deliveryTown" db:"delivery_town"`
// 	DeliveryAddress   string         `json:"deliveryAddress" db:"delivery_address"`
// 	DeliveryPhone     string         `json:"deliveryPhone" db:"delivery_phone"`
// 	DeliveryFee       float64        `json:"deliveryFee" db:"delivery_fee"`
// 	DeliveryPersonID  *string        `json:"deliveryPersonId,omitempty" db:"delivery_person_id"`
// 	DeliveryStatus    DeliveryStatus `json:"deliveryStatus" db:"delivery_status"`
// 	EstimatedDelivery *time.Time     `json:"estimatedDelivery,omitempty" db:"estimated_delivery"`
// 	ActualDelivery    *time.Time     `json:"actualDelivery,omitempty" db:"actual_delivery"`
// 	Notes             *string        `json:"notes,omitempty" db:"notes"`
// 	CreatedAt         time.Time      `json:"createdAt" db:"created_at"`
// 	UpdatedAt         time.Time      `json:"updatedAt" db:"updated_at"`

// 	// Joined data (populated when needed)
// 	Buyer          *User       `json:"buyer,omitempty"`
// 	Seller         *User       `json:"seller,omitempty"`
// 	Chama          *Chama      `json:"chama,omitempty"`
// 	DeliveryPerson *User       `json:"deliveryPerson,omitempty"`
// 	Items          []OrderItem `json:"items,omitempty"`
// }

// // ProductReview represents a review for a product
// type ProductReview struct {
// 	ID                 string    `json:"id" db:"id"`
// 	ProductID          string    `json:"productId" db:"product_id"`
// 	OrderID            string    `json:"orderId" db:"order_id"`
// 	ReviewerID         string    `json:"reviewerId" db:"reviewer_id"`
// 	Rating             int       `json:"rating" db:"rating"`
// 	Comment            *string   `json:"comment,omitempty" db:"comment"`
// 	Images             []string  `json:"images" db:"images"`
// 	IsVerifiedPurchase bool      `json:"isVerifiedPurchase" db:"is_verified_purchase"`
// 	CreatedAt          time.Time `json:"createdAt" db:"created_at"`

// 	// Joined data (populated when needed)
// 	Reviewer *User    `json:"reviewer,omitempty"`
// 	Product  *Product `json:"product,omitempty"`
// 	Order    *Order   `json:"order,omitempty"`
// }

// // ProductCreation represents data for creating a new product
// type ProductCreation struct {
// 	Name        string          `json:"name" validate:"required"`
// 	Description string          `json:"description" validate:"required"`
// 	Category    ProductCategory `json:"category" validate:"required"`
// 	Price       float64         `json:"price" validate:"required,gt=0"`
// 	Images      []string        `json:"images" validate:"required,min=1"`
// 	Stock       int             `json:"stock" validate:"required,gte=0"`
// 	MinOrder    int             `json:"minOrder" validate:"required,gt=0"`
// 	MaxOrder    *int            `json:"maxOrder,omitempty"`
// 	County      string          `json:"county" validate:"required"`
// 	Town        string          `json:"town" validate:"required"`
// 	Address     *string         `json:"address,omitempty"`
// 	Tags        []string        `json:"tags"`
// }

// // ProductUpdate represents data for updating a product
// type ProductUpdate struct {
// 	Name        *string          `json:"name,omitempty"`
// 	Description *string          `json:"description,omitempty"`
// 	Category    *ProductCategory `json:"category,omitempty"`
// 	Price       *float64         `json:"price,omitempty"`
// 	Images      []string         `json:"images,omitempty"`
// 	Status      *ProductStatus   `json:"status,omitempty"`
// 	Stock       *int             `json:"stock,omitempty"`
// 	MinOrder    *int             `json:"minOrder,omitempty"`
// 	MaxOrder    *int             `json:"maxOrder,omitempty"`
// 	County      *string          `json:"county,omitempty"`
// 	Town        *string          `json:"town,omitempty"`
// 	Address     *string          `json:"address,omitempty"`
// 	Tags        []string         `json:"tags,omitempty"`
// }

// // OrderCreation represents data for creating a new order
// type OrderCreation struct {
// 	Items           []CartItem `json:"items" validate:"required,min=1"`
// 	PaymentMethod   string     `json:"paymentMethod" validate:"required"`
// 	DeliveryCounty  string     `json:"deliveryCounty" validate:"required"`
// 	DeliveryTown    string     `json:"deliveryTown" validate:"required"`
// 	DeliveryAddress string     `json:"deliveryAddress" validate:"required"`
// 	DeliveryPhone   string     `json:"deliveryPhone" validate:"required"`
// 	Notes           *string    `json:"notes,omitempty"`
// }

// // IsAvailable checks if the product is available for purchase
// func (p *Product) IsAvailable() bool {
// 	return p.Status == ProductStatusActive && p.Stock > 0
// }

// // IsInStock checks if the product has sufficient stock
// func (p *Product) IsInStock(quantity int) bool {
// 	return p.Stock >= quantity
// }

// // CanOrder checks if the quantity is within order limits
// func (p *Product) CanOrder(quantity int) bool {
// 	if quantity < p.MinOrder {
// 		return false
// 	}
// 	if p.MaxOrder != nil && quantity > *p.MaxOrder {
// 		return false
// 	}
// 	return true
// }

// // GetLocation returns the product's location as a formatted string
// func (p *Product) GetLocation() string {
// 	return p.Town + ", " + p.County
// }

// // GetImagesJSON returns images as JSON string for database storage
// func (p *Product) GetImagesJSON() (string, error) {
// 	if len(p.Images) == 0 {
// 		return "[]", nil
// 	}
// 	data, err := json.Marshal(p.Images)
// 	return string(data), err
// }

// // SetImagesFromJSON sets images from JSON string
// func (p *Product) SetImagesFromJSON(imagesJSON string) error {
// 	if imagesJSON == "" {
// 		p.Images = []string{}
// 		return nil
// 	}
// 	return json.Unmarshal([]byte(imagesJSON), &p.Images)
// }

// // GetTagsJSON returns tags as JSON string for database storage
// func (p *Product) GetTagsJSON() (string, error) {
// 	if len(p.Tags) == 0 {
// 		return "[]", nil
// 	}
// 	data, err := json.Marshal(p.Tags)
// 	return string(data), err
// }

// // SetTagsFromJSON sets tags from JSON string
// func (p *Product) SetTagsFromJSON(tagsJSON string) error {
// 	if tagsJSON == "" {
// 		p.Tags = []string{}
// 		return nil
// 	}
// 	return json.Unmarshal([]byte(tagsJSON), &p.Tags)
// }

// // GetTotalPrice returns the total price for the cart item
// func (ci *CartItem) GetTotalPrice() float64 {
// 	return ci.Price * float64(ci.Quantity)
// }

// // GetTotalPrice returns the total price for the order item
// func (oi *OrderItem) GetTotalPrice() float64 {
// 	return oi.Price * float64(oi.Quantity)
// }

// // GetTotalItems returns the total number of items in the order
// func (o *Order) GetTotalItems() int {
// 	total := 0
// 	for _, item := range o.Items {
// 		total += item.Quantity
// 	}
// 	return total
// }

// // GetGrandTotal returns the total amount including delivery fee
// func (o *Order) GetGrandTotal() float64 {
// 	return o.TotalAmount + o.DeliveryFee
// }

// // IsCompleted checks if the order is completed
// func (o *Order) IsCompleted() bool {
// 	return o.Status == OrderStatusDelivered
// }

// // IsCancelled checks if the order is cancelled
// func (o *Order) IsCancelled() bool {
// 	return o.Status == OrderStatusCancelled
// }

// // CanCancel checks if the order can be cancelled
// func (o *Order) CanCancel() bool {
// 	return o.Status == OrderStatusPending || o.Status == OrderStatusConfirmed
// }

// // CanRate checks if the order can be rated (delivered and not yet rated)
// func (o *Order) CanRate() bool {
// 	return o.Status == OrderStatusDelivered
// }

// // GetDeliveryLocation returns the delivery location as a formatted string
// func (o *Order) GetDeliveryLocation() string {
// 	return o.DeliveryTown + ", " + o.DeliveryCounty
// }

// // IsValidRating checks if the rating is valid (1-5)
// func (pr *ProductReview) IsValidRating() bool {
// 	return pr.Rating >= 1 && pr.Rating <= 5
// }

// // GetImagesJSON returns images as JSON string for database storage
// func (pr *ProductReview) GetImagesJSON() (string, error) {
// 	if len(pr.Images) == 0 {
// 		return "[]", nil
// 	}
// 	data, err := json.Marshal(pr.Images)
// 	return string(data), err
// }

// // SetImagesFromJSON sets images from JSON string
// func (pr *ProductReview) SetImagesFromJSON(imagesJSON string) error {
// 	if imagesJSON == "" {
// 		pr.Images = []string{}
// 		return nil
// 	}
// 	return json.Unmarshal([]byte(imagesJSON), &pr.Images)
// }

// // Review represents a product review (alias for ProductReview for consistency)
// type Review = ProductReview

// // WishlistItem represents an item in user's wishlist
// type WishlistItem struct {
// 	ID        string    `json:"id" db:"id"`
// 	UserID    string    `json:"userId" db:"user_id"`
// 	ProductID string    `json:"productId" db:"product_id"`
// 	AddedAt   time.Time `json:"addedAt" db:"added_at"`

// 	// Relationships
// 	User    *User    `json:"user,omitempty"`
// 	Product *Product `json:"product,omitempty"`
// }
