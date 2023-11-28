package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
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

// CreateWithdrawal создание записи в withdrawal таблице при условии, что пользователю достаточно баланса,
// создание записи изменения баланса + обновление баланса у пользователя
func (s *PgStorage) CreateWithdrawal(ctx context.Context, number string, withdraw float64, userID int64) error {

	if withdraw <= 0 {
		// нет ошибки, но поведение подозрительное
		logger.Log.Warn("передана нулевая или отрицательная сумма списания")
		return nil
	}

	tx, err := s.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("не удалось открыть транзакцию: %w", err)
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("rollback error: %s", err))
		}
	}(tx, ctx)

	// Выбираем заказы с определенным статусом для обновления
	row := tx.QueryRow(ctx, "select id, name, balance from users WHERE id = $1 FOR UPDATE", userID)

	item := model.User{}

	if err := row.Scan(&item.ID, &item.Name, &item.Balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// при skip будет отдана ошибка pgx.ErrNoRows
			return errs.ErrNoRows
		}
		return err
	}

	if (item.Balance - withdraw) < 0 {
		// недостаточно средств на балансе
		return errs.ErrWithdrawalNotEnoughBalance
	}

	if _, err := tx.Exec(ctx, "update users set balance=coalesce(balance, 0) - $1 where id=$2", withdraw, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "insert into withdrawals (order_number, user_id, sum) VALUES ($1, $2, $3)", number, userID, withdraw); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// AddWithdrawal добавление строчки баланса, для работы с триггером(старое)
func (s *PgStorage) AddWithdrawal(ctx context.Context, withdrawal model.Withdrawal) error {
	_, err := s.db.Exec(ctx, "insert into withdrawals (order_number, user_id, sum) VALUES ($1, $2, $3)", withdrawal.Order, withdrawal.UserID, withdrawal.Sum)
	return err
}

func (s *PgStorage) GetWithdrawnSumByUserID(ctx context.Context, userID int64) (float64, error) {
	var returnSum float64
	row := s.db.QueryRow(ctx, "select coalesce(sum(sum),0) from withdrawals where user_id = $1", userID)
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
