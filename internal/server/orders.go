package server

import (
	"bytes"
	"encoding/json"
	"errors"
	errs "github.com/superles/yapgofermart/internal/errors"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"github.com/superles/yapgofermart/internal/utils/luna"
	"github.com/valyala/fasthttp"
	"time"
)

type OrderJSON struct {
	Number     string   `json:"number"`            // Номер заказа
	Status     string   `json:"status"`            // Статус заказа
	Accrual    *float64 `json:"accrual,omitempty"` // Рассчитанные баллы к начислению
	UploadedAt string   `json:"uploaded_at"`       // Дата загрузки товара
}

func (s *Server) createOrderHandlerOld(ctx *fasthttp.RequestCtx) {

	contentType := ctx.Request.Header.ContentType()
	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if !bytes.Contains(contentType, []byte("text/plain")) {
		logger.Log.Errorf("неверный формат запроса: %s", string(contentType))
		ctx.Error("неверный формат запроса", fasthttp.StatusBadRequest)
		return
	}

	// Обработка создания заказа
	body := ctx.Request.Body()
	strBody := string(body)

	if len(body) > 255 {
		logger.Log.Errorf("номер заказа превысил длину: %s", strBody)
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	}
	if isLunaValid, err := luna.Valid(strBody); err != nil {
		logger.Log.Errorf("номер не соответствует алгоритму luna %s", err.Error())
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	} else if !isLunaValid {
		logger.Log.Errorf("номер не соответствует алгоритму luna: %s", strBody)
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	}

	newOrder, err := s.storage.GetOrder(ctx, strBody)

	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		logger.Log.Errorf("ошибка запроса заказа %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if len(newOrder.Number) == 0 {
		if err := s.storage.AddOrder(ctx, model.Order{Number: strBody, Status: model.OrderStatusNew, UserID: userID}); err != nil {
			logger.Log.Errorf("ошибка добавления заказа: %s", err.Error())
			ctx.Error("ошибка добавления заказа", fasthttp.StatusInternalServerError)
		}
		ctx.Response.SetStatusCode(202)
		ctx.Response.SetBody(body)
		return
	}

	if newOrder.UserID == userID {
		ctx.SetBodyString("номер заказа уже был загружен этим пользователем")
		logger.Log.Infof("номер заказа уже был загружен этим пользователем: %s", strBody)
		ctx.SetStatusCode(fasthttp.StatusOK)
	} else {
		ctx.SetBodyString("номер заказа уже был загружен другим пользователем")
		logger.Log.Infof("номер заказа уже был загружен другим пользователем: %s", strBody)
		ctx.SetStatusCode(fasthttp.StatusConflict)
	}

}

func (s *Server) createOrderHandler(ctx *fasthttp.RequestCtx) {

	contentType := ctx.Request.Header.ContentType()
	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if !bytes.Contains(contentType, []byte("text/plain")) {
		logger.Log.Errorf("неверный формат запроса: %s", string(contentType))
		ctx.Error("неверный формат запроса", fasthttp.StatusBadRequest)
		return
	}

	// Обработка создания заказа
	body := ctx.Request.Body()
	orderNumber := string(body)

	if len(body) > 255 {
		logger.Log.Errorf("номер заказа превысил длину: %s", orderNumber)
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	}
	if isLunaValid, err := luna.Valid(orderNumber); err != nil {
		logger.Log.Errorf("номер не соответствует алгоритму luna %s", err.Error())
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	} else if !isLunaValid {
		logger.Log.Errorf("номер не соответствует алгоритму luna: %s", orderNumber)
		ctx.Error("неверный формат номера заказа", fasthttp.StatusUnprocessableEntity)
		return
	}

	err := s.storage.CreateNewOrder(ctx, orderNumber, userID)

	if err == nil {
		ctx.Response.SetStatusCode(202)
		ctx.Response.SetBody(body)
		return
	}

	if errors.Is(err, errs.ErrExistsAnotherUser) {
		ctx.SetBodyString("номер заказа уже был загружен другим пользователем")
		logger.Log.Infof("номер заказа уже был загружен другим пользователем: %s", orderNumber)
		ctx.SetStatusCode(fasthttp.StatusConflict)
	} else if errors.Is(err, errs.ErrExistsSameUser) {
		ctx.SetBodyString("номер заказа уже был загружен этим пользователем")
		logger.Log.Infof("номер заказа уже был загружен этим пользователем: %s", orderNumber)
		ctx.SetStatusCode(fasthttp.StatusOK)
	} else {
		ctx.SetBodyString("ошибка сервера")
		logger.Log.Errorf("ошибка сервера %s", err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
	}
}

func (s *Server) getOrdersHandler(ctx *fasthttp.RequestCtx) {
	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}
	orders, err := s.storage.GetAllOrdersByUser(ctx, userID)
	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		logger.Log.Errorf("ошибка запроса заказов %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		ctx.Response.SetStatusCode(204)
		return
	}

	jsonOrders := make([]OrderJSON, len(orders))
	for i, order := range orders {
		jsonOrders[i] = OrderJSON{
			Number:     order.Number,
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadedAt: order.UploadedAt.Format(time.RFC3339),
		}
	}
	if data, err := json.Marshal(jsonOrders); err != nil {
		logger.Log.Errorf("ошибка запроса сериализации %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetStatusCode(200)
		ctx.Response.SetBody(data)
	}
}
