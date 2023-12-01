package storage

type Storage interface {
	UserStorage
	OrderStorage
	WithdrawalStorage
}
