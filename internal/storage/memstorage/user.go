package memstorage

import (
	"context"
	_ "github.com/jackc/pgx/v5/stdlib"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
)

func (s *MemStorage) GetUserByID(ctx context.Context, id int64) (model.User, error) {
	userStorageSync.RLock()
	defer userStorageSync.RUnlock()
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}

	return model.User{}, errs.ErrNoRows
}

func (s *MemStorage) GetUserByName(ctx context.Context, name string) (model.User, error) {
	userStorageSync.RLock()
	defer userStorageSync.RUnlock()
	for _, user := range s.users {
		if user.Name == name {
			return user, nil
		}
	}

	return model.User{}, errs.ErrNoRows
}

func (s *MemStorage) RegisterUser(ctx context.Context, data model.User) (model.User, error) {
	userStorageSync.Lock()
	data.ID = int64(len(s.users) + 1)
	s.users = append(s.users, data)
	userStorageSync.Unlock()
	return s.GetUserByName(ctx, data.Name)
}
