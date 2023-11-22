package main

import (
	"context"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/server"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/pgstorage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"log"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatal("ошибка инициализации конфига: ", err.Error())
	}
	if err = logger.Initialize(cfg.LogLevel); err != nil {
		log.Fatal("ошибка инициализации logger: ", err.Error())
	}
	var store storage.Storage
	if len(cfg.DatabaseDsn) != 0 {
		if store, err = pgstorage.NewStorage(cfg.DatabaseDsn); err != nil {
			log.Fatal("ошибка инициализации бд", err.Error())
		}
	}
	srv := server.New(cfg, store)
	appContext := context.Background()
	err = srv.Run(appContext)
	if err != nil {
		log.Fatal("ошибка инициализации сервера: ", err.Error())
	}
}
