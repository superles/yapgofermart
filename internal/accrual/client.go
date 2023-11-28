package accrual

import (
	"encoding/json"
	"fmt"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"io"
	"net/http"
)

type Client interface {
	Get(number string) (Accrual, error)
}

type ClientHTTP struct {
	BaseURL string
}

func (c ClientHTTP) Get(number string) (Accrual, error) {

	var orderData Accrual

	// Формирование URL для GET-запроса
	url := fmt.Sprintf("%s/api/orders/%s", c.BaseURL, number)

	// Выполнение GET-запроса
	response, err := http.Get(url)
	if err != nil {
		return orderData, fmt.Errorf("ошибка при выполнении GET-запроса: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Errorf("ошибка закрытия body: %s", err.Error())
		}
	}(response.Body)

	if response.StatusCode == http.StatusTooManyRequests {
		return orderData, ErrTooManyRequests
	}

	if response.StatusCode == http.StatusNoContent {
		return orderData, ErrNotRegistered
	}

	if response.StatusCode == http.StatusInternalServerError {
		return orderData, fmt.Errorf("ошибка запроса сервиса")
	}

	if response.StatusCode != http.StatusOK {
		return orderData, fmt.Errorf("неизвестный статус ответа: %d - %s", response.StatusCode, response.Status)
	}

	// Декодирование JSON-данных в структуру Order
	err = json.NewDecoder(response.Body).Decode(&orderData)

	if err != nil {
		return orderData, fmt.Errorf("ошибка декодирования JSON: %w", err)
	}

	return orderData, nil
}
