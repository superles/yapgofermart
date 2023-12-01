package accrual

import "errors"

var (
	ErrNotRegistered   = errors.New("заказ не зарегистрирован в системе расчета")
	ErrTooManyRequests = errors.New("превышено количество запросов к сервису")
)
