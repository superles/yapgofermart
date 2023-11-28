package memstorage

import (
	"context"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"sort"
	"time"
)

func (s *MemStorage) GetOrder(ctx context.Context, number string) (model.Order, error) {
	orderStorageSync.RLock()
	defer orderStorageSync.RUnlock()
	for _, order := range s.orders {
		if order.Number == number {
			return order, nil
		}
	}

	return model.Order{}, errs.ErrNoRows
}

func sortOrderByUploadedAt(orders []model.Order) {
	// Определение функции Less для интерфейса sort.Interface
	lessFunc := func(i, j int) bool {
		return orders[i].UploadedAt.Before(orders[j].UploadedAt)
	}

	// Использование sort.Slice для сортировки
	sort.Slice(orders, lessFunc)
}

func (s *MemStorage) GetAllOrdersByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	orderStorageSync.RLock()
	defer orderStorageSync.RUnlock()
	var newCollection []model.Order
	for _, order := range s.orders {
		if order.UserID == userID {
			newCollection = append(newCollection, order)
		}
	}

	sortOrderByUploadedAt(newCollection)

	return newCollection, nil
}

// GetAllNewAndProcessingOrders получение всех заказов со статусами NEW и PROCESSING для запроса/повторного запроса в системе лояльности(accrual)
func (s *MemStorage) GetAllNewAndProcessingOrders(ctx context.Context) ([]model.Order, error) {
	orderStorageSync.RLock()
	defer orderStorageSync.RUnlock()
	var newCollection []model.Order
	for _, order := range s.orders {
		if order.Status == model.OrderStatusNew || order.Status == model.OrderStatusProcessing {
			newCollection = append(newCollection, order)
		}
	}

	sortOrderByUploadedAt(newCollection)

	return newCollection, nil
}

func (s *MemStorage) CreateNewOrder(ctx context.Context, number string, userID int64) error {

	order, err := s.GetOrder(ctx, number)

	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		return err
	}

	if len(order.Number) > 0 && order.UserID != userID {
		//Если существует и пользователь не совпадает
		return errs.ErrExistsAnotherUser
	} else if len(order.Number) > 0 && order.UserID == userID {
		//Если существует и пользователь совпадает
		return errs.ErrExistsSameUser
	}

	orderStorageSync.Lock()
	s.orders = append(s.orders, model.Order{Number: number, UserID: userID, Status: model.OrderStatusNew, UploadedAt: time.Now()})
	orderStorageSync.Unlock()

	return nil
}

func (s *MemStorage) UpdateOrderStatus(ctx context.Context, number string, status string) error {
	orderStorageSync.Lock()
	defer orderStorageSync.Unlock()
	for idx, order := range s.orders {
		if order.Number == number {
			order.Status = status
			s.orders[idx] = order
			return nil
		}
	}
	return errs.ErrNoRows
}

func (s *MemStorage) SetOrderProcessedAndUserBalance(ctx context.Context, number string, sum float64) error {

	if sum < 0 {
		return errors.New("невозможно начислить отрицательную сумму в качестве бонусов")
	}

	orderStorageSync.Lock()
	defer orderStorageSync.Unlock()
	userStorageSync.Lock()
	defer userStorageSync.Unlock()

	var updateOrder model.Order
	var updateOrderIndex int

	for idx, order := range s.orders {
		if order.Number == number {
			updateOrder = order
			updateOrderIndex = idx
			break
		}
	}

	if len(updateOrder.Number) == 0 {
		return errs.ErrNoRows
	}

	if updateOrder.Status == model.OrderStatusProcessed || updateOrder.Status == model.OrderStatusInvalid {
		return errs.ErrNoRows
	}

	var updateUser model.User
	var updateUserIndex int

	for idx, user := range s.users {
		if user.ID == updateOrder.UserID {
			updateUser = user
			updateUserIndex = idx
			break
		}
	}

	if len(updateUser.Name) == 0 {
		return errs.ErrNoRows
	}

	updateOrder.Status = model.OrderStatusProcessed
	updateOrder.Accrual = &sum
	s.orders[updateOrderIndex] = updateOrder

	updateUser.Balance += sum
	s.users[updateUserIndex] = updateUser

	return nil
}
