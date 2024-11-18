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

	return &category, nil
}
