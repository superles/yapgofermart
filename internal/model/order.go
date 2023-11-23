package model

import (
	"time"
)

const (
	OrderStatusNew        = "NEW"        // OrderStatusNew заказ загружен в систему, но не попал в обработку
	OrderStatusProcessing = "PROCESSING" // OrderStatusProcessing заказ загружен в систему, но не попал в обработку
	OrderStatusInvalid    = "INVALID"    // OrderStatusInvalid заказ загружен в систему, но не попал в обработку
	OrderStatusProcessed  = "PROCESSED"  // OrderStatusProcessed заказ загружен в систему, но не попал в обработку
)

type Order struct {
	Number         string     `json:"number"`            // Номер заказа
	Status         string     `json:"status"`            // Статус заказа
	Accrual        *float64   `json:"accrual,omitempty"` // Рассчитанные баллы к начислению
	UploadedAt     time.Time  `json:"uploaded_at"`       // Дата загрузки товара
	AccrualCheckAt *time.Time // Время последней проверки
	AccrualStatus  *string    // Статус последней проверки бонусов
	UserID         int64      // UserID - id пользователя заказа
}
