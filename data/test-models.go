package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PostgresTestRepository struct {
	Conn *sql.DB
}

func NewPostgresTestRepository(db *sql.DB) *PostgresTestRepository {
	return &PostgresTestRepository{
		Conn: db,
	}
}

func (u *PostgresTestRepository) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, err := u.Conn.BeginTx(ctx, nil) // Begin a transaction with the provided context
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (u *PostgresTestRepository) AdminGetUsers(ctx context.Context) ([]*User, error) {

	users := []*User{}

	return users, nil
}

func (u *PostgresTestRepository) GetAllCategory(ctx context.Context) ([]*Category, error) {
	// make the query script

	categories := []*Category{
		{
			ID:          "5c120e64-6d34-4184-b1cb-b174e5a51a22",
			Name:        "Electronics & Gadgets",
			Description: "Devices and gadgets for rent, including phones, cameras, laptops, and more.",
			IconClass:   "fa-solid fa-laptop",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
		{
			ID:          "e8c1a446-0310-4b43-9462-5a7a31c4db1b",
			Name:        "Professional Services Equipment",
			Description: "Specialized equipment used for business, medical, or creative services.",
			IconClass:   "fa-solid fa-briefcase",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
	}

	return categories, nil

}
func (u *PostgresTestRepository) GetAllSubCategory(ctx context.Context) ([]*Subcategory, error) {

	subCategories := []*Subcategory{
		{
			ID:          "103f70cb-ff0a-40e4-918a-4132306af66c",
			CategoryId:  "d63173a6-3256-4058-9322-fbea8f32cc0f",
			Name:        "Pet Supplies",
			Description: "Rent pet crates, toys, and supplies.",
			IconClass:   "fa-solid fa-paw",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
		{
			ID:          "1bd9364a-c5ed-4d13-b32c-3dbab9f64972",
			CategoryId:  "9c47613d-0beb-4f46-91e0-14ad5fb88548",
			Name:        "Boats",
			Description: "Rent speedboats, yachts, and fishing boats.",
			IconClass:   "fa-solid fa-ship",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
	}

	return subCategories, nil
}
func (u *PostgresTestRepository) GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error) {

	subCategories := []*Subcategory{
		{
			ID:          "103f70cb-ff0a-40e4-918a-4132306af66c",
			CategoryId:  "d63173a6-3256-4058-9322-fbea8f32cc0f",
			Name:        "Pet Supplies",
			Description: "Rent pet crates, toys, and supplies.",
			IconClass:   "fa-solid fa-paw",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
		{
			ID:          "1bd9364a-c5ed-4d13-b32c-3dbab9f64972",
			CategoryId:  "9c47613d-0beb-4f46-91e0-14ad5fb88548",
			Name:        "Boats",
			Description: "Rent speedboats, yachts, and fishing boats.",
			IconClass:   "fa-solid fa-ship",
			UpdatedAt:   time.Now(),
			CreatedAt:   time.Now(),
		},
	}

	return subCategories, nil
}

func (u *PostgresTestRepository) GetcategoryByID(ctx context.Context, id string) (*Category, error) {

	category := Category{
		ID:          "ca3d802a-9cff-47e3-9081-fba46f451f70",
		Name:        "Glass Table",
		Description: "Nice turn table",
		IconClass:   "fa-solid fa-couch",
		UpdatedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	return &category, nil
}
func (u *PostgresTestRepository) GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error) {

	subCategory := Subcategory{
		ID:          "1bd9364a-c5ed-4d13-b32c-3dbab9f64972",
		CategoryId:  "9c47613d-0beb-4f46-91e0-14ad5fb88548",
		Name:        "Boats",
		Description: "Rent speedboats, yachts, and fishing boats.",
		IconClass:   "fa-solid fa-ship",
		UpdatedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	return &subCategory, nil
}

func (u *PostgresTestRepository) CreateInventory(tx *sql.Tx, ctx context.Context, name string, description string, userId string, categoryId string, subcategoryId string, urls []string) error {

	return nil
}

func (u *PostgresTestRepository) GetInventoryByID(ctx context.Context, id string) (*Inventory, error) {

	inventory := Inventory{
		ID:            "1f426485-e2ad-4f1b-839f-5714dea928ff",
		Name:          "shoes",
		Description:   "size 44 ladies wear",
		UserId:        "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		CategoryId:    "d63173a6-3256-4058-9322-fbea8f32cc0f",
		SubcategoryId: "a64501c2-bbf7-4943-bf47-3a20d727eec0",
		Promoted:      true,
		Deactivated:   false,
		UpdatedAt:     time.Now(),
		CreatedAt:     time.Now(),
	}

	return &inventory, nil
}

func (u *PostgresTestRepository) CreateInventoryRating(
	ctx context.Context,
	inventoryId string,
	raterId string,
	userId string,
	comment string,
	rating int32) (*InventoryRating, error) {

	inventoryRating := InventoryRating{
		ID:          "15abc220-967b-44cb-9e95-183b63571e88",
		InventoryId: "e933c064-2c82-46a4-8c76-bef9558001d8",
		UserId:      "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		RaterId:     "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		Rating:      5,
		Comment:     "the rental process was so succint",
		UpdatedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	return &inventoryRating, nil
}

func (u *PostgresTestRepository) CreateUserRating(
	ctx context.Context,
	userId string,
	rating int32,
	comment string,
	raterId string,
) (*UserRating, error) {

	userRating := UserRating{
		ID:        "6a7b83f0-30cb-4854-a32e-3576bf491858",
		UserId:    "827db1c7-cb77-4663-9153-9e1efc722eec",
		RaterId:   "7a937e9d-1dc2-4e6d-ba38-d1648b05730c",
		Rating:    5,
		Comment:   "nice response",
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	return &userRating, nil
}

func (u *PostgresTestRepository) GetUserByID(ctx context.Context, id string) (*User, error) {

	user := User{
		ID:        "01197718-a7a9-4af8-9870-661e17cd0d81",
		Email:     "amara@gmail.com",
		FirstName: "johnson",
		LastName:  "enyi",
		Verified:  true,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	return &user, nil
}
