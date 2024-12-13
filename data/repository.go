package data

import (
	"context"
	"database/sql"
)

type Repository interface {
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
	GetAll(ctx context.Context) ([]*User, error)
	GetInventoryByID(ctx context.Context, id string) (*Inventory, error)
	GetAllCategory(ctx context.Context) ([]*Category, error)
	GetAllSubCategory(ctx context.Context) ([]*Subcategory, error)
	GetcategoryByID(ctx context.Context, id string) (*Category, error)
	GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error)
	GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error)
	CreateInventory(tx *sql.Tx, ctx context.Context, name string, description string, userId string, categoryId string, subcategoryId string, urls []string) error
	CreateInventoryRating(ctx context.Context, inventoryId string, raterId string, userId string, comment string, rating int32) (*InventoryRating, error)
	CreateUserRating(ctx context.Context, userId string, rating int32, comment string, raterId string) (*UserRating, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetInventoryRatings(ctx context.Context, id string, page int32, limit int32) ([]*InventoryRating, int32, error)
	GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error)
	GetUserRatingSummary(ctx context.Context, userID string) (*RatingSummary, error)
	GetInventoryRatingSummary(ctx context.Context, inventoryID string) (*RatingSummary, error)
	CreateInventoryRatingReply(ctx context.Context, param *ReplyRatingPayload) (*InventoryRatingReply, error)
	CreateUserRatingReply(ctx context.Context, param *ReplyRatingPayload) (*UserRatingReply, error)
}
