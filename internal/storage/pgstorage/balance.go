package pgstorage

import (
	"context"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/superles/yapgofermart/internal/model"
)

func (s *PgStorage) GetAllBalancesByUserId(ctx context.Context, id int64) ([]model.Balance, error) {

	var items []model.Balance
	rows, err := s.db.Query(ctx, `select order_number, user_id, current_balance, accrual, withdrawal, processed_at from balance where user_id = $1`, id)
	if err != nil {
		return items, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Balance
		err = rows.Scan(&item.OrderNumber, &item.UserId, &item.CurrentBalance, &item.Accrual, &item.Withdraw, &item.ProcessedAt)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *PgStorage) AddBalance(ctx context.Context, balance model.Balance) error {
	_, err := s.db.Exec(ctx,
		"insert into balance (order_number, user_id, current_balance, accrual, withdrawal) VALUES ($1, $2, $3, $4, $5)",
		balance.OrderNumber, balance.UserId, balance.CurrentBalance, balance.Accrual, balance.Withdraw)
	return err
}
