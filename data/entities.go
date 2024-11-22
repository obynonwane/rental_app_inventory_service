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
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Subcategories []Subcategory `json:"subcategories"` // One-to-many relationship
	Inventories   []Inventory   `json:"inventories"`   // One-to-many relationship
}

type Subcategory struct {
	ID          string      `json:"id"`
	CategoryId  string      `json:"category_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	IconClass   string      `json:"icon_class"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Inventories []Inventory `json:"inventories"` // One-to-many relationship
}

type Inventory struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	UserId        string           `json:"user_id"`
	CategoryId    string           `json:"category_id"`
	SubcategoryId string           `json:"subcategory_id"`
	Promoted      bool             `json:"promoted"`
	Deactivated   bool             `json:"deactivated"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Images        []InventoryImage `json:"images"` // One-to-many relationship
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
	ID          string    `json:"id"`
	InventoryId string    `json:"inventory_id"`
	UserId      string    `json:"user_id"`
	RaterId     string    `json:"rater_id"`
	Rating      string    `json:"rating"`
	Comment     string    `json:"comment"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserRating struct {
	ID          string    `json:"id"`
	UserId      string    `json:"user_id"`
	RaterId     string    `json:"rater_id"`
	Rating      string    `json:"rating"`
	Comment     string    `json:"comment"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}
