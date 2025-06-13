package data

import (
	"context"
	"database/sql"
)

type Repository interface {
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
	GetAll(ctx context.Context) ([]*User, error)
	GetInventoryByID(ctx context.Context, inventory_id string) (*Inventory, error)
	GetCountryByID(ctx context.Context, country_id string) (*Country, error)
	GetStateByID(ctx context.Context, state_id string) (*State, error)
	GetLgaByID(ctx context.Context, lga_id string) (*Lga, error)
	GetInventoryByIDOrSlug(ctx context.Context, slug_ulid, inventory_id string) (*Inventory, error)
	GetAllCategory(ctx context.Context) ([]*Category, error)
	GetAllSubCategory(ctx context.Context) ([]*Subcategory, error)
	GetCategoryByID(ctx context.Context, p *GetCategoryByIDPayload) (*Category, error)
	GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error)
	GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error)
	CreateInventory(req *CreateInventoryParams) error
	CreateInventoryRating(ctx context.Context, inventoryId string, raterId string, userId string, comment string, rating int32) (*InventoryRating, error)
	CreateUserRating(ctx context.Context, userId string, rating int32, comment string, raterId string) (*UserRating, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetInventoryRatings(ctx context.Context, id string, page int32, limit int32) ([]*InventoryRating, int32, error)
	GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error)
	GetUserRatingSummary(ctx context.Context, userID string) (*RatingSummary, error)
	GetInventoryRatingSummary(ctx context.Context, inventoryID string) (*RatingSummary, error)
	CreateInventoryRatingReply(ctx context.Context, param *ReplyRatingPayload) (*InventoryRatingReply, error)
	CreateUserRatingReply(ctx context.Context, param *ReplyRatingPayload) (*UserRatingReply, error)
	SearchInventory(ctx context.Context, param *SearchPayload) (*InventoryCollection, error)
	CreateBooking(ctx context.Context, param *CreateBookingPayload) (*InventoryBooking, error)
}
