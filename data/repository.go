package data

import "context"

type Repository interface {
	GetAll(ctx context.Context) ([]*User, error)
	GetAllCategory(ctx context.Context) ([]*Category, error)
}
