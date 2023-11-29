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

	appContext, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	srv := fasthttp.Server{}
	srv.Handler = router.Handler

	go func() {
		if err := srv.ListenAndServe(s.cfg.Endpoint); err != nil {
			logger.Log.Error(fmt.Sprintf("не могу запустить сервер %s: %s", s.cfg.Endpoint, err))
		}
	}()

	logger.Log.Info(fmt.Sprintf("Server started at %s", s.cfg.Endpoint))

	service := accrual.Service{Client: accrual.NewHTTPClient(s.cfg.AccrualSystemAddress), Storage: s.storage}

	service.Run(appContext, 5*time.Second)

	<-appContext.Done()

	if appContext.Err() != nil {
		logger.Log.Errorf("ошибка контескта: %s", appContext.Err())
	}

	if err := srv.Shutdown(); err != nil {
		return err
	}

	logger.Log.Info("Server graceful shutdown")

	return nil
}

func pingHandler(ctx *fasthttp.RequestCtx) {
	// Обработка ping запроса
	ctx.Response.Header.Set("Content-Type", "text/plain")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBodyString("pong")

}
