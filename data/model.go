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
	"google.golang.org/protobuf/types/known/wrapperspb"
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

	// 1.Loop through the response to count the inventories for each category and subcategory
	for i, category := range categories {

		var catCount int32

		// execute query to count in inventories where category_id matches category.ID
		catCountQuery := `SELECT COUNT(*) FROM inventories WHERE category_id = $1`

		catRow := u.Conn.QueryRowContext(ctx, catCountQuery, category.ID)

		if err := catRow.Scan(&catCount); err != nil {
			log.Println("Error scanning row category count:", err)
		}

		categories[i].InventoryCount = catCount

		subcategories := categories[i].Subcategories
		// loop through the subcategories
		for k, subcategory := range subcategories {

			var subCatCount int32
			// execute query to count in inventories where category_id matches category.ID
			subcatCountQuery := `SELECT COUNT(*) FROM inventories WHERE subcategory_id = $1`

			subCatRow := u.Conn.QueryRowContext(ctx, subcatCountQuery, subcategory.ID)

			if err := subCatRow.Scan(&subCatCount); err != nil {
				log.Println("Error scanning row category count:", err)
			}

			subcategories[k].InventoryCount = subCatCount
		}

	}

	log.Printf("%+v", categories[0])
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

	for k, subcategory := range subCategories {

		var subCatCount int32
		// execute query to count in inventories where category_id matches category.ID
		subcatCountQuery := `SELECT COUNT(*) FROM inventories WHERE subcategory_id = $1`

		subCatRow := u.Conn.QueryRowContext(ctx, subcatCountQuery, subcategory.ID)

		if err := subCatRow.Scan(&subCatCount); err != nil {
			log.Println("Error scanning row category count:", err)
		}

		subCategories[k].InventoryCount = subCatCount
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
	Condition       string
	UsageGuide      string
	Included        string
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
	usageGuide := req.UsageGuide
	condition := req.Condition
	included := req.Included

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
				usage_guide,
				condition,
				included,

				updated_at, 
				created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8,$9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, NOW(), NOW()) 
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
				usage_guide,
				condition,
				included,

				updated_at, 
				created_at`

	var (
		inventory      Inventory
		userTags       sql.NullString
		itemCondition  sql.NullString
		itemUsageGuide sql.NullString
		itemIncluded   sql.NullString
	)
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
		usageGuide,
		condition,
		included,
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
		&itemUsageGuide,
		&itemCondition,
		&itemIncluded,

		&inventory.CreatedAt,
		&inventory.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create inventory: %w", err)
	}

	if userTags.Valid {
		inventory.Tags = wrapperspb.String(userTags.String)
	} else {
		inventory.Tags = &wrapperspb.StringValue{}
	}

	if itemCondition.Valid {
		inventory.Condition = wrapperspb.String(itemCondition.String)
	} else {
		inventory.Condition = &wrapperspb.StringValue{}
	}

	if itemUsageGuide.Valid {
		inventory.UsageGuide = wrapperspb.String(itemUsageGuide.String)
	} else {
		inventory.UsageGuide = &wrapperspb.StringValue{}
	}
	if itemIncluded.Valid {
		inventory.Included = wrapperspb.String(itemIncluded.String)
	} else {
		inventory.Included = &wrapperspb.StringValue{}
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
			log.Println("no inventory found", err)
			return nil, fmt.Errorf("no inventory found")
		}

		log.Println("no inventory found", err)
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

	// inventory rating
	// Average rating query for one inventory
	ratingSQL := `
    SELECT COALESCE(AVG(rating), 0) AS average_rating
    FROM inventory_ratings
    WHERE inventory_id = $1
`
	var avgRating float64
	err = u.Conn.QueryRowContext(ctx, ratingSQL, inventory.ID).Scan(&avgRating)
	if err != nil {
		return nil, fmt.Errorf("select average rating: %w", err)
	}
	// Assign pointer if protobuf expects *float64, else just assign float64
	inventory.AverageRating = &avgRating

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
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image,
		         tags, condition, usage_guide, included FROM inventories 
		         WHERE id = $1 OR slug = $2`
		args = append(args, inventory_id, slug_ulid)

	case inventory_id != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image,
		         tags, condition, usage_guide, included FROM inventories 
		         WHERE id = $1`
		args = append(args, inventory_id)

	case slug_ulid != "":
		query = `SELECT id, name, description, user_id, category_id, subcategory_id, promoted, deactivated, updated_at, created_at,
				 country_id, state_id, lga_id, slug, ulid, offer_price, state_slug, country_slug, lga_slug, category_slug, subcategory_slug,
				 product_purpose, quantity, is_available, rental_duration, security_deposit, minimum_price, metadata, negotiable, primary_image,
		         tags, condition, usage_guide, included FROM inventories 
		         WHERE slug = $1`
		args = append(args, slug_ulid)

	default:
		return nil, fmt.Errorf("either inventory_id or slug_ulid must be provided")
	}

	var inventory Inventory
	row := u.Conn.QueryRowContext(ctx, query, args...)

	var (
		createdAt, updatedAt time.Time
		primageImage         sql.NullString
		userTags             sql.NullString
		itemCondition        sql.NullString
		itemUsageGuide       sql.NullString
		itemIncluded         sql.NullString
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

		&userTags,
		&itemCondition,
		&itemUsageGuide,
		&itemIncluded,
	)

	if err != nil {
		if err == sql.ErrNoRows {

			log.Println(err, "THE ERROR IN MODEL 0")
			return nil, fmt.Errorf("no inventory found")
		}

		log.Println(err, "THE ERROR IN MODEL 1")
		return nil, fmt.Errorf("error retrieving inventory: %w", err)
	}

	inventory.CreatedAt = createdAt
	inventory.UpdatedAt = updatedAt

	if userTags.Valid {
		inventory.Tags = wrapperspb.String(userTags.String)
	} else {
		inventory.Tags = &wrapperspb.StringValue{}
	}

	if itemCondition.Valid {
		inventory.Condition = wrapperspb.String(itemCondition.String)
	} else {
		inventory.Condition = &wrapperspb.StringValue{}
	}

	if itemUsageGuide.Valid {
		inventory.UsageGuide = wrapperspb.String(itemUsageGuide.String)
	} else {
		inventory.UsageGuide = &wrapperspb.StringValue{}
	}
	if itemIncluded.Valid {
		inventory.Included = wrapperspb.String(itemIncluded.String)
	} else {
		inventory.Included = &wrapperspb.StringValue{}
	}

	if primageImage.Valid {
		inventory.PrimaryImage = primageImage.String
	} else {
		inventory.PrimaryImage = "NULL"
	}

	//============================================================================================================================
	// Fetch images for the single inventory
	imgSQL := `
		SELECT id, live_url, local_url, inventory_id, created_at, updated_at
		FROM inventory_images
		WHERE inventory_id = ANY($1)
	`

	imgRows, err := u.Conn.QueryContext(ctx, imgSQL, pq.Array([]string{inventory.ID}))
	if err != nil {

		log.Println(err, "THE ERROR IN MODEL 2")
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

			log.Println(err, "THE ERROR IN MODEL 3")
			return nil, fmt.Errorf("scan image: %w", err)
		}

		images = append(images, *img)
	}

	inventory.Images = images

	//============================================================================================================================
	// Average rating and count query for one inventory
	ratingSQL := `
    SELECT 
      COALESCE(AVG(rating), 0) AS average_rating,
      COUNT(*) AS total_ratings
    FROM 
      inventory_ratings
    WHERE 
      inventory_id = $1
`

	var avgRating float64
	var totalRatings int32

	err = u.Conn.QueryRowContext(ctx, ratingSQL, inventory.ID).Scan(&avgRating, &totalRatings)
	if err != nil {

		log.Println(err, "THE ERROR IN MODEL 4")
		return nil, fmt.Errorf("select average rating and count: %w", err)
	}

	// If your protobuf field is *float64, use pointers:
	inventory.AverageRating = &avgRating

	// For count, you might want a new field:
	inventory.TotalRatings = &totalRatings // or assign to int64 directly if non-pointer
	//=================================================================================================================================

	// Check user KYC
	userVerification := u.GetUserVerified(ctx, inventory.UserId)
	inventory.UserVerified = &userVerification

	return &inventory, nil
}

func (u *PostgresRepository) GetUserVerified(ctx context.Context, userID string) bool {
	var verified bool

	renterKycSQL := `SELECT verified FROM renter_kycs WHERE user_id = $1`
	err := u.Conn.QueryRowContext(ctx, renterKycSQL, userID).Scan(&verified)
	if err != nil {
		if err == sql.ErrNoRows {
			// Not found in renter_kycs, check business_kycs
			businessKycSQL := `SELECT verified FROM business_kycs WHERE user_id = $1`
			err = u.Conn.QueryRowContext(ctx, businessKycSQL, userID).Scan(&verified)
			if err != nil {
				if err == sql.ErrNoRows {
					// Not found in either table, default to false
					return false
				}
				return false
			}
		} else {
			return false
		}
	}

	return verified
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

	query := `SELECT id, email, first_name, last_name, phone, verified, profile_img, updated_at, created_at, user_slug FROM users WHERE id = $1`

	row := u.Conn.QueryRowContext(ctx, query, id)

	var user User
	var userImg sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Verified,
		&userImg,
		&user.UpdatedAt,
		&user.CreatedAt,
		&user.UserSlug,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user found with ID %s", id)
		}
		return nil, fmt.Errorf("error retrieving user by ID: %w", err)
	}

	if userImg.Valid {
		user.ProfileImg = wrapperspb.String(userImg.String)
	} else {
		user.ProfileImg = &wrapperspb.StringValue{}
	}
	log.Println(user, "the user is here")

	return &user, nil
}
func (r *PostgresRepository) GetUserBySlug(ctx context.Context, slug string) (*User, error) {
	const query = `
			SELECT
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.phone,
			u.verified,
			u.profile_img,
			u.updated_at,
			u.created_at,
			u.user_slug,
			COALESCE(array_agg(at.name) FILTER (WHERE at.name IS NOT NULL), '{}') AS account_types
			FROM users u
			LEFT JOIN user_account_types uat ON uat.user_id = u.id
			LEFT JOIN account_types     at  ON at.id      = uat.account_type_id
			WHERE u.user_slug = $1
			GROUP BY
			u.id,
			u.email,
			u.first_name,
			u.last_name,
			u.phone,
			u.verified,
			u.profile_img,
			u.updated_at,
			u.created_at,
			u.user_slug;
		`

	row := r.Conn.QueryRowContext(ctx, query, slug)

	var user User
	var userImg sql.NullString
	var rawTypes pq.StringArray

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Verified,
		&userImg,
		&user.UpdatedAt,
		&user.CreatedAt,
		&user.UserSlug,
		&rawTypes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user found with slug %q", slug)
		}
		return nil, fmt.Errorf("error retrieving user by slug: %w", err)
	}

	if userImg.Valid {
		user.ProfileImg = wrapperspb.String(userImg.String)
	} else {
		user.ProfileImg = &wrapperspb.StringValue{}
	}

	// Map rawTypes ([]string) into []AccountType
	for _, name := range rawTypes {
		// skip any empty entries
		if strings.TrimSpace(name) == "" {
			continue
		}
		user.AccountTypes = append(user.AccountTypes, AccountType{Name: name})
	}

	log.Printf("loaded user %+v with types %#v\n", user, rawTypes)
	return &user, nil
}

func (u *PostgresRepository) GetBusinessBySubdomain(ctx context.Context, domain string) (*BusinessKyc, error) {

	query := `
		SELECT 
			bk.id, bk.user_id, bk.subdomain, bk.verified, bk.updated_at, bk.created_at,
			bk.state_id, bk.lga_id, bk.shop_banner, bk.plan_id, bk.key_bonus, bk.display_name, 
			bk.description, bk.country_id, bk.address, bk.business_registered, bk.industries,
			u.id, u.email, u.first_name, u.last_name, u.phone, u.password,
			u.profile_img, u.verified, u.created_at, u.updated_at, u.user_slug,
			array_agg(COALESCE(at.name, '')) AS account_type_names
		FROM business_kycs AS bk
		LEFT JOIN users u ON bk.user_id = u.id
		LEFT JOIN user_account_types uat ON uat.user_id = u.id
		LEFT JOIN account_types at ON at.id = uat.account_type_id
		WHERE bk.subdomain = $1
		GROUP BY bk.id, u.id
	`

	row := u.Conn.QueryRowContext(ctx, query, domain)

	var bkyc BusinessKyc
	var user User
	var rawTypes pq.StringArray
	var profileImg sql.NullString
	var rawShopBanner sql.NullString
	var rawKeyBonus sql.NullString

	err := row.Scan(
		&bkyc.ID,
		&bkyc.UserID,
		&bkyc.Subdomain,
		&bkyc.Verified,
		&bkyc.UpdatedAt,
		&bkyc.CreatedAt,
		&bkyc.StateID,
		&bkyc.LgaID,
		&rawShopBanner,
		&bkyc.PlanID,
		&rawKeyBonus,
		&bkyc.DisplayName,
		&bkyc.Description,
		&bkyc.CountryID,
		&bkyc.Address,
		&bkyc.BusinessRegistered,
		&bkyc.Industries,

		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Password,
		&profileImg,
		&user.Verified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.UserSlug,
		&rawTypes,
	)

	if err != nil {
		log.Printf("%v", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no business found with subdomain %s", domain)
		}

		log.Printf("%v", err)
		return nil, fmt.Errorf("error retrieving business by subdomain: %w", err)
	}

	if profileImg.Valid {
		user.ProfileImg = wrapperspb.String(profileImg.String)
	}

	if rawShopBanner.Valid {
		bkyc.ShopBanner = rawShopBanner.String
	}

	if rawKeyBonus.Valid {
		bkyc.KeyBonus = rawKeyBonus.String
	}

	for _, name := range rawTypes {
		if strings.TrimSpace(name) != "" {
			user.AccountTypes = append(user.AccountTypes, AccountType{Name: name})
		}
	}

	// You might want to assign user to bkyc.User or similar if your model expects that.

	return &bkyc, nil

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
			u.id AS rater_id, u.first_name, u.last_name, u.email, u.phone, u.profile_img,
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
		log.Println(err, "ERROR 3")
		return nil, 0, err
	}
	defer rows.Close()

	var ratings []*InventoryRating

	// Iterate through the result set
	for rows.Next() {

		var (
			ratingWithRater InventoryRating
			repliesJSON     string
			imgNull         sql.NullString
		)

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
			&imgNull,
			&repliesJSON, // JSON string of replies
		)
		if err != nil {
			return nil, 0, err
		}

		if imgNull.Valid {
			ratingWithRater.RaterDetails.ProfileImg = wrapperspb.String(imgNull.String)
		} else {
			ratingWithRater.RaterDetails.ProfileImg = nil
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
				log.Println(err, "ERROR 5")
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
		log.Println(err)
		return nil, 0, err
	}

	return ratings, totalRows, nil
}

// func (u *PostgresRepository) GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error) {
// 	offset := (page - 1) * limit // Calculate offset

// 	var totalRows int32 // Variable to hold the total count

// 	// Query to count total rows
// 	countQuery := "SELECT COUNT(*) FROM user_ratings WHERE user_id = $1"
// 	row := u.Conn.QueryRowContext(ctx, countQuery, id)
// 	if err := row.Scan(&totalRows); err != nil {
// 		return nil, 0, err
// 	}

// 	// Query to fetch ratings and rater details
// 	query := `SELECT
//                   ur.id, ur.user_id, ur.rater_id, ur.rating, ur.comment, ur.updated_at, ur.created_at,
//                   u.id AS rater_id, u.first_name, u.last_name, u.email, u.phone, u.profile_img
//               FROM user_ratings ur
//               JOIN users u ON ur.rater_id = u.id
//               WHERE ur.user_id = $1
//               ORDER BY ur.created_at DESC
//               LIMIT $2 OFFSET $3`

// 	rows, err := u.Conn.QueryContext(ctx, query, id, limit, offset)
// 	if err != nil {
// 		log.Println(err, "ERROR")
// 		return nil, 0, err
// 	}
// 	defer rows.Close()

// 	var ratings []*UserRating

// 	// Iterate through the result set
// 	for rows.Next() {
// 		var ratingWithRater UserRating
// 		var imgNull sql.NullString
// 		err := rows.Scan(
// 			&ratingWithRater.ID,
// 			&ratingWithRater.UserId,
// 			&ratingWithRater.RaterId,
// 			&ratingWithRater.Rating,
// 			&ratingWithRater.Comment,
// 			&ratingWithRater.UpdatedAt,
// 			&ratingWithRater.CreatedAt,
// 			&ratingWithRater.RaterDetails.ID,
// 			&ratingWithRater.RaterDetails.FirstName,
// 			&ratingWithRater.RaterDetails.LastName,
// 			&ratingWithRater.RaterDetails.Email,
// 			&ratingWithRater.RaterDetails.Phone,
// 			&imgNull,
// 		)
// 		if err != nil {
// 			log.Println("Error scanning", err)
// 			return nil, 0, err
// 		}

// 		if imgNull.Valid {
// 			ratingWithRater.RaterDetails.ProfileImg = wrapperspb.String(imgNull.String)
// 		} else {
// 			ratingWithRater.RaterDetails.ProfileImg = nil
// 		}

// 		ratings = append(ratings, &ratingWithRater)
// 	}

// 	// Check for errors encountered during iteration
// 	if err := rows.Err(); err != nil {
// 		return nil, 0, err
// 	}

// 	return ratings, totalRows, nil
// }

func (u *PostgresRepository) GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error) {
	offset := (page - 1) * limit

	var totalRows int32
	countQuery := "SELECT COUNT(*) FROM user_ratings WHERE user_id = $1"
	row := u.Conn.QueryRowContext(ctx, countQuery, id)
	if err := row.Scan(&totalRows); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			ur.id, ur.user_id, ur.rater_id, ur.rating, ur.comment, ur.updated_at, ur.created_at,
			u.id AS rater_id, u.first_name, u.last_name, u.email, u.phone, u.profile_img,
			COUNT(urr.id) AS replies_count
		FROM user_ratings ur
		JOIN users u ON ur.rater_id = u.id
		LEFT JOIN user_rating_replies urr ON urr.rating_id = ur.id
		WHERE ur.user_id = $1
		GROUP BY ur.id, u.id
		ORDER BY ur.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := u.Conn.QueryContext(ctx, query, id, limit, offset)
	if err != nil {
		log.Println(err, "ERROR")
		return nil, 0, err
	}
	defer rows.Close()

	var ratings []*UserRating

	for rows.Next() {
		var rating UserRating
		var imgNull sql.NullString

		err := rows.Scan(
			&rating.ID,
			&rating.UserId,
			&rating.RaterId,
			&rating.Rating,
			&rating.Comment,
			&rating.UpdatedAt,
			&rating.CreatedAt,
			&rating.RaterDetails.ID,
			&rating.RaterDetails.FirstName,
			&rating.RaterDetails.LastName,
			&rating.RaterDetails.Email,
			&rating.RaterDetails.Phone,
			&imgNull,
			&rating.RepliesCount,
		)
		if err != nil {
			log.Println("Error scanning rating:", err)
			return nil, 0, err
		}

		if imgNull.Valid {
			rating.RaterDetails.ProfileImg = wrapperspb.String(imgNull.String)
		} else {
			rating.RaterDetails.ProfileImg = nil
		}

		ratings = append(ratings, &rating)
	}

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
	UserID          string `json:"user_id"`
	ProductPurpose  string `json:"product_purpose"`
	UserSlug        string `json:"user_slug"`
	Subdomain       string `json:"subdomain"`
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
	if p.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("l.user_id = $%d", argIdx))
		args = append(args, p.UserID)
		argIdx++
	}
	if p.ProductPurpose != "" {
		conditions = append(conditions, fmt.Sprintf("l.product_purpose = $%d", argIdx))
		args = append(args, p.ProductPurpose)
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

	// Fetch average ratings in batch — ONLY this part is new
	if len(ids) > 0 {
		ratingSQL := `
			SELECT inventory_id, COALESCE(AVG(rating), 0) AS average_rating
			FROM inventory_ratings
			WHERE inventory_id = ANY($1)
			GROUP BY inventory_id
		`
		ratingRows, err := r.Conn.QueryContext(ctx, ratingSQL, pq.Array(ids))
		if err != nil {
			return nil, fmt.Errorf("select average ratings: %w", err)
		}
		defer ratingRows.Close()

		ratingMap := make(map[string]float64)
		for ratingRows.Next() {
			var inventoryID string
			var avgRating float64
			if err := ratingRows.Scan(&inventoryID, &avgRating); err != nil {
				return nil, fmt.Errorf("scan rating: %w", err)
			}
			ratingMap[inventoryID] = avgRating
		}
		for _, inv := range page {

			if avg, ok := ratingMap[inv.Id]; ok {
				inv.AverageRating = &avg
			} else {
				inv.AverageRating = float64Ptr(0.0)
			}

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

func float64Ptr(f float64) *float64 {
	return &f
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
	StartTime         string
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
			start_time,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW()) 
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
			start_time,
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
		p.StartTime,
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
		&inventoryBooking.StartTime,
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
	Content     string  `json:"content"`
	Sender      string  `json:"sender"`
	ReplyTo     *string `json:"reply_to"`
	Receiver    string  `json:"receiver"`
	SentAt      int64   `json:"sent_at"`
	Type        string  `json:"type,omitempty"`         // "text", "image", "file"
	ContentType string  `json:"content_type,omitempty"` // e.g. "image/png", "application/pdf"
	MessageID   string  `json:"message_id"`
}

func (c *PostgresRepository) SubmitChat(ctx context.Context, p *Message) (*Chat, error) {
	log.Println("GOT TO REPO", p)
	log.Println("INSERTED ID", p.MessageID)

	query := `INSERT INTO chats
		(
			id,
			content,
			sender_id, 
			receiver_id, 
			sent_at, 
			type,
			content_type,
			reply_to_id,
			created_at, 
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) 
		RETURNING 
			id,  
			content,
			sender_id, 
			receiver_id, 
			sent_at, 
			type,
			content_type,
			reply_to_id,
			created_at, 
			updated_at`

	// Convert empty string to nil for reply_to_id
	var replyTo interface{}
	if p.ReplyTo != nil && *p.ReplyTo != "" {
		replyTo = *p.ReplyTo
	} else {
		replyTo = nil
	}

	var chat Chat
	err := c.Conn.QueryRowContext(
		ctx,
		query,
		p.MessageID,
		p.Content,
		p.Sender,
		p.Receiver,
		p.SentAt,
		p.Type,
		p.ContentType,
		replyTo,
	).Scan(
		&chat.ID,
		&chat.Content,
		&chat.SenderID,
		&chat.ReceiverID,
		&chat.SentAt,
		&chat.Type,
		&chat.ContentType,
		&chat.ReplyTo,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)

	if err != nil {
		log.Println("Error: failed to create chat DB record", err)
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	return &chat, nil
}

func (c *PostgresRepository) GetChatHistory(ctx context.Context, userA, userB string) ([]Chat, error) {
	query := `
			SELECT
				id,
				content,
				sender_id,
				receiver_id,
				sent_at,
				type,
				content_type,
				is_read,
				reply_to_id,
				created_at,
				updated_at
			FROM chats
			WHERE (
				(sender_id   = $1 AND receiver_id = $2)
			OR (sender_id   = $2 AND receiver_id = $1)
			)
				AND deleted_at IS NULL
			ORDER BY sent_at ASC;
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
			&chat.Type,
			&chat.ContentType,
			&chat.IsRead,
			&chat.ReplyTo,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return chats, nil
}

type ChatSummary struct {
	ID          string                  `json:"id"`
	Content     string                  `json:"last_message"`
	SenderID    string                  `json:"sender_id"`
	ReceiverID  string                  `json:"receiver_id"`
	SentAt      int64                   `json:"sent_at"`
	Type        string                  `json:"type"`
	ContentType string                  `json:"content_type"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	PartnerID   string                  `json:"partner_id"`
	FirstName   string                  `json:"first_name"`
	LastName    string                  `json:"last_name"`
	Email       string                  `json:"email"`
	Phone       string                  `json:"phone"`
	ProfileImg  *wrapperspb.StringValue `json:"profile_img"`
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
			chats.type,
			chats.content_type,
			chats.created_at,
			chats.updated_at,
			u.id AS partner_id,
			u.first_name,
			u.last_name,
			u.email,
			u.phone,
			u.profile_img
		FROM chats
		JOIN users u
			ON u.id = CASE
						WHEN sender_id = $1 THEN receiver_id
						ELSE sender_id
					END
		WHERE (sender_id = $1 OR receiver_id = $1)
		AND deleted_at IS NULL
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
	var userImg sql.NullString

	for rows.Next() {
		var s ChatSummary
		err := rows.Scan(
			&s.ID,
			&s.Content,
			&s.SenderID,
			&s.ReceiverID,
			&s.SentAt,
			&s.Type,
			&s.ContentType,
			&s.CreatedAt,
			&s.UpdatedAt,
			&s.PartnerID,
			&s.FirstName,
			&s.LastName,
			&s.Email,
			&s.Phone,
			&userImg,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		if userImg.Valid {
			s.ProfileImg = wrapperspb.String(userImg.String)
		} else {
			s.ProfileImg = &wrapperspb.StringValue{}
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

func (r *PostgresRepository) GetUnreadChat(ctx context.Context, userID string) (int32, error) {
	var count int32
	query := `SELECT COUNT(*) FROM chats WHERE receiver_id = $1 AND is_read = false`

	err := r.Conn.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error getting unread chat count: %w", err)
	}

	return count, nil
}

func (repo *PostgresRepository) MarkChatAsRead(ctx context.Context, userID, senderID string) error {
	_, err := repo.Conn.ExecContext(ctx, `
		UPDATE chats
		SET is_read = true
		WHERE receiver_id = $1 AND sender_id = $2 AND is_read = false
	`, userID, senderID)
	return err
}

type BusinessAnalytics struct {
	BusinessKycID      string  `json:"business_kyc_id"`
	DisplayName        string  `json:"display_name"`
	Description        *string `json:"description"`
	Address            string  `json:"address"`
	CacNumber          *string `json:"cac_number,omitempty"`
	KeyBonus           *string `json:"key_bonus"`
	BusinessRegistered string  `json:"business_registered"`
	Verified           bool    `json:"verified"`
	ActivePlan         bool    `json:"active_plan"`

	CountryID   string `json:"country_id"`
	CountryName string `json:"country_name"` // ✅ new

	StateID   string `json:"state_id"`
	StateName string `json:"state_name"` // ✅ new

	LgaID   string `json:"lga_id"`
	LgaName string `json:"lga_name"` // ✅ new

	UserID    string `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`

	PlanName         string  `json:"plan_name"`
	TotalInventories int64   `json:"total_inventories"`
	AverageRating    float64 `json:"average_rating"`
	ShopBanner       *string `json:"shop_banner"`
	Industries       *string `json:"industries"`
	Subdomain        *string `json:"subdomain"`
}

type SearchPremiumPartnerPayload struct {
	Text     string `json:"text"`
	Industry string `json:"industry"`
	Limit    string `json:"limit"`
	Offset   string `json:"offset"`
}

type BusinessCollection struct {
	Data       []BusinessAnalytics
	TotalCount int32
	Offset     int32
	Limit      int32
}

func (r *PostgresRepository) GetPremiumPartners(ctx context.Context, p SearchPremiumPartnerPayload) (*BusinessCollection, error) {

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

	// conditions = []string{
	// 	"bk.active_plan = true",
	// }

	if p.Text != "" {
		conditions = append(conditions, fmt.Sprintf(`
			(
				to_tsvector('english',
				coalesce(bk.display_name, '') || ' ' || coalesce(bk.industries, '')
				) @@ websearch_to_tsquery('english', $%d)
				OR bk.display_name ILIKE '%%' || $%d || '%%'
				OR bk.industries   ILIKE '%%' || $%d || '%%'
			)
			`, argIdx, argIdx, argIdx))

		args = append(args, p.Text)
		argIdx++
	}

	if p.Industry != "" {
		conditions = append(conditions, fmt.Sprintf(`
        (
            to_tsvector('english', coalesce(bk.industries, '')) @@ websearch_to_tsquery('english', $%d)
            OR bk.industries ILIKE '%%' || $%d || '%%'
        )
    `, argIdx, argIdx))

		args = append(args, p.Industry)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total results
	var total int32
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM business_kycs bk JOIN plans p ON bk.plan_id = p.id %s`, whereClause)
	if err := r.Conn.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count business_kycs: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			bk.id AS business_kyc_id,
			bk.display_name,
			bk.description,
			bk.address,
			bk.cac_number,
			bk.key_bonus,
			bk.business_registered,
			bk.verified,
			bk.active_plan,

			bk.shop_banner,
			bk.industries,
			bk.subdomain,

			bk.country_id,
			co.name AS country_name,  
			bk.state_id,
			st.name AS state_name,    
			bk.lga_id,
			lg.name AS lga_name,      

			u.id AS user_id,
			u.first_name,
			u.last_name,
			u.email,

			p.name AS plan_name,
			COUNT(i.id) AS total_inventories,
			COALESCE(AVG(ir.rating), 0) AS average_rating
		FROM
			business_kycs bk
		JOIN
			plans p ON bk.plan_id = p.id
		JOIN
			users u ON bk.user_id = u.id
		LEFT JOIN
			inventories i ON i.user_id = u.id
		LEFT JOIN
			inventory_ratings ir ON ir.inventory_id = i.id
		LEFT JOIN
			countries co ON co.id = bk.country_id
		LEFT JOIN
			states st ON st.id = bk.state_id
		LEFT JOIN
			lgas lg ON lg.id = bk.lga_id
		%s	
		GROUP BY
			bk.id, bk.display_name, bk.description, bk.address, bk.cac_number, bk.key_bonus,
			bk.business_registered, bk.verified, bk.active_plan,
			bk.country_id, co.name,
			bk.state_id, st.name,
			bk.lga_id, lg.name,
			u.id, u.first_name, u.last_name, u.email,
			p.name
		ORDER BY (LOWER(p.name) = 'free') ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.Conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []BusinessAnalytics

	for rows.Next() {

		var (
			ba BusinessAnalytics

			shopBanner      sql.NullString
			shopIndustry    sql.NullString
			shopDomain      sql.NullString
			shopDescription sql.NullString
			shopBonus       sql.NullString
		)
		err := rows.Scan(
			&ba.BusinessKycID,
			&ba.DisplayName,
			&shopDescription,
			&ba.Address,
			&ba.CacNumber,
			&shopBonus,
			&ba.BusinessRegistered,
			&ba.Verified,
			&ba.ActivePlan,

			&shopBanner,
			&shopIndustry,
			&shopDomain,

			&ba.CountryID,
			&ba.CountryName,
			&ba.StateID,
			&ba.StateName,
			&ba.LgaID,
			&ba.LgaName,

			&ba.UserID,
			&ba.FirstName,
			&ba.LastName,
			&ba.Email,

			&ba.PlanName,
			&ba.TotalInventories,
			&ba.AverageRating,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}

		if shopBanner.Valid {
			ba.ShopBanner = &shopBanner.String
		} else {
			ba.ShopBanner = nil
		}

		if shopIndustry.Valid {
			ba.Industries = &shopIndustry.String
		} else {
			ba.Industries = nil
		}
		if shopDomain.Valid {
			ba.Subdomain = &shopDomain.String
		} else {
			ba.Subdomain = nil
		}

		if shopDescription.Valid {
			ba.Description = &shopDescription.String
		} else {
			ba.Description = nil
		}

		if shopBonus.Valid {
			ba.KeyBonus = &shopBonus.String
		} else {
			ba.KeyBonus = nil
		}

		results = append(results, ba)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	log.Println(results, "THE RESULTS")
	// return results, nil

	// Return paginated result
	return &BusinessCollection{
		Data:       results,
		TotalCount: total,
		Offset:     int32(offset),
		Limit:      int32(limit),
	}, nil
}

type PremiumExtrasPayload struct {
	ActiveStores   int64 `json:"active_stores"`
	AvailableItems int64 `json:"available_items"`
	VerifiedStores int64 `json:"verified_stores"`
}

func (u *PostgresRepository) GetPremiumUsersExtras(ctx context.Context) (PremiumExtrasPayload, error) {

	var invCatCount int64
	var storeCount int64
	var vStoreCount int64
	// execute query to count in inventories where category_id matches category.ID
	invQuery := `SELECT COUNT(*) FROM inventories`
	invRow := u.Conn.QueryRowContext(ctx, invQuery)
	if err := invRow.Scan(&invCatCount); err != nil {
		log.Println("Error scanning row inventory count:", err)
		return PremiumExtrasPayload{}, err
	}

	storeQuery := `SELECT COUNT(*) FROM business_kycs`
	storeRow := u.Conn.QueryRowContext(ctx, storeQuery)
	if err := storeRow.Scan(&storeCount); err != nil {
		log.Println("Error scanning row business kycs count:", err)
		return PremiumExtrasPayload{}, err
	}

	vstoreQuery := `SELECT COUNT(*) FROM business_kycs where verified = true`
	vstoreRow := u.Conn.QueryRowContext(ctx, vstoreQuery)
	if err := vstoreRow.Scan(&vStoreCount); err != nil {
		log.Println("Error scanning row category count:", err)

		return PremiumExtrasPayload{}, err
	}
	//
	return PremiumExtrasPayload{
		ActiveStores:   storeCount,
		AvailableItems: invCatCount,
		VerifiedStores: vStoreCount,
	}, nil
}

type UserRatingAndCountReturn struct {
	AverageRating float64 `json:"average_rating"`
	Count         int32   `json:"count"`
}

func (r *PostgresRepository) UserRatingAndCount(ctx context.Context, userID string) (UserRatingAndCountReturn, error) {
	var count int32
	var averageRating float64

	ratingQuery := `SELECT COALESCE(AVG(rating), 0) AS average_rating, COALESCE(COUNT(*), 0) FROM user_ratings where user_id = $1`
	err := r.Conn.QueryRowContext(ctx, ratingQuery, userID).Scan(&averageRating, &count)
	if err != nil {
		return UserRatingAndCountReturn{}, nil
	}

	return UserRatingAndCountReturn{
		AverageRating: averageRating,
		Count:         count,
	}, nil

}

type TotalUserListingReturn struct {
	Count int32 `json:"count"`
}

func (r *PostgresRepository) TotalUserInventoryListing(ctx context.Context, userID string) (TotalUserListingReturn, error) {
	var count int32

	countQuery := `SELECT  COALESCE(COUNT(*), 0) FROM inventories where user_id = $1`
	err := r.Conn.QueryRowContext(ctx, countQuery, userID).Scan(&count)
	if err != nil {
		return TotalUserListingReturn{}, nil
	}

	log.Println(count, "the count")
	return TotalUserListingReturn{
		Count: count,
	}, nil
}

func (r *PostgresRepository) GetInventoryWithSuppliedID(ctx context.Context, inventoryId string) (*Inventory, error) {
	query := `SELECT id, created_at, updated_at FROM inventories WHERE id = $1`

	log.Println(inventoryId, "the inventory")

	row := r.Conn.QueryRowContext(ctx, query, inventoryId)

	var inv Inventory
	err := row.Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		log.Println("Error scanning row:", err)
		return nil, err
	}

	log.Printf("Fetched inventory: %+v", inv)
	return &inv, nil
}

func (u *PostgresRepository) GetSavedInventoryByUserIDAndInventoryID(ctx context.Context, userId, inventoryId string) (*SavedInventory, error) {

	query := `SELECT id, user_id, inventory_id, updated_at, created_at FROM saved_inventories WHERE user_id = $1 AND inventory_id = $2`
	row := u.Conn.QueryRowContext(ctx, query, userId, inventoryId)

	var savedInventory SavedInventory

	err := row.Scan(
		&savedInventory.ID,
		&savedInventory.UserID,
		&savedInventory.InventoryID,
		&savedInventory.UpdatedAt, // Ensure the order matches the query
		&savedInventory.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("no inventory found with ID %s", inventoryId)
			return nil, err
		}
		return nil, fmt.Errorf("error retrieving lga by ID: %w", err)
	}
	return &savedInventory, nil
}

func (r *PostgresRepository) SaveInventory(ctx context.Context, userId, inventoryId string) error {
	query := `INSERT INTO saved_inventories (user_id, inventory_id, updated_at, created_at) VALUES ($1, $2, NOW(), NOW())`
	_, err := r.Conn.ExecContext(ctx, query, userId, inventoryId)
	if err != nil {
		log.Println("THE ERROR CREATING SAVED INVENTORY", err)
		return fmt.Errorf("failed to save inventory: %v", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteSaveInventory(ctx context.Context, id, userId, inventoryId string) error {

	query := `DELETE FROM saved_inventories WHERE id = $1 AND user_id = $2 AND inventory_id= $3`
	res, err := r.Conn.ExecContext(ctx, query, id, userId, inventoryId)
	if err != nil {
		log.Println("Delete failed:", err)
		return fmt.Errorf("failed to deleted inventory: %v", err)
	}

	count, _ := res.RowsAffected()
	log.Printf("Deleted %d saved_inventory record(s)", count)

	return nil
}

func (r *PostgresRepository) GetUserSavedInventory(ctx context.Context, userId string) ([]*Inventory, error) {

	log.Println("The UserID", userId)
	query := `SELECT 
					i.id, 
					i.name, 
					i.description, 
					i.user_id, 
					i.category_id, 
					i.subcategory_id, 
					i.promoted, 
					i.deactivated, 
					i.updated_at, 
					i.created_at, 
					i.country_id, 
					i.state_id, 
					i.lga_id, 
					i.slug, 
					i.ulid, 
					i.offer_price, 
					i.state_slug, 
					i.country_slug, 
					i.lga_slug, 
					i.category_slug, 
					i.subcategory_slug, 
					i.product_purpose, 
					i.quantity, 
					i.is_available, 
					i.rental_duration, 
					i.security_deposit, 
					i.minimum_price, 
					i.metadata, 
					i.negotiable, 
					i.primary_image
				FROM saved_inventories si
				JOIN inventories i ON si.inventory_id = i.id
				WHERE si.user_id = $1`

	rows, err := r.Conn.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inventories []*Inventory

	for rows.Next() {
		var inv Inventory
		err := rows.Scan(
			&inv.ID,
			&inv.Name,
			&inv.Description,
			&inv.UserId,
			&inv.CategoryId,
			&inv.SubcategoryId,
			&inv.Promoted,
			&inv.Deactivated,
			&inv.UpdatedAt,
			&inv.CreatedAt,
			&inv.CategoryId,
			&inv.StateId,
			&inv.LgaId,
			&inv.Slug,
			&inv.Ulid,
			&inv.OfferPrice,
			&inv.StateSlug,
			&inv.CountrySlug,
			&inv.LgaSlug,
			&inv.CategorySlug,
			&inv.SubcategorySlug,
			&inv.ProductPurpose,
			&inv.Quantity,
			&inv.IsAvailable,
			&inv.RentalDuration,
			&inv.SecurityDeposit,
			&inv.MinimumPrice,
			&inv.Metadata,
			&inv.Negotiable,
			&inv.PrimaryImage,
		)
		if err != nil {
			log.Println("Error scanning inventory:", err)
			continue
		}
		inventories = append(inventories, &inv)
	}

	return inventories, nil
}

func (r *PostgresRepository) UploadProfileImage(ctx context.Context, img, userId string) error {
	_, err := r.Conn.ExecContext(ctx, `
		UPDATE users
		SET profile_img = $1
		WHERE id = $2 
	`, img, userId)
	return err
}
func (r *PostgresRepository) UploadShopBanner(ctx context.Context, img, userId string) error {
	_, err := r.Conn.ExecContext(ctx, `
		UPDATE business_kycs
		SET shop_banner = $1
		WHERE user_id = $2 
	`, img, userId)
	return err
}

func (r *PostgresRepository) DeleteChat(ctx context.Context, id, userId string) error {
	_, err := r.Conn.ExecContext(ctx, `
		UPDATE chats
		SET deleted_at = NOW()
		WHERE id = $1 AND sender_id = $2 
	`, id, userId)
	return err
}

// GetRenterKycByUserID loads a RenterKyc by user_id, joining IdentityType, User, Country, State, Lga, and Plan.
func (r *PostgresRepository) GetRenterKycByUserID(ctx context.Context, userID string) (*RenterKyc, error) {
	const q = `
			SELECT
			rk.id,
			rk.address,
			rk.uploaded_image,
			rk.identity_number,

			-- identity_type
			it.id   AS it_id,
			it.name AS it_name,

			-- user
			u.id, u.email, u.first_name, u.last_name, u.phone,
			u.verified     AS user_verified,
			u.profile_img,
			u.created_at   AS user_created_at,
			u.updated_at   AS user_updated_at,
			u.user_slug,

			-- country
			c.id           AS country_id,
			c.name         AS country_name,
			c.code         AS country_code,
			c.created_at   AS country_created,
			c.updated_at   AS country_updated,

			-- state
			s.id           AS state_id,
			s.name         AS state_name,
			s.state_slug   AS state_slug,
			s.country_id   AS state_country_id,
			s.created_at   AS state_created,
			s.updated_at   AS state_updated,

			-- lga
			l.id           AS lga_id,
			l.name         AS lga_name,
			l.lga_slug     AS lga_slug,
			l.state_id     AS lga_state_id,
			l.created_at   AS lga_created,
			l.updated_at   AS lga_updated,

			rk.verified,
			rk.active_plan,
			rk.created_at  AS rk_created,
			rk.updated_at  AS rk_updated
			FROM renter_kycs rk
			JOIN identity_types it ON it.id = rk.identity_type_id
			JOIN users         u  ON u.id  = rk.user_id
			JOIN countries     c  ON c.id  = rk.country_id
			JOIN states        s  ON s.id  = rk.state_id
			JOIN lgas          l  ON l.id  = rk.lga_id
			WHERE rk.user_id = $1;
		`

	row := r.Conn.QueryRowContext(ctx, q, userID)

	var (
		rawUploadedImage  sql.NullString
		rawIdentityNumber sql.NullString
		rawUserImg        sql.NullString
	)

	var (
		rk RenterKyc
		it IdentityType
		u  User
		c  Country
		s  State
		l  Lga
	)

	err := row.Scan(
		// renter_kyc
		&rk.ID,
		&rk.Address,
		&rawUploadedImage,
		&rawIdentityNumber,

		// identity_type
		&it.ID,
		&it.Name,

		// user
		&u.ID,
		&u.Email,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.Verified,
		&rawUserImg,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.UserSlug,

		// country
		&c.ID,
		&c.Name,
		&c.Code,
		&c.CreatedAt,
		&c.UpdatedAt,

		// state
		&s.ID,
		&s.Name,
		&s.StateSlug,
		&s.CountryID,
		&s.CreatedAt,
		&s.UpdatedAt,

		// lga
		&l.ID,
		&l.Name,
		&l.LgaSlug,
		&l.StateID,
		&l.CreatedAt,
		&l.UpdatedAt,

		// renter_kyc flags & timestamps
		&rk.Verified,
		&rk.ActivePlan,
		&rk.CreatedAt,
		&rk.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no renter KYC found for user %q", userID)
		}
		return nil, fmt.Errorf("query renter_kyc: %w", err)
	}

	// unpack nullable fields
	if rawUploadedImage.Valid {
		rk.UploadedImage = rawUploadedImage.String
	}
	if rawIdentityNumber.Valid {
		rk.IdentityNumber = rawIdentityNumber.String
	}
	if rawUserImg.Valid {
		u.ProfileImg = wrapperspb.String(rawUserImg.String)
	}

	// assemble relationships
	rk.IdentityType = &it
	rk.User = &u
	rk.Country = &c
	rk.State = &s
	rk.Lga = &l

	// set the foreign‑key IDs
	rk.IdentityTypeID = it.ID
	rk.UserID = u.ID
	rk.CountryID = c.ID
	rk.StateID = s.ID
	rk.LgaID = l.ID

	return &rk, nil
}

func (r *PostgresRepository) GetBusinessKycByUserID(ctx context.Context, userID string) (*BusinessKyc, error) {
	const q = `
			SELECT
			b.id,
			b.address,
			b.cac_number,
			b.display_name,
			b.user_id,
			b.description,
			b.key_bonus,
			b.business_registered,
			b.subdomain,
			b.country_id,
			b.state_id,
			b.lga_id,
			b.industries,

			-- user fields
			u.id, u.email, u.first_name, u.last_name, u.phone,
			u.verified AS user_verified,
			u.profile_img,
			u.created_at AS user_created_at,
			u.updated_at AS user_updated_at,
			u.user_slug,

			-- country
			c.id AS country_id, c.name AS country_name, c.code AS country_code,
			c.created_at AS country_created, c.updated_at AS country_updated,

			-- state
			s.id AS state_id, s.name AS state_name, s.state_slug,
			s.country_id AS state_country_id,
			s.created_at AS state_created, s.updated_at AS state_updated,

			-- lga
			l.id AS lga_id, l.name AS lga_name, l.lga_slug,
			l.state_id AS lga_state_id,
			l.created_at AS lga_created, l.updated_at AS lga_updated,

			b.verified,
			b.active_plan,
			b.shop_banner,
			b.created_at AS b_created_at,
			b.updated_at AS b_updated_at
			FROM business_kycs b
			JOIN users       u ON u.id      = b.user_id
			JOIN countries   c ON c.id      = b.country_id
			JOIN states      s ON s.id      = b.state_id
			JOIN lgas        l ON l.id      = b.lga_id
			WHERE b.user_id = $1;
		`

	row := r.Conn.QueryRowContext(ctx, q, userID)

	var (
		rawCac        sql.NullString
		rawUserImg    sql.NullString
		rawShopBanner sql.NullString

		bc BusinessKyc
		u  User
		c  Country
		s  State
		l  Lga
	)

	err := row.Scan(
		// business_kyc
		&bc.ID,
		&bc.Address,
		&rawCac,
		&bc.DisplayName,
		&bc.UserID,
		&bc.Description,
		&bc.KeyBonus,
		&bc.BusinessRegistered,
		&bc.Subdomain,
		&bc.StateID,
		&bc.CountryID,
		&bc.LgaID,
		&bc.Industries,

		// user
		&u.ID,
		&u.Email,
		&u.FirstName,
		&u.LastName,
		&u.Phone,
		&u.Verified,
		&rawUserImg,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.UserSlug,

		// country
		&c.ID,
		&c.Name,
		&c.Code,
		&c.CreatedAt,
		&c.UpdatedAt,

		// state
		&s.ID,
		&s.Name,
		&s.StateSlug,
		&s.CountryID,
		&s.CreatedAt,
		&s.UpdatedAt,

		// lga
		&l.ID,
		&l.Name,
		&l.LgaSlug,
		&l.StateID,
		&l.CreatedAt,
		&l.UpdatedAt,

		// business_kyc flags & banner & timestamps
		&bc.Verified,
		&bc.ActivePlan,
		&rawShopBanner,
		&bc.CreatedAt,
		&bc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("no BusinessKyc for user %q", userID)
			return nil, fmt.Errorf("no BusinessKyc for user %q", userID)
		}
		return nil, fmt.Errorf("query BusinessKyc: %w", err)
	}

	// handle nullable fields
	if rawCac.Valid {
		bc.CacNumber = &rawCac.String
	}
	if rawUserImg.Valid {
		u.ProfileImg = wrapperspb.String(rawUserImg.String)
	}
	if rawShopBanner.Valid {
		bc.ShopBanner = rawShopBanner.String
	}

	// assemble
	bc.User = &u
	bc.Country = &c
	bc.State = &s
	bc.Lga = &l
	bc.UserID = userID
	bc.CountryID = c.ID
	bc.StateID = s.ID
	bc.LgaID = l.ID

	return &bc, nil
}

func (u *PostgresRepository) GetUserWithSuppliedSlug(ctx context.Context, slug string) (*User, error) {

	query := `SELECT id, email, first_name, last_name, phone, verified, profile_img, updated_at, created_at, user_slug FROM users WHERE user_slug = $1`

	row := u.Conn.QueryRowContext(ctx, query, slug)

	var user User
	var userImg sql.NullString

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Verified,
		&userImg,
		&user.UpdatedAt,
		&user.CreatedAt,
		&user.UserSlug,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no user found with slug %s", slug)
		}
		return nil, fmt.Errorf("error retrieving user by slug: %w", err)
	}

	if userImg.Valid {
		user.ProfileImg = wrapperspb.String(userImg.String)
	} else {
		user.ProfileImg = &wrapperspb.StringValue{}
	}
	log.Println(user, "the user is here")

	return &user, nil
}

func (u *PostgresRepository) GetInventoryRatingReplies(ctx context.Context, ratingID string) ([]*InventoryRatingReply, error) {
	const query = `
        SELECT
            r.id,
            r.rating_id,
            r.replier_id,
            r.comment,
            r.created_at      AS reply_created_at,
            r.updated_at      AS reply_updated_at,
            u.id              AS user_id,
            u.email           AS user_email,
            u.first_name      AS user_first_name,
            u.last_name       AS user_last_name,
            u.phone           AS user_phone,
            u.verified        AS user_verified,
            u.profile_img     AS user_profile_img,
            u.created_at      AS user_created_at,
            u.updated_at      AS user_updated_at,
            u.user_slug       AS user_slug
        FROM inventory_rating_replies AS r
        JOIN users AS u
          ON r.replier_id = u.id
        WHERE r.rating_id = $1
        ORDER BY r.created_at ASC;
    `

	rows, err := u.Conn.QueryContext(ctx, query, ratingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize to an empty slice so we never return nil
	replies := make([]*InventoryRatingReply, 0, 8)

	for rows.Next() {
		var (
			reply   InventoryRatingReply
			imgNull sql.NullString
		)
		if err := rows.Scan(
			&reply.ID,
			&reply.RatingID,
			&reply.ReplierID,
			&reply.Comment,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.ReplierDetails.ID,
			&reply.ReplierDetails.Email,
			&reply.ReplierDetails.FirstName,
			&reply.ReplierDetails.LastName,
			&reply.ReplierDetails.Phone,
			&reply.ReplierDetails.Verified,
			&imgNull, // nullable profile_img
			&reply.ReplierDetails.CreatedAt,
			&reply.ReplierDetails.UpdatedAt,
			&reply.ReplierDetails.UserSlug,
		); err != nil {
			return nil, err
		}

		if imgNull.Valid {
			reply.ReplierDetails.ProfileImg = wrapperspb.String(imgNull.String)
		} else {
			reply.ReplierDetails.ProfileImg = nil
		}

		replies = append(replies, &reply)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Even if len(replies)==0, this returns [](*InventoryRatingReply){}, not nil
	return replies, nil
}

func (u *PostgresRepository) GetUserRatingReplies(ctx context.Context, ratingID string) ([]*UserRatingReply, error) {
	const query = `
        SELECT
            r.id,
            r.rating_id,
            r.replier_id,
            r.comment,
            r.created_at      AS reply_created_at,
            r.updated_at      AS reply_updated_at,
            u.id              AS user_id,
            u.email           AS user_email,
            u.first_name      AS user_first_name,
            u.last_name       AS user_last_name,
            u.phone           AS user_phone,
            u.verified        AS user_verified,
            u.profile_img     AS user_profile_img,
            u.created_at      AS user_created_at,
            u.updated_at      AS user_updated_at,
            u.user_slug       AS user_slug
        FROM user_rating_replies AS r
        JOIN users AS u
          ON r.replier_id = u.id
        WHERE r.rating_id = $1
        ORDER BY r.created_at ASC;
    `

	rows, err := u.Conn.QueryContext(ctx, query, ratingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize to an empty slice so we never return nil
	replies := make([]*UserRatingReply, 0, 8)

	for rows.Next() {
		var (
			reply   UserRatingReply
			imgNull sql.NullString
		)
		if err := rows.Scan(
			&reply.ID,
			&reply.RatingID,
			&reply.ReplierID,
			&reply.Comment,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.ReplierDetails.ID,
			&reply.ReplierDetails.Email,
			&reply.ReplierDetails.FirstName,
			&reply.ReplierDetails.LastName,
			&reply.ReplierDetails.Phone,
			&reply.ReplierDetails.Verified,
			&imgNull, // nullable profile_img
			&reply.ReplierDetails.CreatedAt,
			&reply.ReplierDetails.UpdatedAt,
			&reply.ReplierDetails.UserSlug,
		); err != nil {
			return nil, err
		}

		if imgNull.Valid {
			reply.ReplierDetails.ProfileImg = wrapperspb.String(imgNull.String)
		} else {
			reply.ReplierDetails.ProfileImg = nil
		}

		replies = append(replies, &reply)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Even if len(replies)==0, this returns [](*InventoryRatingReply){}, not nil
	return replies, nil
}
