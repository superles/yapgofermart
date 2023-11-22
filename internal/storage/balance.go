package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type BalanceStorage interface {
	GetAllBalancesByUserId(ctx context.Context, id int64) ([]model.Balance, error)
	AddBalance(ctx context.Context, balance model.Balance) error
}