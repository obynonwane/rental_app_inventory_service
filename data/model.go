package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lib/pq"
	"github.com/obynonwane/rental-service-proto/inventory"
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
	query := `
		SELECT
			c.id, c.name, c.description, c.icon_class, c.category_slug, c.created_at, c.updated_at,
			s.id, s.name, s.description, s.icon_class, s.subcategory_slug, s.created_at, s.updated_at, s.category_id
		FROM categories c
		LEFT JOIN subcategories s ON c.id = s.category_id
		ORDER BY c.name ASC, s.name ASC
	`

	rows, err := u.Conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categoryMap := make(map[string]*Category)

	for rows.Next() {
		var (
			cat                        Category
			sub                        Subcategory
			subID, subName             sql.NullString
			subDesc, subIcon, subSlug  sql.NullString
			subCreatedAt, subUpdatedAt sql.NullTime
			subCatID                   sql.NullString
		)

		err := rows.Scan(
			&cat.ID,
			&cat.Name,
			&cat.Description,
			&cat.IconClass,
			&cat.CategorySlug,
			&cat.CreatedAt,
			&cat.UpdatedAt,
			&subID,
			&subName,
			&subDesc,
			&subIcon,
			&subSlug,
			&subCreatedAt,
			&subUpdatedAt,
			&subCatID,
		)

		if err != nil {
			log.Println("Error scanning row:", err)
			continue
		}

		// Check if the category already exist
		existing, exists := categoryMap[cat.ID]
		if !exists {
			existing = &cat
			categoryMap[cat.ID] = existing
		}

		// Append subcategory if it exists
		if subID.Valid {
			sub.ID = subID.String
			sub.Name = subName.String
			sub.Description = subDesc.String
			sub.IconClass = subIcon.String
			sub.SubCategorySlug = subSlug.String
			sub.CreatedAt = subCreatedAt.Time
			sub.UpdatedAt = subUpdatedAt.Time
			sub.CategoryId = subCatID.String

			existing.Subcategories = append(existing.Subcategories, sub)
		}
	}

	// Convert map to slice
	var categories []*Category
	for _, c := range categoryMap {
		categories = append(categories, c)
	}

	// Sort by name (or any other deterministic field)
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	return categories, nil

}

func (u *PostgresRepository) GetAllSubCategory(ctx context.Context) ([]*Subcategory, error) {
	// make the query script
	query := `SELECT id, category_id, name, description, icon_class, subcategory_slug, updated_at, created_at FROM subcategories`

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
			&subCategory.SubCategorySlug,
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
	query := `SELECT id, category_id, name, description, icon_class, subcategory_slug, updated_at, created_at FROM subcategories where category_id = $1`

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
			&subCategory.SubCategorySlug,
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

func (u *PostgresRepository) GetCategoryByID(ctx context.Context, p *GetCategoryByIDPayload) (*Category, error) {

	var (
		query  string
		args   []interface{}
		filter []string
	)

	if p.CategoryID != "" {
		args = append(args, p.CategoryID)
		filter = append(filter, fmt.Sprintf("id = $%d", len(args)))
	}
	if p.CategorySlug != "" {
		args = append(args, p.CategorySlug)
		filter = append(filter, fmt.Sprintf("category_slug = $%d", len(args)))
	}

	if len(filter) == 0 {
		return nil, fmt.Errorf("no identifier provided to search category")
	}

	query = fmt.Sprintf(`
		SELECT id, name, description, icon_class, category_slug, updated_at, created_at
		FROM categories
		WHERE %s
		LIMIT 1`, strings.Join(filter, " AND "),
	)

	row := u.Conn.QueryRowContext(ctx, query, args...)

	var category Category
	err := row.Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.IconClass,
		&category.CategorySlug,
		&category.UpdatedAt,
		&category.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no category found with the provided identifier(s)")
		}
		return nil, fmt.Errorf("error retrieving category: %w", err)
	}

	return &category, nil
}

func (u *PostgresRepository) GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error) {

	// query to select
	query := `SELECT id, category_id, name, description, icon_class, subcategory_slug, updated_at, created_at FROM subcategories WHERE id = $1`

	row := u.Conn.QueryRowContext(ctx, query, id)

	var subCategory Subcategory

	err := row.Scan(
		&subCategory.ID,
		&subCategory.CategoryId,
		&subCategory.Name,
		&subCategory.Description,
		&subCategory.IconClass,
		&subCategory.SubCategorySlug,
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

	return &subCategory, nil
}

type CreateInventoryParams struct {
	Tx              *sql.Tx
	Ctx             context.Context
	Name            string
	Description     string
	UserID          string
	CategoryID      string
	SubcategoryID   string
	CountryID       string
	StateID         string
	LgaID           string
	Slug            string
	ULID            string
	StateSlug       string
	CountrySlug     string
	LgaSlug         string
	CategorySlug    string
	SubcategorySlug string
	OfferPrice      float64
	MinimumPrice    float64
	URLs            []string

	ProductPurpose  string
	Quantity        float64
	IsAvailable     string
	RentalDuration  string
	SecurityDeposit float64
	Tags            string
	Metadata        string
	Negotiable      string
	PrimaryImage    string
}

func (u *PostgresRepository) CreateInventory(req *CreateInventoryParams) error {

	log.Printf("%v", req)

	tx := req.Tx
	ctx := req.Ctx
	name := req.Name
	description := req.Description
	userId := req.UserID
	categoryId := req.CategoryID
	subcategoryId := req.SubcategoryID
	countryId := req.CountryID
	stateId := req.StateID
	lgaId := req.LgaID
	slug := req.Slug
	ulid := req.ULID
	offerPrice := req.OfferPrice
	stateSlug := req.StateSlug
	lgaSlug := req.LgaSlug
	countrySlug := req.CountrySlug
	categorySlug := req.CategorySlug
	subcategorySlug := req.SubcategorySlug
	urls := req.URLs

	productPurpose := req.ProductPurpose
	quantity := req.Quantity
	isAvailable := req.IsAvailable
	rentalDuration := req.RentalDuration
	securityDeposit := req.SecurityDeposit
	tags := req.Tags
	metadata := req.Metadata
	negotiable := req.Negotiable
	primaryImage := req.PrimaryImage
	minimumPrice := req.MinimumPrice

	query := `INSERT INTO inventories (
				name, 
				description, 
				user_id, 
				category_id, 
				subcategory_id, 
				country_id, 
				state_id, 
				lga_id, 
				slug, 
				ulid, 
				offer_price, 
				state_slug, 
				lga_slug, 
				country_slug, 
				category_slug, 
				subcategory_slug, 

				product_purpose,
				quantity,
				is_available,
				rental_duration,
				security_deposit,
				tags,
				metadata,
				negotiable,
				primary_image,
				minimum_price,

				updated_at, 
				created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8,$9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, NOW(), NOW()) 
			RETURNING 
				id, 
				name, 
				description, 
				user_id, 
				category_id, 
				subcategory_id, 
				country_id, 
				state_id, 
				lga_id, 
				slug, 
				ulid, 
				offer_price, 
				state_slug, 
				lga_slug, 
				country_slug, 
				category_slug, 
				subcategory_slug, 

				product_purpose,
				quantity,
				is_available,
				rental_duration,
				security_deposit,
				metadata,
				negotiable,
				primary_image,
				minimum_price,

				updated_at, 
				created_at`

	var inventory Inventory
	err := tx.QueryRowContext(ctx,
		query,
		name,
		description,
		userId,
		categoryId,
		subcategoryId,
		countryId,
		stateId,
		lgaId,
		slug,
		ulid,
		offerPrice,
		stateSlug,
		lgaSlug,
		countrySlug,
		categorySlug,
		subcategorySlug,
		productPurpose,
		quantity,
		isAvailable,
		rentalDuration,
		securityDeposit,
		tags,
		metadata,
		negotiable,
		primaryImage,
		minimumPrice,
	).Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.CountryId,
		&inventory.StateId,
		&inventory.LgaId,
		&inventory.Slug,
		&inventory.Ulid,
		&inventory.OfferPrice,
		&inventory.StateSlug,
		&inventory.LgaSlug,
		&inventory.CountrySlug,
		&inventory.CategorySlug,
		&inventory.SubcategorySlug,

		&inventory.ProductPurpose,
		&inventory.Quantity,
		&inventory.IsAvailable,
		&inventory.RentalDuration,
		&inventory.SecurityDeposit,
		&inventory.Metadata,
		&inventory.Negotiable,
		&inventory.PrimaryImage,
		&inventory.MinimumPrice,

		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	// Insert image URLs into a separate table
	for _, url := range urls {
		imageQuery := `
				INSERT INTO inventory_images (live_url, local_url, inventory_id, updated_at, created_at) 
				VALUES ($1, $2, $3, NOW(), NOW())`
		_, err := tx.ExecContext(ctx, imageQuery, url, url, inventory.ID)
		if err != nil {
			return fmt.Errorf("failed to insert image URL: %w", err)
		}
	}

	return nil
}

// func (u *PostgresRepository) GetInventoryByID(ctx context.Context, id string) (*Inventory, error) {

// 	query := `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
// 				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
// 				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image
// 		         FROM inventories
// 		         WHERE id = $1`
// 	row := u.Conn.QueryRowContext(ctx, query, id)

// 	var inventory Inventory

// 	err := row.Scan(
// 		&inventory.ID,
// 		&inventory.Name,
// 		&inventory.Description,
// 		&inventory.UserId,
// 		&inventory.CategoryId,
// 		&inventory.SubcategoryId,
// 		&inventory.Promoted,
// 		&inventory.Deactivated,
// 		&inventory.UpdatedAt, // Ensure the order matches the query
// 		&inventory.CreatedAt,
// 	)

// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, fmt.Errorf("no inventory found with ID %s", id)
// 		}
// 		return nil, fmt.Errorf("error retrieving inventory by ID: %w", err)
// 	}

// 	return &inventory, nil
// }

func (u *PostgresRepository) GetInventoryByID(ctx context.Context, inventory_id string) (*Inventory, error) {
	var (
		query string
		args  []interface{}
	)

	// Build query based on provided inputs
	switch {

	case inventory_id != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image
		         FROM inventories 
		         WHERE id = $1`
		args = append(args, inventory_id)

	default:
		return nil, fmt.Errorf("either inventory_id or slug_ulid must be provided")
	}

	var inventory Inventory
	row := u.Conn.QueryRowContext(ctx, query, args...)

	var (
		createdAt, updatedAt time.Time
		// slug                 sql.NullString
		// ulid                 sql.NullString
		// offerPrice           float64
		// stateSlug            sql.NullString
		// lgaSlug              sql.NullString
		// countrySlug          sql.NullString
		// categorySlug         sql.NullString
		// subcategorySlug      sql.NullString
		primageImage sql.NullString
	)

	err := row.Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.Promoted,
		&inventory.Deactivated,
		&createdAt,
		&updatedAt,

		&inventory.CountryId,
		&inventory.StateId,
		&inventory.LgaId,
		&inventory.Slug,
		&inventory.Ulid,
		&inventory.OfferPrice,

		&inventory.StateSlug,
		&inventory.CountrySlug,
		&inventory.LgaSlug,
		&inventory.CategorySlug,
		&inventory.SubcategorySlug,
		&inventory.ProductPurpose,
		&inventory.Quantity,
		&inventory.IsAvailable,
		&inventory.RentalDuration,
		&inventory.SecurityDeposit,
		&inventory.MinimumPrice,
		&inventory.Metadata,
		&inventory.Negotiable,
		&primageImage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found")
		}
		return nil, fmt.Errorf("error retrieving inventory: %w", err)
	}

	inventory.CreatedAt = createdAt
	inventory.UpdatedAt = updatedAt

	if primageImage.Valid {
		inventory.PrimaryImage = primageImage.String
	} else {
		inventory.PrimaryImage = "NULL"
	}

	// Fetch images for the single inventory
	imgSQL := `
		SELECT id, live_url, local_url, inventory_id, created_at, updated_at
		FROM inventory_images
		WHERE inventory_id = ANY($1)
	`

	imgRows, err := u.Conn.QueryContext(ctx, imgSQL, pq.Array([]string{inventory.ID}))
	if err != nil {
		return nil, fmt.Errorf("select images: %w", err)
	}
	defer imgRows.Close()

	var images []InventoryImage
	for imgRows.Next() {
		img := &InventoryImage{}
		var createdAt, updatedAt time.Time
		if err := imgRows.Scan(
			&img.ID, &img.LiveUrl, &img.LocalUrl, &img.InventoryId,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}

		images = append(images, *img)
	}

	inventory.Images = images

	return &inventory, nil
}
func (u *PostgresRepository) GetCountryByID(ctx context.Context, id string) (*Country, error) {

	query := `SELECT id, name, code, updated_at, created_at FROM countries WHERE id = $1`
	row := u.Conn.QueryRowContext(ctx, query, id)

	var country Country

	err := row.Scan(
		&country.ID,
		&country.Name,
		&country.Code,
		&country.UpdatedAt, // Ensure the order matches the query
		&country.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no country found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving country by ID: %w", err)
	}

	return &country, nil
}

func (u *PostgresRepository) GetStateByID(ctx context.Context, id string) (*State, error) {

	query := `SELECT id, name, state_slug, country_id, updated_at, created_at FROM states WHERE id = $1`
	row := u.Conn.QueryRowContext(ctx, query, id)

	var state State

	err := row.Scan(
		&state.ID,
		&state.Name,
		&state.StateSlug,
		&state.CountryID,
		&state.UpdatedAt, // Ensure the order matches the query
		&state.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no state found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving state by ID: %w", err)
	}

	return &state, nil
}

func (u *PostgresRepository) GetLgaByID(ctx context.Context, id string) (*Lga, error) {

	query := `SELECT id, name, lga_slug, state_id, updated_at, created_at FROM lgas WHERE id = $1`
	row := u.Conn.QueryRowContext(ctx, query, id)

	var lga Lga

	err := row.Scan(
		&lga.ID,
		&lga.Name,
		&lga.LgaSlug,
		&lga.StateID,
		&lga.UpdatedAt, // Ensure the order matches the query
		&lga.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no lga found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving lga by ID: %w", err)
	}

	return &lga, nil
}

func (u *PostgresRepository) GetInventoryByIDOrSlug(ctx context.Context, slug_ulid, inventory_id string) (*Inventory, error) {
	var (
		query string
		args  []interface{}
	)

	// Build query based on provided inputs
	switch {
	case inventory_id != "" && slug_ulid != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image
		         FROM inventories 
		         WHERE id = $1 OR slug = $2`
		args = append(args, inventory_id, slug_ulid)

	case inventory_id != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image
		         FROM inventories 
		         WHERE id = $1`
		args = append(args, inventory_id)

	case slug_ulid != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image
		         FROM inventories 
		         WHERE slug = $1`
		args = append(args, slug_ulid)

	default:
		return nil, fmt.Errorf("either inventory_id or slug_ulid must be provided")
	}

	var inventory Inventory
	row := u.Conn.QueryRowContext(ctx, query, args...)

	var (
		createdAt, updatedAt time.Time
		// slug                 sql.NullString
		// ulid                 sql.NullString
		// offerPrice           float64
		// stateSlug            sql.NullString
		// lgaSlug              sql.NullString
		// countrySlug          sql.NullString
		// categorySlug         sql.NullString
		// subcategorySlug      sql.NullString
		primageImage sql.NullString
	)

	err := row.Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.Promoted,
		&inventory.Deactivated,
		&createdAt,
		&updatedAt,

		&inventory.CountryId,
		&inventory.StateId,
		&inventory.LgaId,
		&inventory.Slug,
		&inventory.Ulid,
		&inventory.OfferPrice,

		&inventory.StateSlug,
		&inventory.CountrySlug,
		&inventory.LgaSlug,
		&inventory.CategorySlug,
		&inventory.SubcategorySlug,
		&inventory.ProductPurpose,
		&inventory.Quantity,
		&inventory.IsAvailable,
		&inventory.RentalDuration,
		&inventory.SecurityDeposit,
		&inventory.MinimumPrice,
		&inventory.Metadata,
		&inventory.Negotiable,
		&primageImage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no inventory found")
		}
		return nil, fmt.Errorf("error retrieving inventory: %w", err)
	}

	inventory.CreatedAt = createdAt
	inventory.UpdatedAt = updatedAt

	if primageImage.Valid {
		inventory.PrimaryImage = primageImage.String
	} else {
		inventory.PrimaryImage = "NULL"
	}

	// Fetch images for the single inventory
	imgSQL := `
		SELECT id, live_url, local_url, inventory_id, created_at, updated_at
		FROM inventory_images
		WHERE inventory_id = ANY($1)
	`

	imgRows, err := u.Conn.QueryContext(ctx, imgSQL, pq.Array([]string{inventory.ID}))
	if err != nil {
		return nil, fmt.Errorf("select images: %w", err)
	}
	defer imgRows.Close()

	var images []InventoryImage
	for imgRows.Next() {
		img := &InventoryImage{}
		var createdAt, updatedAt time.Time
		if err := imgRows.Scan(
			&img.ID, &img.LiveUrl, &img.LocalUrl, &img.InventoryId,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan image: %w", err)
		}

		images = append(images, *img)
	}

	inventory.Images = images

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

type RatingSummary struct {
	FiveStar      int32   `json:"five_star"`
	FourStar      int32   `json:"four_star"`
	ThreeStar     int32   `json:"three_star"`
	TwoStar       int32   `json:"two_star"`
	OneStar       int32   `json:"one_star"`
	AverageRating float64 `json:"average_rating"`
}

func (u *PostgresRepository) GetInventoryRatings(ctx context.Context, id string, page int32, limit int32) ([]*InventoryRating, int32, error) {
	offset := (page - 1) * limit // Calculate offset

	var totalRows int32 // Variable to hold the total count

	// Query to count total rows
	countQuery := "SELECT COUNT(*) FROM inventory_ratings WHERE inventory_id = $1"

	row := u.Conn.QueryRowContext(ctx, countQuery, id)

	if err := row.Scan(&totalRows); err != nil {
		log.Println(err, "ERROR 2")
		return nil, 0, err
	}

	// Query to fetch ratings, rater details, and replies
	query := `
		SELECT 
			ir.id, ir.inventory_id, ir.user_id, ir.rater_id, ir.rating, ir.comment, ir.updated_at, ir.created_at,
			u.id AS rater_id, u.first_name, u.last_name, u.email, u.phone,
			COALESCE(
				JSON_AGG(
					JSON_BUILD_OBJECT(
						'id', irr.id,
						'rating_id', irr.rating_id,
						'replier_id', irr.replier_id,
						'parent_reply_id', irr.parent_reply_id,
						'comment', irr.comment,
						'updated_at', irr.updated_at,
						'created_at', irr.created_at
					)
				) FILTER (WHERE irr.id IS NOT NULL), '[]'
			) AS replies
		FROM inventory_ratings ir
		JOIN users u ON ir.rater_id = u.id
		LEFT JOIN inventory_rating_replies irr ON irr.rating_id = ir.id
		WHERE ir.inventory_id = $1
		GROUP BY ir.id, u.id
		ORDER BY ir.created_at DESC
		LIMIT $2 OFFSET $3
	`

	// stmt.QueryRowContext
	rows, err := u.Conn.QueryContext(ctx, query, id, limit, offset)

	if err != nil {
		log.Println(err, "ERROR 4")
		return nil, 0, err
	}
	defer rows.Close()

	var ratings []*InventoryRating

	// Iterate through the result set
	for rows.Next() {
		var ratingWithRater InventoryRating
		var repliesJSON string

		err := rows.Scan(
			&ratingWithRater.ID,
			&ratingWithRater.InventoryId,
			&ratingWithRater.UserId,
			&ratingWithRater.RaterId,
			&ratingWithRater.Rating,
			&ratingWithRater.Comment,
			&ratingWithRater.UpdatedAt,
			&ratingWithRater.CreatedAt,
			&ratingWithRater.RaterDetails.ID,
			&ratingWithRater.RaterDetails.FirstName,
			&ratingWithRater.RaterDetails.LastName,
			&ratingWithRater.RaterDetails.Email,
			&ratingWithRater.RaterDetails.Phone,
			&repliesJSON, // JSON string of replies
		)
		if err != nil {
			log.Println("Error scanning", err)
			return nil, 0, err
		}

		// Parse replies JSON into a slice of replies
		var replies []InventoryRatingReply
		if err := json.Unmarshal([]byte(repliesJSON), &replies); err != nil {
			log.Println("Error unmarshalling replies", err)
			return nil, 0, err
		}

		// For each reply, include replier details
		for i, reply := range replies {
			// Fetch replier details (this assumes you have a function to get replier details by ID)
			replierDetails, err := u.GetUserByID(ctx, reply.ReplierID)
			if err != nil {
				log.Printf("Error fetching replier details for reply %s: %v", reply.ID, err)
				return nil, 0, err
			}
			replies[i].ReplierDetails = *replierDetails // Populate replier details
		}

		// Assign replies to rating
		ratingWithRater.Replies = replies

		// Add the rating to the ratings slice
		ratings = append(ratings, &ratingWithRater)
	}

	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		log.Println(err, "ERROR 5")
		return nil, 0, err
	}

	return ratings, totalRows, nil
}

func (u *PostgresRepository) GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error) {
	offset := (page - 1) * limit // Calculate offset

	var totalRows int32 // Variable to hold the total count

	// Query to count total rows
	countQuery := "SELECT COUNT(*) FROM user_ratings WHERE user_id = $1"
	row := u.Conn.QueryRowContext(ctx, countQuery, id)
	if err := row.Scan(&totalRows); err != nil {
		return nil, 0, err
	}

	// Query to fetch ratings and rater details
	query := `SELECT 
                  ur.id, ur.user_id, ur.rater_id, ur.rating, ur.comment, ur.updated_at, ur.created_at,
                  u.id AS rater_id, u.first_name, u.last_name, u.email, u.phone
              FROM user_ratings ur
              JOIN users u ON ur.rater_id = u.id
              WHERE ur.user_id = $1
              ORDER BY ur.created_at DESC
              LIMIT $2 OFFSET $3`

	rows, err := u.Conn.QueryContext(ctx, query, id, limit, offset)
	if err != nil {
		log.Println(err, "ERROR")
		return nil, 0, err
	}
	defer rows.Close()

	var ratings []*UserRating

	// Iterate through the result set
	for rows.Next() {
		var ratingWithRater UserRating
		err := rows.Scan(
			&ratingWithRater.ID,
			&ratingWithRater.UserId,
			&ratingWithRater.RaterId,
			&ratingWithRater.Rating,
			&ratingWithRater.Comment,
			&ratingWithRater.UpdatedAt,
			&ratingWithRater.CreatedAt,
			&ratingWithRater.RaterDetails.ID,
			&ratingWithRater.RaterDetails.FirstName,
			&ratingWithRater.RaterDetails.LastName,
			&ratingWithRater.RaterDetails.Email,
			&ratingWithRater.RaterDetails.Phone,
		)
		if err != nil {
			log.Println("Error scanning", err)
			return nil, 0, err
		}

		ratings = append(ratings, &ratingWithRater)
	}

	// Check for errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return ratings, totalRows, nil
}

func (u *PostgresRepository) GetUserRatingSummary(ctx context.Context, userID string) (*RatingSummary, error) {
	query := `SELECT json_build_object(
		'five_star', COALESCE(COUNT(CASE WHEN rating = 5 THEN 1 END), 0),
		'four_star', COALESCE(COUNT(CASE WHEN rating = 4 THEN 1 END), 0),
		'three_star', COALESCE(COUNT(CASE WHEN rating = 3 THEN 1 END), 0),
		'two_star', COALESCE(COUNT(CASE WHEN rating = 2 THEN 1 END), 0),
		'one_star', COALESCE(COUNT(CASE WHEN rating = 1 THEN 1 END), 0),
		'average_rating', COALESCE(ROUND(AVG(rating)::NUMERIC, 1), 0)
	) AS ratings_summary
	FROM user_ratings
	WHERE user_id = $1;`

	row := u.Conn.QueryRowContext(ctx, query, userID)

	var summaryJSON []byte
	err := row.Scan(&summaryJSON)
	if err != nil {
		return nil, err
	}

	var summary RatingSummary
	err = json.Unmarshal(summaryJSON, &summary)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (u *PostgresRepository) GetInventoryRatingSummary(ctx context.Context, inventoryID string) (*RatingSummary, error) {
	query := `SELECT json_build_object(
		'five_star', COALESCE(COUNT(CASE WHEN rating = 5 THEN 1 END), 0),
		'four_star', COALESCE(COUNT(CASE WHEN rating = 4 THEN 1 END), 0),
		'three_star', COALESCE(COUNT(CASE WHEN rating = 3 THEN 1 END), 0),
		'two_star', COALESCE(COUNT(CASE WHEN rating = 2 THEN 1 END), 0),
		'one_star', COALESCE(COUNT(CASE WHEN rating = 1 THEN 1 END), 0),
		'average_rating', COALESCE(ROUND(AVG(rating)::NUMERIC, 1), 0)
	) AS ratings_summary
	FROM inventory_ratings
	WHERE inventory_id = $1;`

	row := u.Conn.QueryRowContext(ctx, query, inventoryID)

	var summaryJSON []byte
	err := row.Scan(&summaryJSON)
	if err != nil {
		log.Println(err, "Error GetInventoryRatingSummary Model")
		return nil, err
	}

	var summary RatingSummary
	err = json.Unmarshal(summaryJSON, &summary)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

type ReplyRatingPayload struct {
	RatingID      string `json:"rating_id"`
	ReplierID     string `json:"replier_id"`
	Comment       string `json:"comment"`
	ParentReplyID string `json:"parent_reply_id"`
}

func (u *PostgresRepository) CreateInventoryRatingReply(ctx context.Context, param *ReplyRatingPayload) (*InventoryRatingReply, error) {

	// Convert empty ParentReplyID to nil for UUID compatibility
	var parentReplyID *string
	if param.ParentReplyID != "" {
		parentReplyID = &param.ParentReplyID
	}

	query := `INSERT INTO inventory_rating_replies (rating_id, replier_id, parent_reply_id, comment, updated_at, created_at)
              VALUES ($1, $2, $3, $4, NOW(), NOW())
              RETURNING id, rating_id, replier_id, parent_reply_id, comment, updated_at, created_at`

	var inventoryRatingReply InventoryRatingReply

	err := u.Conn.QueryRowContext(ctx, query, param.RatingID, param.ReplierID, parentReplyID, param.Comment).Scan(
		&inventoryRatingReply.ID,
		&inventoryRatingReply.RatingID,
		&inventoryRatingReply.ReplierID,
		&inventoryRatingReply.ParentReplyID,
		&inventoryRatingReply.Comment,
		&inventoryRatingReply.UpdatedAt,
		&inventoryRatingReply.CreatedAt,
	)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to create user reply: %w", err)
	}

	return &inventoryRatingReply, nil

}

func (u *PostgresRepository) CreateUserRatingReply(ctx context.Context, param *ReplyRatingPayload) (*UserRatingReply, error) {

	// Convert empty ParentReplyID to nil for UUID compatibility
	var parentReplyID *string
	if param.ParentReplyID != "" {
		parentReplyID = &param.ParentReplyID
	}

	query := `INSERT INTO user_rating_replies (rating_id, replier_id, parent_reply_id, comment, updated_at, created_at)
              VALUES ($1, $2, $3, $4, NOW(), NOW())
              RETURNING id, rating_id, replier_id, parent_reply_id, comment, updated_at, created_at`

	stmt, err := u.Conn.PrepareContext(ctx, query) // create a prepared statement for later execution
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close() // closes when the statement has been executed

	var userRatingReply UserRatingReply
	err = stmt.QueryRowContext(ctx, param.RatingID, param.ReplierID, parentReplyID, param.Comment).Scan(
		&userRatingReply.ID,
		&userRatingReply.RatingID,
		&userRatingReply.ReplierID,
		&userRatingReply.ParentReplyID,
		&userRatingReply.Comment,
		&userRatingReply.UpdatedAt,
		&userRatingReply.CreatedAt,
	)

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("failed to create user reply: %w", err)
	}

	return &userRatingReply, nil

}

// InventoryCollection matches your proto message.
type InventoryCollection struct {
	Inventories []*inventory.Inventory
	TotalCount  int32
	Offset      int32
	Limit       int32
}

type SearchPayload struct {
	CountryID     string `json:"country_id"`
	StateID       string `json:"state_id"`
	LgaID         string `json:"lga_id"`
	Text          string `json:"text"`
	Limit         string `json:"limit"`
	Offset        string `json:"offet"`
	CategoryID    string `json:"category_id"`
	SubcategoryID string `json:"subcategory_id"`
	Ulid          string `json:"ulid"`

	StateSlug       string `json:"state_slug"`
	CountrySlug     string `json:"country_slug"`
	LgaSlug         string `json:"lga_slug"`
	CategorySlug    string `json:"category_slug"`
	SubcategorySlug string `json:"subcategory_slug"`
}

type GetCategoryByIDPayload struct {
	CategoryID   string `json:"category_id"`
	CategorySlug string `json:"category_slug"`
}

func (r *PostgresRepository) SearchInventory(
	ctx context.Context,
	p *SearchPayload,
) (*InventoryCollection, error) {

	log.Println(p, "the payload")
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	log.Println(p, "The param")
	// Parse limit & offset
	limit := 20
	offset := 0
	var err error
	if p.Limit != "" {
		limit, err = strconv.Atoi(p.Limit)
		if err != nil {
			return nil, fmt.Errorf("invalid limit: %w", err)
		}
	}
	if p.Offset != "" {
		offset, err = strconv.Atoi(p.Offset)
		if err != nil {
			return nil, fmt.Errorf("invalid offset: %w", err)
		}
	}

	// Build dynamic WHERE clause
	var (
		conditions []string
		args       []interface{}
		argIdx     = 1
	)

	if p.CountryID != "" {
		conditions = append(conditions, fmt.Sprintf("l.country_id = $%d", argIdx))
		args = append(args, p.CountryID)
		argIdx++
	}
	if p.StateID != "" {
		conditions = append(conditions, fmt.Sprintf("l.state_id = $%d", argIdx))
		args = append(args, p.StateID)
		argIdx++
	}
	if p.LgaID != "" {
		conditions = append(conditions, fmt.Sprintf("l.lga_id = $%d", argIdx))
		args = append(args, p.LgaID)
		argIdx++
	}
	if p.Text != "" {
		conditions = append(conditions, fmt.Sprintf(`
		(to_tsvector('english', coalesce(l.name, '') || ' ' || coalesce(l.description, '')) @@ websearch_to_tsquery('english', $%d)
		OR l.name ILIKE '%%' || $%d || '%%'
		OR l.description ILIKE '%%' || $%d || '%%')
	`, argIdx, argIdx, argIdx))
		args = append(args, p.Text)
		argIdx++
	}
	if p.CategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("l.category_id = $%d", argIdx))
		args = append(args, p.CategoryID)
		argIdx++
	}
	if p.SubcategoryID != "" {
		conditions = append(conditions, fmt.Sprintf("l.subcategory_id = $%d", argIdx))
		args = append(args, p.SubcategoryID)
		argIdx++
	}
	if p.Ulid != "" {
		conditions = append(conditions, fmt.Sprintf("l.ulid = $%d", argIdx))
		args = append(args, p.Ulid)
		argIdx++
	}
	if p.StateSlug != "" {
		conditions = append(conditions, fmt.Sprintf("l.state_slug = $%d", argIdx))
		args = append(args, p.StateSlug)
		argIdx++
	}
	if p.CountrySlug != "" {
		conditions = append(conditions, fmt.Sprintf("l.country_slug = $%d", argIdx))
		args = append(args, p.CountrySlug)
		argIdx++
	}
	if p.LgaSlug != "" {
		conditions = append(conditions, fmt.Sprintf("l.lga_slug = $%d", argIdx))
		args = append(args, p.LgaSlug)
		argIdx++
	}
	if p.CategorySlug != "" {
		conditions = append(conditions, fmt.Sprintf("l.category_slug = $%d", argIdx))
		args = append(args, p.CategorySlug)
		argIdx++
	}
	if p.SubcategorySlug != "" {
		conditions = append(conditions, fmt.Sprintf("l.subcategory_slug = $%d", argIdx))
		args = append(args, p.SubcategorySlug)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total results
	var total int32
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM inventories l %s`, whereClause)
	if err := r.Conn.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count inventories: %w", err)
	}

	// Build SELECT query with LEFT JOINs
	selectSQL := fmt.Sprintf(`
		SELECT
			l.id,
			l.name,
			l.description,
			l.user_id,
			l.category_id,
			l.subcategory_id,
			l.promoted,
			l.deactivated,
			l.created_at,
			l.updated_at,
			l.slug,
			l.ulid,
			l.offer_price,
			l.state_slug,
			l.country_slug,
			l.lga_slug,
			l.category_slug,
			l.subcategory_slug,

			l.product_purpose,
			l.quantity,
			l.is_available,
			l.rental_duration,
			l.security_deposit,
			l.metadata,
			l.negotiable,
			l.primary_image,
			l.minimum_price,

			l.country_id,
			co.name AS country_name,
			l.state_id,
			st.name AS state_name,
			l.lga_id,
			la.name AS lga_name,
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.phone
		FROM inventories l
		LEFT JOIN countries co ON l.country_id = co.id
		LEFT JOIN states st ON l.state_id = st.id
		LEFT JOIN lgas la ON l.lga_id = la.id
		LEFT JOIN users u ON l.user_id = u.id
		%s
		ORDER BY l.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	// Execute SELECT query
	rows, err := r.Conn.QueryContext(ctx, selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("select inventories: %w", err)
	}
	defer rows.Close()

	// Parse inventory rows
	var (
		page []*inventory.Inventory
		ids  []string
	)
	for rows.Next() {
		inv := &inventory.Inventory{
			Country: &inventory.Country{},
			State:   &inventory.State{},
			Lga:     &inventory.LGA{},
			Images:  []*inventory.InventoryImage{},
			User:    &inventory.User{},
		}

		var (
			createdAt, updatedAt time.Time
			slug                 sql.NullString
			ulid                 sql.NullString
			offerPrice           float64
			// minimumPrice         float64
			stateSlug       sql.NullString
			lgaSlug         sql.NullString
			countrySlug     sql.NullString
			categorySlug    sql.NullString
			subcategorySlug sql.NullString
			primageImage    sql.NullString
		)

		if err := rows.Scan(
			&inv.Id,
			&inv.Name,
			&inv.Description,
			&inv.UserId,
			&inv.CategoryId,
			&inv.SubcategoryId,
			&inv.Promoted,
			&inv.Deactivated,
			&createdAt,
			&updatedAt,
			&slug,
			&ulid,
			&offerPrice,
			&stateSlug,
			&countrySlug,
			&lgaSlug,
			&categorySlug,
			&subcategorySlug,

			&inv.ProductPurpose,
			&inv.Quantity,
			&inv.IsAvailable,
			&inv.RentalDuration,
			&inv.SecurityDeposit,
			&inv.Metadata,
			&inv.Negotiable,
			&primageImage,
			&inv.MinimumPrice,

			&inv.CountryId,
			&inv.Country.Name,
			&inv.StateId,
			&inv.State.Name,
			&inv.LgaId,
			&inv.Lga.Name,
			&inv.User.Id,
			&inv.User.Email,
			&inv.User.FirstName,
			&inv.User.LastName,
			&inv.User.Phone,
		); err != nil {
			return nil, fmt.Errorf("scan inventory: %w", err)
		}

		if slug.Valid {
			inv.Slug = slug.String
		} else {
			inv.Slug = ""
		}
		if ulid.Valid {
			inv.Ulid = ulid.String
		} else {
			inv.Ulid = ""
		}

		if stateSlug.Valid {
			inv.StateSlug = stateSlug.String
		} else {
			inv.StateSlug = ""
		}

		if lgaSlug.Valid {
			inv.LgaSlug = lgaSlug.String
		} else {
			inv.LgaSlug = ""
		}

		if countrySlug.Valid {
			inv.CountrySlug = countrySlug.String
		} else {
			inv.CountrySlug = ""
		}
		if categorySlug.Valid {
			inv.CategorySlug = categorySlug.String
		} else {
			inv.CategorySlug = ""
		}

		if subcategorySlug.Valid {
			inv.SubcategorySlug = subcategorySlug.String
		} else {
			inv.SubcategorySlug = ""
		}
		if primageImage.Valid {
			inv.PrimaryImage = primageImage.String
		} else {
			inv.PrimaryImage = "NULL"
		}

		inv.OfferPrice = offerPrice
		// inv.MinimumPrice = minimumPrice
		inv.CreatedAt = timestamppb.New(createdAt)
		inv.UpdatedAt = timestamppb.New(updatedAt)

		page = append(page, inv)
		ids = append(ids, inv.Id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch images in batch
	if len(ids) > 0 {
		imgSQL := `
			SELECT id, live_url, local_url, inventory_id, created_at, updated_at
			FROM inventory_images
			WHERE inventory_id = ANY($1)
		`
		imgRows, err := r.Conn.QueryContext(ctx, imgSQL, pq.Array(ids))
		if err != nil {
			return nil, fmt.Errorf("select images: %w", err)
		}
		defer imgRows.Close()

		imgMap := make(map[string][]*inventory.InventoryImage)
		for imgRows.Next() {
			img := &inventory.InventoryImage{}
			var createdAt, updatedAt time.Time
			if err := imgRows.Scan(
				&img.Id, &img.LiveUrl, &img.LocalUrl, &img.InventoryId,
				&createdAt, &updatedAt,
			); err != nil {
				return nil, fmt.Errorf("scan image: %w", err)
			}
			img.CreatedAt = timestamppb.New(createdAt)
			img.UpdatedAt = timestamppb.New(updatedAt)
			imgMap[img.InventoryId] = append(imgMap[img.InventoryId], img)
		}
		for _, inv := range page {
			inv.Images = imgMap[inv.Id]
		}
	}

	// Return paginated result
	return &InventoryCollection{
		Inventories: page,
		TotalCount:  total,
		Offset:      int32(offset),
		Limit:       int32(limit),
	}, nil
}

type CreateBookingPayload struct {
	OwnerId           string
	RenterId          string
	InventoryId       string
	RentalType        string
	RentalDuration    int32
	SecurityDeposit   float64
	OfferPricePerUnit float64
	Quantity          int32
	TotalAmount       float64
	StartDate         time.Time // for DATE (YYYY-MM-DD)
	EndDate           time.Time // for DATE (YYYY-MM-DD)
	EndTime           string
}

func (b *PostgresRepository) CreateBooking(ctx context.Context, p *CreateBookingPayload) (*InventoryBooking, error) {

	log.Println(p)
	query := `INSERT INTO inventory_bookings 
		(
			inventory_id, 
			renter_id, 
			owner_id, 
			start_date, 
			end_date, 
			end_time, 
			offer_price_per_unit, 
			total_amount, 
			security_deposit, 
			quantity, 
			rental_type, 
			rental_duration,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW()) 
		RETURNING 
			id,  
			inventory_id,  
			renter_id,  
			owner_id,  
			start_date,  
			end_date,  
			end_time,  
			offer_price_per_unit,  
			total_amount,  
			security_deposit,  
			quantity,  
			status,  
			payment_status,  
			rental_type,  
			rental_duration,  
			created_at,  
			updated_at`

	var inventoryBooking InventoryBooking
	err := b.Conn.QueryRowContext(
		ctx,
		query,
		p.InventoryId,
		p.RenterId,
		p.OwnerId,
		p.StartDate,
		p.EndDate,
		p.EndTime,
		p.OfferPricePerUnit,
		p.TotalAmount,
		p.SecurityDeposit,
		p.Quantity,
		p.RentalType,
		p.RentalDuration,
	).Scan(
		&inventoryBooking.ID,
		&inventoryBooking.InventoryID,
		&inventoryBooking.RenterID,
		&inventoryBooking.OwnerID,
		&inventoryBooking.StartDate,
		&inventoryBooking.EndDate,
		&inventoryBooking.EndTime,
		&inventoryBooking.OfferPricePerUnit,
		&inventoryBooking.TotalAmount,
		&inventoryBooking.SecurityDeposit,
		&inventoryBooking.Quantity,
		&inventoryBooking.Status,
		&inventoryBooking.PaymentStatus,
		&inventoryBooking.RentalType,
		&inventoryBooking.RentalDuration,
		&inventoryBooking.CreatedAt,
		&inventoryBooking.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create inventory booking: %w", err)
	}

	return &inventoryBooking, nil
}

type CreatePurchaseOrderPayload struct {
	SellerId          string
	BuyerId           string
	InventoryId       string
	OfferPricePerUnit float64
	Quantity          int32
	TotalAmount       float64
}

func (b *PostgresRepository) CreatePurchaseOrder(ctx context.Context, p *CreatePurchaseOrderPayload) (*InventorySale, error) {

	query := `INSERT INTO inventory_sales
		(
			inventory_id, 
			seller_id, 
			buyer_id, 
			offer_price_per_unit, 
			quantity, 
			total_amount,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
		RETURNING 
			id,  
			inventory_id, 
			seller_id, 
			buyer_id, 
			offer_price_per_unit, 
			quantity, 
			total_amount,
			status,
			payment_status,
			created_at, 
			updated_at`

	var inventorySale InventorySale
	err := b.Conn.QueryRowContext(
		ctx,
		query,
		p.InventoryId,
		p.SellerId,
		p.BuyerId,
		p.OfferPricePerUnit,
		p.Quantity,
		p.TotalAmount,
	).Scan(
		&inventorySale.ID,
		&inventorySale.InventoryID,
		&inventorySale.SellerID,
		&inventorySale.BuyerID,
		&inventorySale.OfferPricePerUnit,
		&inventorySale.Quantity,
		&inventorySale.TotalAmount,
		&inventorySale.Status,
		&inventorySale.PaymentStatus,
		&inventorySale.CreatedAt,
		&inventorySale.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create purchase order: %w", err)
	}

	return &inventorySale, nil
}

// // Message struct defines the message payload
// type Message struct {
// 	Content  string `json:"content"`
// 	Sender   string `json:"sender"`
// 	Receiver string `json:"receiver"`
// 	SentAt   int64  `json:"sent_at"`

// }
type Message struct {
	Content     string `json:"content"`
	Sender      string `json:"sender"`
	Receiver    string `json:"receiver"`
	SentAt      int64  `json:"sent_at"`
	Type        string `json:"type,omitempty"`        // "text", "image", "file"
	ContentType string `json:"contentType,omitempty"` // e.g. "image/png", "application/pdf"
}

func (c *PostgresRepository) SubmitChat(ctx context.Context, p *Message) (*Chat, error) {

	query := `INSERT INTO chats
		(
			content,
			sender_id, 
			receiver_id, 
			sent_at, 
			type,
			content_type,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) 
		RETURNING 
			id,  
			content,
			sender_id, 
			receiver_id, 
			sent_at, 
			type,
			content_type,
			created_at, 
			updated_at`

	var chat Chat
	err := c.Conn.QueryRowContext(
		ctx,
		query,
		p.Content,
		p.Sender,
		p.Receiver,
		p.SentAt,
	).Scan(
		&chat.ID,
		&chat.Content,
		&chat.SenderID,
		&chat.ReceiverID,
		&chat.SentAt,
		&chat.Type,
		&chat.ContentType,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return &chat, nil
}

func (c *PostgresRepository) GetChatHistory(ctx context.Context, userA, userB string) ([]Chat, error) {
	query := `
		SELECT id, content, sender_id, receiver_id, sent_at, created_at, updated_at
		FROM chats
		WHERE (sender_id = $1 AND receiver_id = $2)
		   OR (sender_id = $2 AND receiver_id = $1)
		ORDER BY sent_at ASC
	`

	rows, err := c.Conn.QueryContext(ctx, query, userA, userB)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		err := rows.Scan(
			&chat.ID,
			&chat.Content,
			&chat.SenderID,
			&chat.ReceiverID,
			&chat.SentAt,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		chats = append(chats, chat)
	}

	return chats, nil
}

type ChatSummary struct {
	ID         string    `json:"id"`
	Content    string    `json:"last_message"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	SentAt     int64     `json:"sent_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	PartnerID  string    `json:"partner_id"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	Email      string    `json:"email"`
}

func (r *PostgresRepository) GetChatList(ctx context.Context, userID string) ([]ChatSummary, error) {
	query := `
		SELECT * FROM (
		SELECT DISTINCT ON (partner_id)
			chats.id,
			chats.content,
			chats.sender_id,
			chats.receiver_id,
			chats.sent_at,
			chats.created_at,
			chats.updated_at,
			u.id AS partner_id,
			u.first_name,
			u.last_name,
			u.email
		FROM chats
		JOIN users u
			ON u.id = CASE
						WHEN sender_id = $1 THEN receiver_id
						ELSE sender_id
					END
		WHERE sender_id = $1 OR receiver_id = $1
		ORDER BY partner_id, sent_at DESC
		) sub
		ORDER BY sent_at DESC
	`

	rows, err := r.Conn.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var summaries []ChatSummary

	for rows.Next() {
		var s ChatSummary
		err := rows.Scan(
			&s.ID,
			&s.Content,
			&s.SenderID,
			&s.ReceiverID,
			&s.SentAt,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.PartnerID,
			&s.FirstName,
			&s.LastName,
			&s.Email,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}
