package accrual

const (
	StatusRegistered = "REGISTERED" // StatusRegistered заказ зарегистрирован, но не начисление не рассчитано
	StatusProcessing = "PROCESSING" // StatusProcessing заказ не принят к расчёту, и вознаграждение не будет начислено
	StatusInvalid    = "INVALID"    // StatusInvalid расчёт начисления в процессе
	StatusProcessed  = "PROCESSED"  // StatusProcessed расчёт начисления окончен
)

type Accrual struct {
	Number  string   `json:"order"`  // Номер заказа
	Status  string   `json:"status"` // Статус заказа
	Accrual *float64 `json:"accrual,omitempty"`
}
