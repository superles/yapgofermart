package accrual

import (
	"context"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"runtime"
	"sync"
	"time"
)

type Service struct {
	Storage storage.Storage
	Client  Client
}

func (s *Service) generator(ctx context.Context, ch chan<- model.Order, reportInterval time.Duration) {
	ticker := time.NewTicker(reportInterval)
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
				if err != nil {
					logger.Log.Errorf("update orders status to processing error: %s", err.Error())
					continue
				}
				ch <- order
			}

		}
	}
}

func (s *Service) worker(id int, ctx context.Context, input <-chan model.Order) {
	for {
		select {
		case <-ctx.Done():
			return // Выход из горутины при отмене контекста
		case order, ok := <-input:
			if !ok {
				logger.Log.Error("input channel closed")
				return
			}

			accrual, err := s.Client.Get(order.Number)

			if err != nil {
				logger.Log.Errorf("woker #%d, ошибка запроса сумы начисления: %s", id, err.Error())
				continue
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
		}
	}
}

func (s *Service) Run(ctx context.Context, reportInterval time.Duration) {
	var wg sync.WaitGroup
	rateLimit := runtime.GOMAXPROCS(0)
	requestChan := make(chan model.Order, rateLimit)
	go s.generator(ctx, requestChan, reportInterval)
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
