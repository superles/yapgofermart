package main

import (
	"fmt"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"log"
)

func main() {
	log.Println("start app")
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
	}()
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

	//var _ storage.Storage
	//if _, err = pgstorage.NewStorage(cfg.DatabaseDsn); err != nil {
	//	log.Fatal("ошибка инициализации бд", err.Error())
	//}
	logger.Log.Error("конфиг:", cfg)
	//srv := server.New(cfg, store)
	//appContext := context.Background()
	//err = srv.Run(appContext)
	//if err != nil {
	//	log.Fatal("ошибка инициализации сервера: ", err.Error())
	//}
}
