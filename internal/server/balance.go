package server

import (
	"bytes"
	"encoding/json"
	"github.com/superles/yapgofermart/internal/model"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"github.com/superles/yapgofermart/internal/utils/luna"
	"github.com/valyala/fasthttp"
	"time"
)

type balanceResponse struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type balanceWithdrawRequest struct {
	Order     string  `json:"order"`
	Withdrawn float64 `json:"sum"`
}

func (s *Server) getUserBalanceHandler(ctx *fasthttp.RequestCtx) {
	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}
	withdrawnSum, err := s.storage.GetWithdrawnSumByUserID(ctx, userID)
	if err != nil {
		logger.Log.Errorf("ошибка получения суммы списаных баллов: %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	user, err := s.storage.GetUserByID(ctx, userID)

	if err != nil {
		logger.Log.Errorf("ошибка получения пользователя %d: %s", userID, err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	response := balanceResponse{Current: user.Balance, Withdrawn: withdrawnSum}

	if data, err := json.Marshal(response); err != nil {
		logger.Log.Errorf("ошибка запроса сериализации %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
	} else {
		ctx.Response.Header.Set("Content-Type", "application/json")
		ctx.Response.SetStatusCode(200)
		ctx.Response.SetBody(data)
	}
}

func (s *Server) withdrawFromBalanceHandler(ctx *fasthttp.RequestCtx) {
	contentType := ctx.Request.Header.ContentType()
	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}
	if !bytes.Contains(contentType, []byte("application/json")) {
		logger.Log.Errorf("неверный формат запроса: %s", string(contentType))
		ctx.Error("неверный формат запроса", fasthttp.StatusBadRequest)
		return
	}

	var reqData balanceWithdrawRequest

	err := json.Unmarshal(ctx.Request.Body(), &reqData)

	if err != nil {
		logger.Log.Errorf("ошибка декода запроса: %s", err.Error())
		ctx.Error("неверный формат запроса", fasthttp.StatusBadRequest)
		return
	}

	orderNumber := reqData.Order

	if len(orderNumber) > 255 {
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

	user, err := s.storage.GetUserByID(ctx, userID)

	if err != nil {
		logger.Log.Errorf("ошибка получения пользователя: %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if user.Balance < reqData.Withdrawn {
		logger.Log.Error("на счету недостаточно средств")
		ctx.Error("на счету недостаточно средств", fasthttp.StatusPaymentRequired)
		return
	}

	err = s.storage.AddWithdrawal(ctx, model.Withdrawal{Order: orderNumber, Sum: reqData.Withdrawn, UserID: userID})

	if err != nil {
		logger.Log.Errorf("ошибка добавления списания средств: %s", err.Error())
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetBodyString("списано успешно")
	logger.Log.Infof("успешно списано: заказ - %s, сумма - %f", orderNumber, reqData.Withdrawn)
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (s *Server) getUserWithdrawalsHandler(ctx *fasthttp.RequestCtx) {

	userID, ok := ctx.UserValue("userID").(int64)
	if !ok {
		logger.Log.Errorf("ошибка получения пользователя из контекста")
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	withdrawals, err := s.storage.GetAllWithdrawalsByUserID(ctx, userID)

	if err != nil {
		logger.Log.Errorf("ошибка получения выводов средств, пользователь: %d", userID)
		ctx.Error("ошибка сервера", fasthttp.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		ctx.Response.SetStatusCode(fasthttp.StatusNoContent)
		return
	}

	var outputData []model.WithdrawalJSON

	for _, w := range withdrawals {
		outputData = append(outputData, model.WithdrawalJSON{
			Order:       w.Order,
			Sum:         w.Sum,
			ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
		})
	}

	jData, err := json.Marshal(outputData)
	if err != nil {
		logger.Log.Error(err)
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.SetBody(jData)
}
