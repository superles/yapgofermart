package main

import (
	"context"
	"github.com/superles/yapgofermart/internal/accrual"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/server"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/pgstorage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal("ошибка инициализации конфига: ", err.Error())
	}
	if err = logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatal("ошибка инициализации logger: ", err.Error())
	}

	if len(cfg.Endpoint) == 0 {
		log.Fatal("не настроен адрес запуска сервера")
	}

	if len(cfg.AccrualSystemAddress) == 0 {
		log.Fatal("не настроен адрес системы расчёта")
	}

	if len(cfg.DatabaseDsn) == 0 {
		log.Fatal("не настроена бд")
	}

	var store storage.Storage
	if store, err = pgstorage.NewStorage(cfg.DatabaseDsn); err != nil {
		log.Fatal("ошибка инициализации бд", err.Error())
	}

	service := accrual.Service{Client: accrual.NewHTTPClient(cfg.AccrualSystemAddress), Storage: store}

	srv := server.New(cfg, store, service)
	appContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	err = srv.Run(appContext)
	if err != nil {
		log.Fatal("ошибка запуска сервера: ", err.Error())
	}

	logger.Log.Info("app graceful shutdown")

}
