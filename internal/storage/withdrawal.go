package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type WithdrawalStorage interface {
	GetAllWithdrawalsByUserId(ctx context.Context, id int64) ([]model.Withdrawal, error)
	AddWithdrawal(ctx context.Context, withdrawal model.Withdrawal) error
	GetWithdrawnSumByUserId(ctx context.Context, userId int64) (float64, error)
}