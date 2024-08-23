package data

type Repository interface {
	GetAll() ([]*User, error)
}
