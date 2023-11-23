package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type UserStorage interface {
	GetUserByName(ctx context.Context, name string) (model.User, error)
	GetUserByID(ctx context.Context, id int64) (model.User, error)
	RegisterUser(ctx context.Context, user model.User) (model.User, error)
}
