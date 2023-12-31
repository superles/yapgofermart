package accrual

import (
	"context"
	"fmt"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"runtime"
	"sync"
	"time"
)

type Service struct {
	Storage      storage.Storage
	Client       Client
	PoolInterval time.Duration
}

func (s *Service) generator(ctx context.Context, ch chan<- model.Order) {
	ticker := time.NewTicker(s.PoolInterval)
	defer func() {
		ticker.Stop()
		logger.Log.Debug("stop generator ticker and close input channel")
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := s.Storage.GetAllNewAndProcessingOrders(ctx)
			if err != nil {
				logger.Log.Errorf("generator GetAll error: %s", err.Error())
				continue
			}
			for _, order := range orders {
				ch <- order
			}

		}
	}
}

func (s *Service) ProcessOrder(ctx context.Context, order model.Order, id int) error {

	accrual, err := s.Client.Get(order.Number)

	if err != nil {
		return fmt.Errorf("woker #%d, ошибка запроса сумы начисления: %s", id, err.Error())
	}

	var status string

	switch accrual.Status {
	case StatusRegistered, StatusProcessing:
		status = model.OrderStatusProcessing
		err = s.Storage.UpdateOrderStatus(ctx, accrual.Number, status)
	case StatusInvalid:
		status = model.OrderStatusInvalid
		err = s.Storage.UpdateOrderStatus(ctx, accrual.Number, status)
	case StatusProcessed:
		status = model.OrderStatusProcessed
		if accrual.Accrual != nil && *accrual.Accrual > 0 {
			err = s.Storage.SetOrderProcessedAndUserBalance(ctx, accrual.Number, *accrual.Accrual)
		} else {
			err = s.Storage.UpdateOrderStatus(ctx, accrual.Number, status)
		}
	}

	if err != nil {
		logger.Log.Errorf("woker #%d, ошибка установки статуса %s заказа %s: %s", id, status, accrual.Number, err.Error())
	}
	return nil
}

func (s *Service) worker(id int, ctx context.Context, input <-chan model.Order) {
	for {
		select {
		case <-ctx.Done():
			logger.Log.Debugf("worker %d finished, context done", id)
			return // Выход из горутины при отмене контекста
		case order, ok := <-input:
			if !ok {
				logger.Log.Error("input channel closed")
				return
			}

			if err := s.ProcessOrder(ctx, order, id); err != nil {
				logger.Log.Error(err.Error())
			}
		}
	}
}

func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	rateLimit := runtime.GOMAXPROCS(0)
	if s.PoolInterval == 0 {
		s.PoolInterval = 5 * time.Second
	}
	requestChan := make(chan model.Order, rateLimit)
	go s.generator(ctx, requestChan)
	for i := 1; i <= rateLimit; i++ {
		go func(workerID int) {
			defer wg.Done()
			s.worker(workerID, ctx, requestChan)
		}(i)
	}
	wg.Add(rateLimit)
	go func() {
		wg.Wait()
		logger.Log.Debug("free all channels")
		close(requestChan)
	}()
}
