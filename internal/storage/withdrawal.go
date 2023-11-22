package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type WithdrawalStorage interface {
	GetAllWithdrawalsByUserID(ctx context.Context, id int64) ([]model.Withdrawal, error)
	AddWithdrawal(ctx context.Context, withdrawal model.Withdrawal) error
	GetWithdrawnSumByUserID(ctx context.Context, userID int64) (float64, error)
}
