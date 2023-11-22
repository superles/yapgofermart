package accrual

import (
	"encoding/json"
	"fmt"
	"github.com/superles/yapgofermart/internal/utils/logger"
	"io"
	"net/http"
)

type Client struct {
	BaseUrl string
}

func (c Client) Get(number string) (Accrual, error) {

	var orderData Accrual

	// Формирование URL для GET-запроса
	url := fmt.Sprintf("%s/api/orders/%s", c.BaseUrl, number)

	// Выполнение GET-запроса
	response, err := http.Get(url)
	if err != nil {
		return orderData, fmt.Errorf("ошибка при выполнении GET-запроса: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Log.Error("ошибка закрытия body", err)
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