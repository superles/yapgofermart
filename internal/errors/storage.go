package errors

import "errors"

var (
	ErrNoRows            = errors.New("не найдено записей")
	ErrExistsSameUser    = errors.New("номер заказа уже был загружен этим пользователем")
	ErrExistsAnotherUser = errors.New("номер заказа уже был загружен другим пользователем")
)
