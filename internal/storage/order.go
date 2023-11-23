package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type OrderUpdateOptions struct {
	Status  string
	Accrual float64
}

type OrderUpdateOption func(*OrderUpdateOptions)

func WithUpdateStatus(status string) OrderUpdateOption {
	return func(o *OrderUpdateOptions) {
		o.Status = status
	}
}

func WithUpdateAccrual(accrual float64) OrderUpdateOption {
	return func(o *OrderUpdateOptions) {
		o.Accrual = accrual
	}
}

type OrderFindOptions struct {
	Number         string
	Status         []string
	OrderBy        string
	OrderDirection string
	UserID         int64
	Limit          int64
	Offset         int64
}

type OrderFindOption func(*OrderFindOptions)

func WithFindNumber(number string) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.Number = number
	}
}

func WithFindUser(userID int64) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.UserID = userID
	}
}

func WithFindStatus(status ...string) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.Status = status
	}
}

// WithFindSortBy сортировка заказов
func WithFindSortBy(sortBy string, direction string) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.OrderBy = sortBy
		o.OrderDirection = direction
	}
}

// WithFindLimit добавить limit
func WithFindLimit(limit int64) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.Limit = limit
	}
}

// WithFindOffset  добавить offset
func WithFindOffset(offset int64) OrderFindOption {
	return func(o *OrderFindOptions) {
		o.Offset = offset
	}
}

type OrderStorage interface {
	GetAllOrders(ctx context.Context, options ...OrderFindOption) ([]model.Order, error)
	GetAllOrdersByUser(ctx context.Context, userID int64) ([]model.Order, error)
	GetOrder(ctx context.Context, number string) (model.Order, error)
	AddOrder(ctx context.Context, order model.Order) error
	CreateNewOrder(ctx context.Context, number string, userID int64) error
	UpdateOrder(ctx context.Context, number string, options ...OrderUpdateOption) error
}
