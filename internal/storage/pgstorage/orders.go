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

func (s *PgStorage) GetOrder(ctx context.Context, number string) (model.Order, error) {

	item := model.Order{}

	row := s.db.QueryRow(ctx, `select number, status, accrual, uploaded_at, user_id from orders where number=$1`, number)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return item, errs.ErrNoRows
		}
		return item, err
	}

	return item, nil
}

func (s *PgStorage) GetAllOrdersByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	var items []model.Order
	rows, err := s.db.Query(ctx, `select number, status, accrual, uploaded_at, user_id from orders where user_id=$1 order by uploaded_at desc`, userID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Order
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.UserID)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

// GetAllNewAndProcessingOrders получение всех заказов со статусами NEW и PROCESSING для запроса/повторного запроса в системе лояльности(accrual)
func (s *PgStorage) GetAllNewAndProcessingOrders(ctx context.Context) ([]model.Order, error) {
	var items []model.Order
	rows, err := s.db.Query(ctx, `select number, status, accrual, uploaded_at, user_id from orders where status=$1 or status=$2 order by uploaded_at asc`, model.OrderStatusNew, model.OrderStatusProcessing)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Order
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.UserID)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *PgStorage) CreateNewOrder(ctx context.Context, number string, userID int64) error {

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
	row := tx.QueryRow(ctx, "SELECT number, status, accrual, uploaded_at, user_id FROM orders WHERE number = $1", number)

	item := model.Order{}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.UserID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if len(item.Number) > 0 && item.UserID != userID {
		//Если существует и пользователь не совпадает
		return errs.ErrExistsAnotherUser
	} else if len(item.Number) > 0 && item.UserID == userID {
		//Если существует и пользователь совпадает
		return errs.ErrExistsSameUser
	}

	if _, err := tx.Exec(ctx, "insert into orders (number, status, user_id) VALUES ($1, $2, $3)", number, model.OrderStatusNew, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *PgStorage) UpdateOrderStatus(ctx context.Context, number string, status string) error {
	_, err := s.db.Exec(ctx, "update orders set status=$1 where number=$2", number, status)
	return err
}

func (s *PgStorage) SetOrderProcessedAndUserBalance(ctx context.Context, number string, sum float64) error {

	if sum < 0 {
		return errors.New("невозможно начислить отрицательную сумму в качестве бонусов")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("UpdateOrderStatusAndAccrual. не удалось открыть транзакцию: %w", err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("rollback error: %s", err))
		}
	}(tx, ctx)

	// выбираем заказ для обновления, select for update skip locked - для невозможности параллельной обработки ни горутиной ни другим инстансом
	row := tx.QueryRow(ctx, "SELECT number, status, accrual, uploaded_at, user_id FROM orders WHERE number = $1 FOR UPDATE SKIP LOCKED", number)

	item := model.Order{}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.ErrNoRows
		}
		return err
	}

	if item.Status == model.OrderStatusProcessed || item.Status == model.OrderStatusInvalid {
		return errs.ErrNoRows
	}

	if _, err := tx.Exec(ctx, "update orders set status=$1, accrual=$2 where number=$3", model.OrderStatusProcessed, sum, number); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, "update users set balance=coalesce(balance, 0) + $1 where id=$2", sum, item.UserID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
