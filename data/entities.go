package data

import (
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

// User is the structure which holds one user from the database.
type User struct {
	ID          string                  `json:"id"`
	Email       string                  `json:"email"`
	FirstName   string                  `json:"first_name,omitempty"`
	LastName    string                  `json:"last_name,omitempty"`
	Phone       string                  `json:"phone"`
	Password    string                  `json:"-"`
	ProfileImg  *wrapperspb.StringValue `json:"profile_img"`
	Verified    bool                    `json:"verified"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	UserSlug    string                  `json:"user_slug"`
	Inventories []Inventory             `json:"inventories,omitempty"` // One-to-many relationship
}

type Category struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	IconClass      string        `json:"icon_class"`
	CategorySlug   string        `json:"category_slug"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Subcategories  []Subcategory `json:"subcategories"` // One-to-many relationship
	Inventories    []Inventory   `json:"inventories"`   // One-to-many relationship
	InventoryCount int32         `json:"inventory_count"`
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
	InventoryCount  int32       `json:"inventory_count"`
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

	ProductPurpose  string                  `json:"product_purpose"`  // e.g., "sale" or "rental"
	Quantity        float64                 `json:"quantity"`         // default to 1
	IsAvailable     string                  `json:"is_available"`     // e.g., "yes" or "no"
	RentalDuration  string                  `json:"rental_duration"`  // e.g., "hourly", "daily"
	SecurityDeposit float64                 `json:"security_deposit"` // default to 0
	Tags            *wrapperspb.StringValue `json:"tags"`             // comma- or space-separated
	Metadata        string                  `json:"metadata"`         // optional JSON string
	Negotiable      string                  `json:"negotiable"`       // e.g., "yes" or "no"
	PrimaryImage    string                  `json:"primary_image"`
	MinimumPrice    float64                 `json:"minimum_price"`
	UsageGuide      *wrapperspb.StringValue `json:"usage_guide"`
	Condition       *wrapperspb.StringValue `json:"condition"`
	Included        *wrapperspb.StringValue `json:"included"`

	Images []InventoryImage `json:"images"` // One-to-many relationship
	User   User             `json:"user"`
	// Add the following:
	Country *Country `json:"country"`
	State   *State   `json:"state"`
	Lga     *Lga     `json:"lga"`

	AverageRating *float64          `json:"average_rating"` // computed sum
	TotalRatings  *int32            `json:"total_ratings"`
	UserVerified  *bool             `json:"user_verified"`
	Ratings       []InventoryRating `json:"ratings"`
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

type Chat struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	SenderID    string    `json:"sender"`              // filled in by the server
	ReceiverID  string    `json:"receiver"`            // who should get it
	ImageUrl    *string   `json:"image_url,omitempty"` // optional, stored as string e.g. "15:04:05"
	SentAt      int64     `json:"sent_at"`             // unix millis
	Type        *string   `json:"type"`
	ContentType *string   `json:"content_type"`
	IsRead      bool      `json:"is_read"`
	ReplyTo     *string   `json:"reply_to_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BusinessKyc struct {
	ID                 string  `json:"id"`
	Address            string  `json:"address"`
	CacNumber          *string `json:"cac_number,omitempty"`
	DisplayName        string  `json:"display_name"`
	Description        string  `json:"description"`
	KeyBonus           string  `json:"key_bonus"`
	BusinessRegistered string  `json:"business_registered"` // e.g., "YES" or "NO"

	UserID string `json:"user_id"`
	User   *User  `json:"user,omitempty"`

	CountryID string   `json:"country_id"`
	Country   *Country `json:"country,omitempty"`

	StateID string `json:"state_id"`
	State   *State `json:"state,omitempty"`

	LgaID string `json:"lga_id"`
	Lga   *Lga   `json:"lga,omitempty"`

	PlanID string `json:"plan_id"`
	Plan   *Plan  `json:"plan,omitempty"`

	Verified   bool `json:"verified"`
	ActivePlan bool `json:"active_plan"`

	ShopBanner string    `json:"shop_banner"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type Plan struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	MonthlyPrice float64   `json:"monthly_price"`
	AnnualPrice  float64   `json:"annual_price"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SavedInventory struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	InventoryID string    `json:"inventory_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}
