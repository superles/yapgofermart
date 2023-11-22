package accrual

import (
	"context"
	"errors"
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

type response struct {
	Status   int
	Error    error
	WorkerID int
	Accrual  Accrual
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
			orders, err := s.Storage.GetAllOrders(ctx, storage.WithFindStatus(model.OrderStatusNew, model.OrderStatusProcessing), storage.WithFindSortBy("uploaded_at", "asc"))
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

func (s *Service) dispatcher(ctx context.Context, input <-chan model.Order, out chan<- model.Order) {
	for {
		select {
		case <-ctx.Done():
			return // Выход из горутины при отмене контекста
		case order, ok := <-input:
			if !ok {
				logger.Log.Debug("dispatcher input channel closed")
				return
			}

			out <- order
		}
	}
}

func (s *Service) worker(id int, ctx context.Context, input <-chan model.Order, results chan<- response) {
	for {
		select {
		case <-ctx.Done():
			return // Выход из горутины при отмене контекста
		case order, ok := <-input:
			if !ok {
				results <- response{WorkerID: id, Error: errors.New("input channel closed")}
				return
			}

			accrual, err := s.Client.Get(order.Number)

			if err != nil {
				results <- response{WorkerID: id, Error: err}
				logger.Log.Errorf("ошибка запроса сумы начисления: %s", err.Error())
				continue
			}

			results <- response{WorkerID: id, Error: err, Accrual: accrual}
		}
	}
}

func (s *Service) resultProcessing(ctx context.Context, resultChan <-chan response) {
	for resp := range resultChan {

		if resp.Error != nil {
			logger.Log.Errorf("worker error %d: %s", resp.WorkerID, resp.Error.Error())
			continue
		}

		accrual := resp.Accrual

		var err error
		var status string

		switch accrual.Status {
		case StatusRegistered, StatusProcessing:
			status = model.OrderStatusProcessing
			err = s.Storage.UpdateOrder(ctx, resp.Accrual.Number, storage.WithUpdateStatus(status))
		case StatusInvalid:
			status = model.OrderStatusInvalid
			err = s.Storage.UpdateOrder(ctx, resp.Accrual.Number, storage.WithUpdateStatus(status))
		case StatusProcessed:
			status = model.OrderStatusProcessed
			options := []storage.OrderUpdateOption{storage.WithUpdateStatus(status)}
			if accrual.Accrual != nil && *accrual.Accrual > 0 {
				options = append(options, storage.WithUpdateAccrual(*accrual.Accrual))
			}
			err = s.Storage.UpdateOrder(ctx, resp.Accrual.Number, options...)
		}

		if err != nil {
			logger.Log.Errorf("ошибка установки статуса %s заказа %s: %s", status, resp.Accrual.Number, err.Error())
		}
	}
}

func (s *Service) Run(ctx context.Context, reportInterval time.Duration) {
	var wg sync.WaitGroup
	rateLimit := runtime.GOMAXPROCS(0)
	requestChan := make(chan model.Order)
	go s.generator(ctx, requestChan, reportInterval)
	dispatcherChan := make(chan model.Order, rateLimit)
	go s.dispatcher(ctx, requestChan, dispatcherChan)
	resultChan := make(chan response, rateLimit)
	for i := 1; i <= int(rateLimit); i++ {
		go func(workerID int) {
			defer wg.Done()
			s.worker(workerID, ctx, dispatcherChan, resultChan)
		}(i)
	}
	wg.Add(rateLimit)
	go func() {
		wg.Wait()
		logger.Log.Debug("free all channels")
		close(requestChan)
		close(dispatcherChan)
		close(resultChan)
	}()

	go s.resultProcessing(ctx, resultChan)
}
