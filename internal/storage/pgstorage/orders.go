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

func (s *PgStorage) CreateNewOrder(ctx context.Context, number string, userID int64) error {
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
