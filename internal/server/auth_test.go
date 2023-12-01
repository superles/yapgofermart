package server

import (
	"github.com/stretchr/testify/assert"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/memstorage"
	"github.com/valyala/fasthttp"
	"testing"
)

func TestServer_registerUserHandler(t *testing.T) {
	type fields struct {
		cfg     *config.Config
		storage storage.Storage
	}
	type args struct {
		body       string
		statusCode int
	}

	memStorage, err := memstorage.NewStorage()

	if err != nil {
		t.Fatalf("ошибка инициализации ")
	}

	cfg, err := config.New()

	if err != nil {
		t.Fatalf("ошибка инициализации конфига")
	}

	s := &Server{
		cfg:     cfg,
		storage: memStorage,
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "#1 positive register",

			args: args{`{ "login":"user", "password":"pass" }`, fasthttp.StatusOK},
		},
		{
			name: "#2 user exists",

			args: args{`{ "login":"user", "password":"pass" }`, fasthttp.StatusConflict},
		},
		{
			name: "#3 bad request",

			args: args{`{`, fasthttp.StatusBadRequest},
		},
		{
			name: "#4 positive register",

			args: args{`{ "login":"user1", "password":"pass" }`, fasthttp.StatusOK},
		},
		{
			name: "#5 empty password",

			args: args{`{ "login":"user2", "password":"" }`, fasthttp.StatusBadRequest},
		},
		{
			name: "#6 empty login",

			args: args{`{ "login":"", "password":"user2" }`, fasthttp.StatusBadRequest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCtx := createRequestWithBody(tt.args.body)
			s.registerUserHandler(reqCtx)
			responseCode := reqCtx.Response.StatusCode()
			assert.Equal(t, tt.args.statusCode, responseCode)
		})
	}
}

func TestServer_loginUserHandler(t *testing.T) {

	type fields struct {
		cfg     *config.Config
		storage storage.Storage
	}
	type args struct {
		body       string
		statusCode int
	}

	memStorage, err := memstorage.NewStorage()

	if err != nil {
		t.Fatalf("ошибка инициализации хранилища: %s", err.Error())
	}

	generateTestUsers(t, memStorage)

	cfg, err := config.New()

	if err != nil {
		t.Fatalf("ошибка инициализации конфига: %s", err.Error())
	}

	s := &Server{
		cfg:     cfg,
		storage: memStorage,
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "#1 positive login",

			args: args{`{ "login":"user", "password":"pass" }`, fasthttp.StatusOK},
		},
		{
			name: "#2 wrong pass format",

			args: args{`{ "login":"user", "password":"" }`, fasthttp.StatusBadRequest},
		},
		{
			name: "#3 wrong login format",

			args: args{`{ "login":"", "password":"pass" }`, fasthttp.StatusBadRequest},
		},
		{
			name: "#4 bad request",

			args: args{`{`, fasthttp.StatusBadRequest},
		},
		{
			name: "#4 negative login",

			args: args{`{ "login":"user3", "password":"pass3" }`, fasthttp.StatusUnauthorized},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCtx := createRequestWithBody(tt.args.body)
			s.loginUserHandler(reqCtx)
			responseCode := reqCtx.Response.StatusCode()
			assert.Equal(t, tt.args.statusCode, responseCode)
		})
	}
}
