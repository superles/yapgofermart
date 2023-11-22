package model

import "time"

type Balance struct {
	OrderNumber    string    `json:"order_number"`
	UserId         int64     `json:"user_id"`
	CurrentBalance float64   `json:"current_balance"`
	Accrual        float64   `json:"accrual,omitempty"`
	Withdraw       float64   `json:"withdraw,omitempty"`
	ProcessedAt    time.Time `json:"processed_at"`
}
