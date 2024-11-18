package data

import "context"

type Repository interface {
	GetAll(ctx context.Context) ([]*User, error)
	GetAllCategory(ctx context.Context) ([]*Category, error)
	GetAllSubCategory(ctx context.Context) ([]*Subcategory, error)
	GetcategoryByID(ctx context.Context, id string) (*Category, error)
	GetcategorySubcategories(ctx context.Context, id string) ([]*Subcategory, error)
}
