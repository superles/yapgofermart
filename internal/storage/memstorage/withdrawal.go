package memstorage

import (
	"context"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"sort"
	"time"
)

func sortWithdrawalsByProcessedAt(orders []model.Withdrawal) {
	// Определение функции Less для интерфейса sort.Interface
	lessFunc := func(i, j int) bool {
		return orders[i].ProcessedAt.Before(orders[j].ProcessedAt)
	}

	// Использование sort.Slice для сортировки
	sort.Slice(orders, lessFunc)
}

func (s *MemStorage) GetAllWithdrawalsByUserID(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	withdrawStorageSync.RLock()
	defer withdrawStorageSync.RUnlock()
	var newCollection []model.Withdrawal
	for _, withdraw := range s.withdraws {
		if withdraw.UserID == userID {
			newCollection = append(newCollection, withdraw)
		}
	}

	sortWithdrawalsByProcessedAt(newCollection)

	return newCollection, nil
}

// CreateWithdrawal создание записи в withdrawal таблице при условии, что пользователю достаточно баланса + обновление баланса у пользователя
func (s *MemStorage) CreateWithdrawal(ctx context.Context, number string, withdraw float64, userID int64) error {

	if withdraw <= 0 {
		// нет ошибки, но поведение подозрительное
		logger.Log.Warn("передана нулевая или отрицательная сумма списания")
		return nil
	}

	userStorageSync.Lock()
	defer userStorageSync.Unlock()
	withdrawStorageSync.Lock()
	defer withdrawStorageSync.Unlock()

	var updateUser model.User
	var updateUserIndex int

	for idx, user := range s.users {
		if user.ID == userID {
			updateUser = user
			updateUserIndex = idx
			break
		}
	}

	if len(updateUser.Name) == 0 {
		return errs.ErrNoRows
	}

	if (updateUser.Balance - withdraw) < 0 {
		// недостаточно средств на балансе
		return errs.ErrWithdrawalNotEnoughBalance
	}

	updateUser.Balance -= withdraw
	s.users[updateUserIndex] = updateUser

	s.withdraws = append(s.withdraws, model.Withdrawal{UserID: userID, Order: number, Sum: withdraw, ProcessedAt: time.Now()})

	return nil
}

func (s *MemStorage) GetWithdrawnSumByUserID(ctx context.Context, userID int64) (float64, error) {
	withdrawStorageSync.RLock()
	defer withdrawStorageSync.RUnlock()
	var returnSum float64
	for _, withdraw := range s.withdraws {
		if withdraw.UserID == userID {
			returnSum += withdraw.Sum
		}
	}
	return returnSum, nil
}
