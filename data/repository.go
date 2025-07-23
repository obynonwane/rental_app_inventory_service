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
	GetUserBySlug(ctx context.Context, slug string) (*User, error)
	GetUserWithSuppliedSlug(ctx context.Context, slug string) (*User, error)
	GetInventoryRatings(ctx context.Context, id string, page int32, limit int32) ([]*InventoryRating, int32, error)
	GetUserRatings(ctx context.Context, id string, page int32, limit int32) ([]*UserRating, int32, error)
	GetUserRatingSummary(ctx context.Context, userID string) (*RatingSummary, error)
	GetInventoryRatingSummary(ctx context.Context, inventoryID string) (*RatingSummary, error)
	CreateInventoryRatingReply(ctx context.Context, param *ReplyRatingPayload) (*InventoryRatingReply, error)
	CreateUserRatingReply(ctx context.Context, param *ReplyRatingPayload) (*UserRatingReply, error)
	SearchInventory(ctx context.Context, param *SearchPayload) (*InventoryCollection, error)
	CreateBooking(ctx context.Context, param *CreateBookingPayload) (*InventoryBooking, error)
	CreatePurchaseOrder(ctx context.Context, param *CreatePurchaseOrderPayload) (*InventorySale, error)
	SubmitChat(ctx context.Context, param *Message) (*Chat, error)
	GetChatList(ctx context.Context, userID string) ([]ChatSummary, error)
	GetChatHistory(ctx context.Context, userA, userB string) ([]Chat, error)
	GetUnreadChat(ctx context.Context, userID string) (int32, error)
	MarkChatAsRead(ctx context.Context, userID, senderID string) error
	GetPremiumPartners(ctx context.Context, req SearchPremiumPartnerPayload) (*BusinessCollection, error)
	GetPremiumUsersExtras(ctx context.Context) (PremiumExtrasPayload, error)
	UploadProfileImage(ctx context.Context, img, userId string) error
	UploadShopBanner(ctx context.Context, img, userId string) error
	UserRatingAndCount(ctx context.Context, userID string) (UserRatingAndCountReturn, error)
	TotalUserInventoryListing(ctx context.Context, userID string) (TotalUserListingReturn, error)
	SaveInventory(ctx context.Context, userId, inventoryId string) error
	DeleteSaveInventory(ctx context.Context, id, userId, inventoryId string) error
	DeleteChat(ctx context.Context, id, userId string) error
	GetSavedInventoryByUserIDAndInventoryID(ctx context.Context, userId, inventoryId string) (*SavedInventory, error)
	GetInventoryWithSuppliedID(ctx context.Context, inventoryId string) (*Inventory, error)
	GetUserSavedInventory(ctx context.Context, userId string) ([]*Inventory, error)
	GetBusinessKycByUserID(ctx context.Context, userID string) (*BusinessKyc, error)
	GetRenterKycByUserID(ctx context.Context, userID string) (*RenterKyc, error)

	GetInventoryRatingReplies(ctx context.Context, ratingID string) ([]*InventoryRatingReply, error)
	GetUserRatingReplies(ctx context.Context, ratingID string) ([]*UserRatingReply, error)
}
