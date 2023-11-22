package server

import (
	"context"
	"fmt"
	fastRouter "github.com/fasthttp/router"
	"github.com/superles/yapgofermart/internal/accrual"
	"github.com/superles/yapgofermart/internal/config"
	"github.com/superles/yapgofermart/internal/storage"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"github.com/valyala/fasthttp"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Middleware func(h fasthttp.RequestHandler) fasthttp.RequestHandler

func NewMiddleware(middlewares []Middleware) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(endpoint fasthttp.RequestHandler) fasthttp.RequestHandler {
		h := middlewares[len(middlewares)-1](endpoint)
		for i := len(middlewares) - 2; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

type Server struct {
	cfg     *config.Config
	storage storage.Storage
}

func New(cfg *config.Config, s storage.Storage) *Server {
	return &Server{cfg, s}
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Другие поля пользователя
}

type Order struct {
	ID    int   `json:"id"`
	Items []int `json:"items"`
	// Другие поля заказа
}

type Balance struct {
	Amount float64 `json:"amount"`
	// Другие поля баланса
}

func withCompressMiddleware(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.CompressHandler(h)
}

func (s *Server) newRouter() *fastRouter.Router {
	router := fastRouter.New()
	withAuth := NewMiddleware([]Middleware{withCompressMiddleware, s.authMiddleware})
	noAuth := NewMiddleware([]Middleware{withCompressMiddleware})
	//router.GET("/api/ping", withAuth(withCompress(pingHandler)))
	router.GET("/api/ping", noAuth(pingHandler))
	//router.GET("/api/ping", middleware(withAuth, withCompress, pingHandler))
	router.POST("/api/user/register", noAuth(s.registerUserHandler))
	router.POST("/api/user/login", noAuth(s.loginUserHandler))
	router.POST("/api/user/orders", withAuth(s.createOrderHandler))
	router.GET("/api/user/orders", withAuth(s.getOrdersHandler))
	router.GET("/api/user/balance", withAuth(s.getUserBalanceHandler))
	router.POST("/api/user/balance/withdraw", withAuth(s.withdrawFromBalanceHandler))
	router.GET("/api/user/withdrawals", withAuth(s.getUserWithdrawalsHandler))
	return router
}

func (s *Server) Run(ctx context.Context) error {

	router := s.newRouter()

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := fasthttp.ListenAndServe(s.cfg.Endpoint, router.Handler); err != http.ErrServerClosed {
			logger.Log.Error(fmt.Sprintf("не могу запустить сервер: %s", err))
		}
	}()

	logger.Log.Info(fmt.Sprintf("Server started at %s", s.cfg.Endpoint))

	service := accrual.Service{Client: accrual.Client{BaseURL: s.cfg.AccrualSystemAddress}, Storage: s.storage}

	go service.Run(ctx, 5*time.Second)

	logger.Log.Info("Server Started")
	<-done
	logger.Log.Info("Server Stopped")
	//
	//if err := srv.Shutdown(ctx); err != nil {
	//	return err
	//}

	return nil
}

func pingHandler(ctx *fasthttp.RequestCtx) {
	// Обработка ping запроса
	ctx.Response.Header.Set("Content-Type", "text/plain")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBodyString("pong")

}
