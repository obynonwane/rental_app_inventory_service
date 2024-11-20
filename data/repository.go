package data

import (
	"context"
	"database/sql"
)

type Repository interface {
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
	GetAll(ctx context.Context) ([]*User, error)
	GetAllCategory(ctx context.Context) ([]*Category, error)
	GetAllSubCategory(ctx context.Context) ([]*Subcategory, error)
	GetcategoryByID(ctx context.Context, id string) (*Category, error)
	GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error)
	GetSubcategoryByID(ctx context.Context, id string) (*Subcategory, error)
	CreateInventory(tx *sql.Tx, ctx context.Context, name string, description string, userId string, categoryId string, subcategoryId string, urls []string) error
}
