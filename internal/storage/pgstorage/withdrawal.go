package pgstorage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
)

func (s *PgStorage) GetAllWithdrawalsByUserID(ctx context.Context, id int64) ([]model.Withdrawal, error) {

	var items []model.Withdrawal
	rows, err := s.db.Query(ctx, `select order_number, user_id, sum, processed_at from withdrawals where user_id = $1 order by processed_at desc`, id)
	if err != nil {
		return items, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Withdrawal
		err = rows.Scan(&item.Order, &item.UserID, &item.Sum, &item.ProcessedAt)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *PgStorage) AddWithdrawal(ctx context.Context, withdrawal model.Withdrawal) error {
	_, err := s.db.Exec(ctx, "insert into withdrawals (order_number, user_id, sum) VALUES ($1, $2, $3)", withdrawal.Order, withdrawal.UserID, withdrawal.Sum)
	return err
}

func (s *PgStorage) GetWithdrawnSumByUserID(ctx context.Context, userID int64) (float64, error) {
	var returnSum float64
	row := s.db.QueryRow(ctx, "select coalesce(sum(withdrawal),0) from balance where user_id = $1", userID)
	if row == nil {
		return 0, errs.ErrNoRows
	}
	if err := row.Scan(&returnSum); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errs.ErrNoRows
		}
		return 0, err
	}
	return returnSum, nil
}
