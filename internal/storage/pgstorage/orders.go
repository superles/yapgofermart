package pgstorage

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"strings"
)

func (s *PgStorage) GetOrder(ctx context.Context, number string) (model.Order, error) {

	item := model.Order{}

	row := s.db.QueryRow(ctx, `select number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id from orders where number=$1`, number)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return item, errs.ErrNoRows
		}
		return item, err
	}

	return item, nil
}

func (s *PgStorage) GetAllOrders(ctx context.Context, opts ...storage.OrderFindOption) ([]model.Order, error) {

	var items []model.Order
	queryOptions := &storage.OrderFindOptions{}
	for _, opt := range opts {
		opt(queryOptions)
	}

	query := `select number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id from orders`

	var whereClause []string
	var params []interface{}

	// Формирование WHERE части запроса на основе опций
	if len(queryOptions.Status) > 0 {
		statusesPlaceholder := make([]string, len(queryOptions.Status))
		for i, status := range queryOptions.Status {
			params = append(params, status)
			statusesPlaceholder[i] = fmt.Sprintf("$%d", len(params))
		}
		whereClause = append(whereClause, fmt.Sprintf("status IN (%s)", strings.Join(statusesPlaceholder, ",")))
	}

	if queryOptions.UserID != 0 {
		params = append(params, queryOptions.UserID)
		whereClause = append(whereClause, fmt.Sprintf("user_id = $%d", len(params)))
	}

	if len(queryOptions.Number) != 0 {
		params = append(params, queryOptions.Number)
		whereClause = append(whereClause, fmt.Sprintf("number = $%d", len(params)))
	}

	if len(whereClause) > 0 {
		query += " WHERE " + strings.Join(whereClause, " AND ")
	}

	if queryOptions.OrderBy != "" {
		if len(queryOptions.OrderDirection) == 0 {
			queryOptions.OrderDirection = "asc"
		}
		direction := strings.ToLower(queryOptions.OrderDirection)
		if direction == "asc" || direction == "desc" {
			query += fmt.Sprintf(" ORDER BY %s %s", pgx.Identifier{queryOptions.OrderBy}.Sanitize(), direction)
		}
	}

	if queryOptions.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", queryOptions.Limit)
	}

	if queryOptions.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", queryOptions.Offset)
	}

	rows, err := s.db.Query(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Order
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *PgStorage) GetAllOrdersByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	var items []model.Order
	rows, err := s.db.Query(ctx, `select number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id from orders where user_id=$1 order by uploaded_at desc`, userID)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Order
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID)
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
	rows, err := s.db.Query(ctx, `select number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id from orders where status=$1 or status=$2 order by uploaded_at asc`, model.OrderStatusNew, model.OrderStatusProcessing)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	for rows.Next() {
		var item model.Order
		err = rows.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *PgStorage) CreateNewOrderOld(ctx context.Context, number string, userID int64) error {
	row := s.db.QueryRow(ctx, "select check_and_insert_order($1, $2, $3)", number, model.OrderStatusNew, userID)

	if row == nil {
		return errs.ErrNoRows
	}

	var returnVal int64

	if err := row.Scan(&returnVal); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.ErrNoRows
		}
		return err
	}

	switch returnVal {
	case 1:
		//Если существует и пользователь совпадает
		return errs.ErrExistsSameUser
	case 2:
		//Если существует и пользователь не совпадает
		return errs.ErrExistsAnotherUser
	default:
		return nil
	}
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
	row := tx.QueryRow(ctx, "SELECT number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id FROM orders WHERE number = $1", number)

	item := model.Order{}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
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

func (s *PgStorage) AddOrder(ctx context.Context, order model.Order) error {
	_, err := s.db.Exec(ctx, "insert into orders (number, status, user_id) VALUES ($1, $2, $3)", order.Number, order.Status, order.UserID)
	return err
}

func (s *PgStorage) UpdateOrder(ctx context.Context, number string, options ...storage.OrderUpdateOption) error {

	if len(options) == 0 {
		return nil
	}

	setOptions := &storage.OrderUpdateOptions{}

	for _, opt := range options {
		opt(setOptions)
	}

	var setClause []string
	var params []interface{}

	if len(setOptions.Status) != 0 {
		params = append(params, setOptions.Status)
		setClause = append(setClause, fmt.Sprintf("status = $%d", len(params)))
	}

	if setOptions.Accrual > 0 {
		params = append(params, setOptions.Accrual)
		setClause = append(setClause, fmt.Sprintf("accrual = $%d", len(params)))
	}

	query := "update orders"

	if len(setClause) > 0 {
		query += " SET " + strings.Join(setClause, " , ")
	}
	params = append(params, number)
	query += fmt.Sprintf(" where number = $%d", len(params))
	_, err := s.db.Exec(ctx, query, params...)
	return err
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
	row := tx.QueryRow(ctx, "SELECT number, status, accrual, uploaded_at, accrual_check_at, accrual_status, user_id FROM orders WHERE number = $1 FOR UPDATE SKIP LOCKED", number)

	item := model.Order{}

	if err := row.Scan(&item.Number, &item.Status, &item.Accrual, &item.UploadedAt, &item.AccrualCheckAt, &item.AccrualStatus, &item.UserID); err != nil {
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
