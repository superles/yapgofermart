package storage

type Storage interface {
	UserStorage
	OrderStorage
	BalanceStorage
	WithdrawalStorage
}
