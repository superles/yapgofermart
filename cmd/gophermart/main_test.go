package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/superles/yapgofermart/internal/accrual"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/server"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/memstorage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func fillCfg(cfg *config.Config) {
	cfg.LogLevel = "debug"
	cfg.Endpoint = "localhost:33190"
	cfg.AccrualSystemAddress = "localhost:33191"
	cfg.SecretKey = "test"
	cfg.SecretKeyBytes = []byte(cfg.SecretKey)
}

func getTestUser(ctx context.Context, store storage.Storage) (model.User, error) {
	hashPass, err := server.HashPasswordWithRandomSalt("pass1")
	if err != nil {
		return model.User{}, err
	}
	registerUser, err := store.RegisterUser(ctx, model.User{Name: "user1", PasswordHash: hashPass})
	if err != nil {
		return registerUser, err
	}

	return registerUser, nil
}

type clientTest struct {
}

func (c clientTest) Get(number string) (accrual.Accrual, error) {
	sum := float64(100)
	return accrual.Accrual{Number: number, Status: accrual.StatusProcessed, Accrual: &sum}, nil
}

func TestEndToEnd(t *testing.T) {
	appContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg, err := config.New()
	if err != nil {
		log.Fatal("ошибка инициализации конфига: ", err.Error())
	}
	fillCfg(cfg)
	if err = logger.Initialize(cfg.LogLevel); err != nil {
		t.Fatal("ошибка инициализации logger: ", err.Error())
	}
	if len(cfg.Endpoint) == 0 {
		t.Fatal("не настроен адрес запуска сервера")
	}
	if len(cfg.AccrualSystemAddress) == 0 {
		t.Fatal("не настроен адрес системы расчёта")
	}

	var store storage.Storage
	if store, err = memstorage.NewStorage(); err != nil {
		t.Fatal("ошибка инициализации бд", err.Error())
	}

	testUser, err := getTestUser(appContext, store)
	if err != nil {
		t.Fatal("ошибка добавления тестового пользователя", err.Error())
	}

	service := accrual.Service{Client: clientTest{}, Storage: store, PoolInterval: 1 * time.Second}

	srv := server.New(cfg, store, service)

	token, err := srv.GetAuthToken(testUser)
	if err != nil {
		t.Fatal("ошибка добавления тестового токена", err.Error())
	}

	authHeader := fmt.Sprintf("Bearer %s", token)

	go func() {
		err = srv.Run(appContext)
		if err != nil {
			logger.Log.Errorf("ошибка инициализации сервера: %s", err.Error())
		}
	}()

	type fields struct {
		cfg     *config.Config
		storage storage.Storage
	}
	type args struct {
		body        string
		statusCode  int
		contentType string
		url         string
		method      string
		headers     map[string]string
	}

	tests := []struct {
		name      string
		fields    fields
		args      args
		checkFunc func(res *http.Response) error
	}{
		{
			name: "#1 register user positive",
			args: args{
				`{ "login":"user", "password":"pass" }`,
				fasthttp.StatusBadRequest,
				"",
				"/api/user/register",
				http.MethodPost,
				map[string]string{
					"Content-Type": "application/json",
				},
			},
			checkFunc: func(res *http.Response) error {
				if res.StatusCode != http.StatusOK {
					return fmt.Errorf("неожиданный статус, ожидается: %d, получен: %d", http.StatusOK, res.StatusCode)
				}
				_, err := store.GetUserByName(appContext, "user")
				return err
			},
		},
		{
			name: "#2 login user positive",
			args: args{
				`{ "login":"user1", "password":"pass1" }`,
				fasthttp.StatusBadRequest,
				"",
				"/api/user/login",
				http.MethodPost,
				map[string]string{
					"Content-Type":  "application/json",
					"Authorization": authHeader,
				},
			},
			checkFunc: func(res *http.Response) error {
				if res.StatusCode != http.StatusOK {
					return fmt.Errorf("неожиданный статус, ожидается: %d, получен: %d", http.StatusOK, res.StatusCode)
				}
				return nil
			},
		},
		{
			name: "#3 register order positive",
			args: args{
				`123456789049`,
				fasthttp.StatusBadRequest,
				"",
				"/api/user/orders",
				http.MethodPost,
				map[string]string{
					"Content-Type":  "text/plain",
					"Authorization": authHeader,
				},
			},
			checkFunc: func(res *http.Response) error {
				if res.StatusCode != http.StatusAccepted {
					return fmt.Errorf("неожиданный статус, ожидается: %d, получен: %d", http.StatusOK, res.StatusCode)
				}
				time.Sleep(5 * time.Second)
				usr, err := store.GetUserByName(appContext, testUser.Name)
				if err != nil {
					return err
				}
				if usr.Balance != float64(100) {
					return errors.New("ошибка начисления баланса")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := http.Client{}
			req, err := http.NewRequest(tt.args.method, fmt.Sprintf("http://%s%s", cfg.Endpoint, tt.args.url), strings.NewReader(tt.args.body))
			assert.NoError(t, err)
			if len(tt.args.headers) != 0 {
				for key, header := range tt.args.headers {
					req.Header.Set(key, header)
				}
			}
			res, err := client.Do(req)
			if err != nil {
				assert.NoError(t, err)
				return
			}
			defer res.Body.Close()
			err = tt.checkFunc(res)
			assert.NoError(t, err)
		})
	}

	logger.Log.Info("test finished")
}
