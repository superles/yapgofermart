package errors

import "errors"

var (
	ErrWithdrawalNotEnoughBalance = errors.New("на счету недостаточно средств")
)
