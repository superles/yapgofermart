package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func parseFlags() Config {

	var config Config

	flag.StringVar(&config.Endpoint, "a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	flag.StringVar(&config.LogLevel, "v", "info", "уровень логирования")
	flag.StringVar(&config.AccrualSystemAddress, "f", filepath.Join(os.TempDir(), "metrics-db.json"), "интервал сохранения метрик на диск")
	//example: postgresql://test_user:test_user@localhost/test_db
	flag.StringVar(&config.DatabaseDsn, "d", "", "строка подключения к базе данных в формате dsn")
	//todo для отладки, убрать. небезопасно передавать ключ в строке запуска и держать значение по умолчанию
	flag.StringVar(&config.SecretKey, "s", "secretKey", "секретный ключ для авторизации")

	var Usage = func() {
		_, err := fmt.Fprintf(flag.CommandLine.Output(), "Параметры командной строки сервера:\n")
		if err != nil {
			fmt.Println(err.Error())
		}
		flag.PrintDefaults()
	}
	flag.Usage = Usage
	flag.Parse()

	return config
}
