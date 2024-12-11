package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// db timeout period
const dbTimeout = time.Second * 3

// data of sqlDB type here connections to DB will live
var db *sql.DB

type PostgresRepository struct {
	Conn *sql.DB
}

// new instance of the PostgresRepository struct
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		Conn: db,
	}
}

func (p *PostgresRepository) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, err := p.Conn.BeginTx(ctx, nil) // Begin a transaction with the provided context
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (u *PostgresRepository) GetAll(ctx context.Context) ([]*User, error) {

	query := `SELECT id, email, first_name, last_name, password, verified, updated_at, created_at FROM users`

	rows, err := u.Conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User

	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.FirstName,
			&user.LastName,
			&user.Password,
			&user.Verified,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			log.Println("Error scanning", err)
			return nil, err
		}

		users = append(users, &user)
	}

	return users, nil
}

func (u *PostgresRepository) GetAllCategory(ctx context.Context) ([]*Category, error) {
	// make the query script
	query := `SELECT id, name, description, icon_class, updated_at, created_at FROM categories`

	rows, err := u.Conn.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var categories []*Category

	for rows.Next() {
		var category Category
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Description,
			&category.IconClass,
			&category.UpdatedAt,
			&category.CreatedAt,
		)

		if err != nil {
			log.Println("Error scanning", err)
		}

		categories = append(categories, &category)

	}

	return categories, nil

}
func (u *PostgresRepository) GetAllSubCategory(ctx context.Context) ([]*Subcategory, error) {
	// make the query script
	query := `SELECT id, category_id, name, description, icon_class, updated_at, created_at FROM subcategories`

	rows, err := u.Conn.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subCategories []*Subcategory

	for rows.Next() {
		var subCategory Subcategory
		err := rows.Scan(
			&subCategory.ID,
			&subCategory.CategoryId,
			&subCategory.Name,
			&subCategory.Description,
			&subCategory.IconClass,
			&subCategory.UpdatedAt,
			&subCategory.CreatedAt,
		)

		if err != nil {
			log.Println("Error scanning", err)
		}

		subCategories = append(subCategories, &subCategory)

	}

	return subCategories, nil
}
func (u *PostgresRepository) GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error) {
	// make the query script
	query := `SELECT id, category_id, name, description, icon_class, updated_at, created_at FROM subcategories where category_id = $1`

	rows, err := u.Conn.QueryContext(ctx, query, id)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var subCategories []*Subcategory

	for rows.Next() {
		var subCategory Subcategory
		err := rows.Scan(
			&subCategory.ID,
			&subCategory.CategoryId,
			&subCategory.Name,
			&subCategory.Description,
			&subCategory.IconClass,
			&subCategory.UpdatedAt,
			&subCategory.CreatedAt,
		)

		if err != nil {
			log.Println("Error scanning", err)
		}

		subCategories = append(subCategories, &subCategory)

	}

	return subCategories, nil
}

func (u *PostgresRepository) GetcategoryByID(ctx context.Context, id string) (*Category, error) {

	start := time.Now()

	// query to select
	query := `SELECT id, name, description, icon_class, updated_at, created_at FROM categories WHERE id = $1`

	row := u.Conn.QueryRowContext(ctx, query, id)

	var category Category

	err := row.Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.IconClass,
		&category.UpdatedAt,
		&category.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Handle case where no category is found for the given ID
			return nil, fmt.Errorf("no category found with ID %s", id)
		}
		// Handle other possible errors
		return nil, fmt.Errorf("error retrieving category by ID: %w", err)
	}

	log.Printf("GetcategoryByID took %s", time.Since(start))

	return &category, nil
}
func (u *PostgresRepository) GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error) {
	start := time.Now()
	log.Println("Inside Get subcategory Query")
	// query to select
	query := `SELECT id, category_id, name, description, icon_class, updated_at, created_at FROM subcategories WHERE id = $1`

	row := u.Conn.QueryRowContext(ctx, query, id)

	var subCategory Subcategory

	err := row.Scan(
		&subCategory.ID,
		&subCategory.CategoryId,
		&subCategory.Name,
		&subCategory.Description,
		&subCategory.IconClass,
		&subCategory.UpdatedAt,
		&subCategory.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Handle case where no category is found for the given ID
			return nil, fmt.Errorf("no subcategory found with ID %s", id)
		}
		// Handle other possible errors
		return nil, fmt.Errorf("error retrieving subcategory by ID: %w", err)
	}

	log.Printf("SubGetcategoryByID took %s", time.Since(start))
	return &subCategory, nil
}

func (u *PostgresRepository) CreateInventory(tx *sql.Tx, ctx context.Context, name string, description string, userId string, categoryId string, subcategoryId string, urls []string) error {

	query := `INSERT INTO inventories (name, description, user_id, category_id, subcategory_id, updated_at, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
			RETURNING id, name, description, user_id, category_id, subcategory_id, updated_at, created_at`

	var inventory Inventory
	err := tx.QueryRowContext(ctx, query, name, description, userId, categoryId, subcategoryId).Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	// Insert image URLs into a separate table
	for _, url := range urls {
		imageQuery := `
				INSERT INTO inventory_images (live_url, inventory_id, updated_at, created_at) 
				VALUES ($1, $2, NOW(), NOW())`
		_, err := tx.ExecContext(ctx, imageQuery, url, inventory.ID)
		if err != nil {
			return fmt.Errorf("failed to insert image URL: %w", err)
		}
	}

	return nil
}

func (u *PostgresRepository) GetInventoryByID(ctx context.Context, id string) (*Inventory, error) {

	query := `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at FROM inventories WHERE id = $1`
	row := u.Conn.QueryRowContext(ctx, query, id)

	var inventory Inventory

	err := row.Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.Promoted,
		&inventory.Deactivated,
		&inventory.UpdatedAt, // Ensure the order matches the query
		&inventory.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving inventory by ID: %w", err)
	}

	return &inventory, nil
}

func (u *PostgresRepository) CreateInventoryRating(
	ctx context.Context,
	inventoryId string,
	raterId string,
	userId string,
	comment string,
	rating int32) (*InventoryRating, error) {

	query := `INSERT INTO inventory_ratings (inventory_id, user_id, rater_id, rating, comment, updated_at, created_at)
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) 
	RETURNING id, inventory_id, user_id, rater_id, rating, comment, updated_at, created_at`

	var inventoryRating InventoryRating
	err := u.Conn.QueryRowContext(ctx, query, inventoryId, userId, raterId, rating, comment).Scan(
		&inventoryRating.ID,
		&inventoryRating.InventoryId,
		&inventoryRating.UserId,
		&inventoryRating.RaterId,
		&inventoryRating.Rating,
		&inventoryRating.Comment,
		&inventoryRating.UpdatedAt,
		&inventoryRating.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory rating: %w", err)
	}

	return &inventoryRating, nil
}

func (u *PostgresRepository) CreateUserRating(
	ctx context.Context,
	userId string,
	rating int32,
	comment string,
	raterId string,
) (*UserRating, error) {

	query := `INSERT INTO user_ratings (user_id, rater_id, rating, comment, updated_at, created_at)
	VALUES ($1, $2, $3, $4, NOW(), NOW()) 
	RETURNING id, user_id, rater_id, rating, comment, updated_at, created_at`

	var userRating UserRating
	err := u.Conn.QueryRowContext(ctx, query, userId, raterId, rating, comment).Scan(
		&userRating.ID,
		&userRating.UserId,
		&userRating.RaterId,
		&userRating.Rating,
		&userRating.Comment,
		&userRating.UpdatedAt,
		&userRating.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user rating: %w", err)
	}

	return &userRating, nil
}

func (u *PostgresRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, email, first_name, last_name, phone, verified, updated_at, created_at FROM users WHERE id = $1`
	row := u.Conn.QueryRowContext(ctx, query, id)

	var user User

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Verified,
		&user.UpdatedAt,
		&user.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving user by ID: %w", err)
	}

	log.Println(user, "the user is here")

	return &user, nil
}

func (u *PostgresRepository) GetInventoryRatings(ctx context.Context, id string) ([]*InventoryRating, error) {

	query := `SELECT id, inventory_id, user_id, rater_id, rating, comment, updated_at, created_at FROM inventory_ratings where id = $1`

	rows, err := u.Conn.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ratings []*InventoryRating

	for rows.Next() {
		var rating InventoryRating
		err := rows.Scan(
			&rating.ID,
			&rating.InventoryId,
			&rating.UserId,
			&rating.RaterId,
			&rating.Rating,
			&rating.Comment,
			&rating.UpdatedAt,
			&rating.CreatedAt,
		)
		if err != nil {
			log.Println("Error scanning", err)
			return nil, err
		}

		ratings = append(ratings, &rating)
	}

	return ratings, nil
}

func (u *PostgresRepository) GetUserRatings(ctx context.Context, id string) ([]*UserRating, error) {

	query := `SELECT id, user_id, rater_id, rating, comment, updated_at, created_at FROM user_ratings where user_id = $1`

	rows, err := u.Conn.QueryContext(ctx, query, id)
	if err != nil {
		log.Println(err, "ERROR")
		return nil, err
	}
	defer rows.Close()

	var ratings []*UserRating

	for rows.Next() {
		var rating UserRating
		err := rows.Scan(
			&rating.ID,
			&rating.UserId,
			&rating.RaterId,
			&rating.Rating,
			&rating.Comment,
			&rating.UpdatedAt,
			&rating.CreatedAt,
		)
		if err != nil {
			log.Println("Error scanning", err)
			return nil, err
		}

		ratings = append(ratings, &rating)
	}

	return ratings, nil
}
