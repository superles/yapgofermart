package memstorage

import (
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"sync"
)

var userStorageSync = sync.RWMutex{}
var orderStorageSync = sync.RWMutex{}
var withdrawStorageSync = sync.RWMutex{}

type MemStorage struct {
	users     []model.User
	orders    []model.Order
	withdraws []model.Withdrawal
}

func NewStorage() (storage.Storage, error) {
	return &MemStorage{}, nil
}
