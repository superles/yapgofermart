package config

import (
	"net/url"
	"strings"
	"sync"
)

type Config struct {
	Endpoint             string `env:"RUN_ADDRESS"`
	LogLevel             string `env:"SERVER_LOG_LEVEL"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	DatabaseDsn          string `env:"DATABASE_URI"`
	SecretKey            string `env:"KEY"`
	SecretKeyBytes       []byte
}

var (
	once     sync.Once
	instance Config
)

func toURL(hostWithPort string) string {
	var fullURL string

	// Проверка наличия протокола в строке, добавление http:// в случае его отсутствия
	if !strings.Contains(hostWithPort, "://") {
		hostWithPort = "http://" + hostWithPort
	}

	// Попытка разбора строки в объект URL
	parsedURL, err := url.Parse(hostWithPort)
	if err != nil {
		return "" // Возвращаем пустую строку в случае ошибки парсинга
	}

	// Получение строки URL
	fullURL = parsedURL.String()
	return fullURL
}

func New() (*Config, error) {

	var err error

	once.Do(func() {

		instance = Config{}

		flagConfig := parseFlags()
		envConfig := parseEnv()

		if len(envConfig.Endpoint) > 0 {
			instance.Endpoint = envConfig.Endpoint
		} else {
			instance.Endpoint = flagConfig.Endpoint
		}

		if len(envConfig.LogLevel) > 0 {
			instance.LogLevel = envConfig.LogLevel
		} else {
			instance.LogLevel = flagConfig.LogLevel
		}

		if len(envConfig.AccrualSystemAddress) > 0 {
			instance.AccrualSystemAddress = toURL(envConfig.AccrualSystemAddress)
		} else {
			instance.AccrualSystemAddress = toURL(flagConfig.AccrualSystemAddress)
		}

		if len(envConfig.DatabaseDsn) > 0 {
			instance.DatabaseDsn = envConfig.DatabaseDsn
		} else {
			instance.DatabaseDsn = flagConfig.DatabaseDsn
		}

		if len(envConfig.SecretKey) > 0 {
			instance.SecretKey = envConfig.SecretKey
			instance.SecretKeyBytes = []byte(instance.SecretKey)
		} else {
			//err = errors.New("не указан секретный ключ")
			//todo убрать после отладки
			instance.SecretKey = flagConfig.SecretKey
			instance.SecretKeyBytes = []byte(instance.SecretKey)
		}
	})

	return &instance, err
}
