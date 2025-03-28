package data

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (u *PostgresRepository) CreateInventory(tx *sql.Tx, ctx context.Context, name, description, userId, categoryId, subcategoryId, countryId, stateId, lgaId string, urls []string) error {

	query := `INSERT INTO inventories (name, description, user_id, category_id, subcategory_id, country_id, state_id, lga_id, updated_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()) 
			RETURNING id, name, description, user_id, category_id, subcategory_id, country_id, state_id, lga_id, updated_at, created_at`

	var inventory Inventory
	err := tx.QueryRowContext(ctx, query, name, description, userId, categoryId, subcategoryId, countryId, stateId, lgaId).Scan(
		&inventory.ID,
		&inventory.Name,
		&inventory.Description,
		&inventory.UserId,
		&inventory.CategoryId,
		&inventory.SubcategoryId,
		&inventory.CountryId,
		&inventory.StateId,
		&inventory.LgaId,
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
