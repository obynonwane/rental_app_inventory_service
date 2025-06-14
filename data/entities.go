package data

import (
	"time"
)

// User is the structure which holds one user from the database.
type User struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	FirstName   string      `json:"first_name,omitempty"`
	LastName    string      `json:"last_name,omitempty"`
	Phone       string      `json:"phone"`
	Password    string      `json:"-"`
	Verified    bool        `json:"verified"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Inventories []Inventory `json:"inventories,omitempty"` // One-to-many relationship
}

type Category struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	IconClass     string        `json:"icon_class"`
	CategorySlug  string        `json:"category_slug"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Subcategories []Subcategory `json:"subcategories"` // One-to-many relationship
	Inventories   []Inventory   `json:"inventories"`   // One-to-many relationship
}

type Subcategory struct {
	ID              string      `json:"id"`
	CategoryId      string      `json:"category_id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	IconClass       string      `json:"icon_class"`
	SubCategorySlug string      `json:"subcategory_slug"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Inventories     []Inventory `json:"inventories"` // One-to-many relationship
}

type Inventory struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	UserId        string    `json:"user_id"`
	CategoryId    string    `json:"category_id"`
	SubcategoryId string    `json:"subcategory_id"`
	Promoted      bool      `json:"promoted"`
	Deactivated   bool      `json:"deactivated"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CountryId     string    `json:"country_id"`
	StateId       string    `json:"state_id"`
	LgaId         string    `json:"lga_id"`
	Slug          string    `json:"slug"`
	Ulid          string    `json:"ulid"`
	OfferPrice    float64   `json:"offer_price"`

	StateSlug       string `json:"state_slug"`
	CountrySlug     string `json:"country_slug"`
	LgaSlug         string `json:"lga_slug"`
	CategorySlug    string `json:"category_slug"`
	SubcategorySlug string `json:"subcategory_slug"`

	ProductPurpose  string  `json:"product_purpose"`  // e.g., "sale" or "rental"
	Quantity        float64 `json:"quantity"`         // default to 1
	IsAvailable     string  `json:"is_available"`     // e.g., "yes" or "no"
	RentalDuration  string  `json:"rental_duration"`  // e.g., "hourly", "daily"
	SecurityDeposit float64 `json:"security_deposit"` // default to 0
	Tags            string  `json:"tags"`             // comma- or space-separated
	Metadata        string  `json:"metadata"`         // optional JSON string
	Negotiable      string  `json:"negotiable"`       // e.g., "yes" or "no"
	PrimaryImage    string  `json:"primary_image"`
	MinimumPrice    float64 `json:"minimum_price"`

	Images []InventoryImage `json:"images"` // One-to-many relationship
	User   User             `json:"user"`
	// Add the following:
	Country *Country `json:"country"`
	State   *State   `json:"state"`
	Lga     *Lga     `json:"lga"`
}

type InventoryImage struct {
	ID          string    `json:"id"`
	LiveUrl     string    `json:"live_url"`
	LocalUrl    string    `json:"local_url"`
	InventoryId string    `json:"inventory_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InventoryRating struct {
	ID           string                 `json:"id"`
	InventoryId  string                 `json:"inventory_id"`
	UserId       string                 `json:"user_id"`
	RaterId      string                 `json:"rater_id"`
	Rating       int32                  `json:"rating"`
	Comment      string                 `json:"comment"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedAt    time.Time              `json:"created_at"`
	RaterDetails User                   `json:"rater_details"`
	Replies      []InventoryRatingReply `json:"replies"`
}

type UserRating struct {
	ID           string    `json:"id"`
	UserId       string    `json:"user_id"`
	RaterId      string    `json:"rater_id"`
	Rating       int32     `json:"rating"`
	Comment      string    `json:"comment"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
	RaterDetails User      `json:"rater_details"`
}

type InventoryRatingReply struct {
	ID             string    `json:"id"`
	RatingID       string    `json:"rating_id"`
	ReplierID      string    `json:"replier_id"`
	ParentReplyID  *string   `json:"parent_reply_id"`
	Comment        string    `json:"comment"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
	ReplierDetails User      `json:"replier_details"`
}

type UserRatingReply struct {
	ID            string    `json:"id"`
	RatingID      string    `json:"rating_id"`
	ReplierID     string    `json:"replier_id"`
	ParentReplyID *string   `json:"parent_reply_id"`
	Comment       string    `json:"comment"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type Country struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type State struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StateSlug string    `json:"state_slug"`
	CountryID string    `json:"country_id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Lga struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	LgaSlug   string    `json:"lga_slug"`
	StateID   string    `json:"state_id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

type InventoryBooking struct {
	ID                string    `json:"id"`
	InventoryID       string    `json:"inventory_id"`
	RenterID          string    `json:"renter_id"`
	OwnerID           string    `json:"owner_id"`
	StartDate         time.Time `json:"start_date"`           // just the date part
	StartTime         *string   `json:"start_time,omitempty"` // optional, stored as string e.g. "15:04:05"
	EndDate           time.Time `json:"end_date"`
	EndTime           *string   `json:"end_time,omitempty"`
	OfferPricePerUnit float64   `json:"offer_price_per_unit"`
	TotalAmount       float64   `json:"total_amount"`
	SecurityDeposit   float64   `json:"security_deposit"`
	Quantity          float64   `json:"quantity"`
	Status            string    `json:"status"`
	PaymentStatus     string    `json:"payment_status"`
	RentalType        string    `json:"rental_type"` // e.g. hourly, daily
	RentalDuration    float64   `json:"rental_duration"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type InventorySale struct {
	ID                string    `json:"id"`
	InventoryID       string    `json:"inventory_id"`
	SellerID          string    `json:"seller_id"`
	BuyerID           *string   `json:"buyer_id,omitempty"` // Nullable
	OfferPricePerUnit float64   `json:"offer_price_per_unit"`
	Quantity          float64   `json:"quantity"`
	TotalAmount       float64   `json:"total_amount"`
	Status            string    `json:"status"`         // available, sold, cancelled
	PaymentStatus     string    `json:"payment_status"` // pending, paid, failed
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
