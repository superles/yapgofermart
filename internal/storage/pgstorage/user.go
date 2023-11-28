package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
)

func (s *PgStorage) GetUserByID(ctx context.Context, id int64) (model.User, error) {

	item := model.User{}

	row := s.db.QueryRow(ctx, `SELECT id, name, password_hash, role, coalesce(balance, 0) from users where id=$1`, id)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.ID, &item.Name, &item.PasswordHash, &item.Role, &item.Balance); err != nil {
		return item, err
	}

	return item, nil
}

func (s *PgStorage) GetUserByName(ctx context.Context, name string) (model.User, error) {

	item := model.User{}

	row := s.db.QueryRow(ctx, `SELECT id, name, password_hash, role, coalesce(balance, 0) from users where name=$1`, name)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.ID, &item.Name, &item.PasswordHash, &item.Role, &item.Balance); err != nil {
		return item, err
	}

	return item, nil
}

func (s *PgStorage) RegisterUser(ctx context.Context, data model.User) (model.User, error) {
	_, err := s.db.Exec(ctx, "insert into users (name, password_hash, role) VALUES ($1, $2, $3)", data.Name, data.PasswordHash, data.Role)
	if err != nil {
		return data, err
	}
	return s.GetUserByName(ctx, data.Name)
}

func (s *PgStorage) UpdateUserBalance(ctx context.Context, userID int64, sum float64) error {

	// проверяем сумму на == 0
	if sum == 0 {
		// нет ошибки, но поведение подозрительное
		logger.Log.Warn("передана нулевая сумма изменения баланса")
		return nil
	} else if sum > 0 {
		_, err := s.db.Exec(ctx, "update users set balance=coalesce(balance, 0) + $1 where id=$2", userID, sum)
		return fmt.Errorf("ошибка обновления баланса: %w", err)
	}

	tx, err := s.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("UpdateUserBalance. не удалось открыть транзакцию: %w", err)
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("rollback error: %s", err))
		}
	}(tx, ctx)

	// Выбираем заказы с определенным статусом для обновления
	row := tx.QueryRow(ctx, "select id, name, balance from users WHERE number = $1 FOR UPDATE SKIP LOCKED", userID)

	item := model.User{}

	if err := row.Scan(&item.ID, &item.Name, &item.Balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// при skip будет отдана ошибка pgx.ErrNoRows
			return errs.ErrNoRows
		}
		return err
	}

	if (item.Balance + sum) < 0 {
		// недостаточно средств на балансе
		return errs.ErrWithdrawalNotEnoughBalance
	}

	if _, err := tx.Exec(ctx, "update users set balance=coalesce(balance, 0) + $1 where id=$2", sum, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
