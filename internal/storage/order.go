package storage

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
)

type OrderStorage interface {
	GetAllNewAndProcessingOrders(ctx context.Context) ([]model.Order, error)
	GetAllOrdersByUser(ctx context.Context, userID int64) ([]model.Order, error)
	GetOrder(ctx context.Context, number string) (model.Order, error)
	CreateNewOrder(ctx context.Context, number string, userID int64) error
	UpdateOrderStatus(ctx context.Context, number string, status string) error
	SetOrderProcessedAndUserBalance(ctx context.Context, number string, sum float64) error
}
