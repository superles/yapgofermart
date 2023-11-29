package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Client interface {
	Get(number string) (Accrual, error)
}

func NewHTTPClient(baseUrl string) Client {
	return clientHTTP{baseUrl}
}

type clientHTTP struct {
	baseUrl string
}

func (c clientHTTP) Get(number string) (Accrual, error) {

	var orderData Accrual

	// Формирование URL для GET-запроса
	url := fmt.Sprintf("%s/api/orders/%s", c.baseUrl, number)

	// Выполнение GET-запроса
	response, err := http.Get(url)
	if err != nil {
		return orderData, fmt.Errorf("ошибка при выполнении GET-запроса: %w", err)
	}
	defer response.Body.Close()

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
