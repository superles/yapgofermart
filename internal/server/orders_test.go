package server

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/superles/yapgofermart/internal/accrual"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/storage/memstorage"
	"github.com/valyala/fasthttp"
	"testing"
	"time"
)

func TestServer_createOrderHandler(t *testing.T) {
	type fields struct {
		cfg     *config.Config
		storage storage.Storage
	}
	type args struct {
		body        string
		statusCode  int
		contentType string
	}

	memStorage, err := memstorage.NewStorage()

	if err != nil {
		t.Fatalf("ошибка инициализации хранилища: %s", err.Error())
	}

	cfg, err := config.New()

	if err != nil {
		t.Fatalf("ошибка инициализации конфига: %s", err.Error())
	}

	users := generateTestUsers(t, memStorage)
	user := users[0]

	s := &Server{
		cfg:     cfg,
		storage: memStorage,
	}

	sum := float64(700)

	client := accrual.NewMockClient(map[string][]accrual.ClientMockResponse{
		"123456789049": []accrual.ClientMockResponse{
			{Accrual: accrual.Accrual{}, Error: accrual.ErrNotRegistered},
			{Accrual: accrual.Accrual{Number: "123456789049", Status: accrual.StatusProcessed, Accrual: &sum}, Error: nil},
		},
	})

	service := accrual.Service{Client: client, Storage: s.storage}

	ctx, done := context.WithCancel(context.Background())
	defer done()

	okReq := fasthttp.Request{}
	okReq.Header.Set("Content-Type", "text/plain")

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "#1 register order negative",
			args: args{`123456789049`, fasthttp.StatusBadRequest, ""},
		},
		{
			name: "#2 register order positive",
			args: args{`123456789049`, fasthttp.StatusAccepted, "text/plain"},
		},
		{
			name: "#3 register order same user",
			args: args{`123456789049`, fasthttp.StatusOK, "text/plain"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCtx := createRequestWithBodyAndContentType(tt.args.body, tt.args.contentType)
			authCtxWithUser(reqCtx, user)
			s.createOrderHandler(reqCtx)
			_ = service.ProcessOrder(ctx, model.Order{Number: tt.args.body, Status: model.OrderStatusNew, UserID: user.ID, UploadedAt: time.Now(), Accrual: nil}, 1)
			responseCode := reqCtx.Response.StatusCode()
			assert.Equal(t, tt.args.statusCode, responseCode)
		})
	}
}
