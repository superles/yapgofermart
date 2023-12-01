package model

const (
	// RoleUser роль авторизированного пользователя
	RoleUser = "user"
	// RoleAdmin роль пользователя с админ правами
	RoleAdmin = "admin"
	// RoleGuest роль не авторизированного пользователя
	RoleGuest = "guest"
)

type User struct {
	ID           int64
	Name         string  // Имя пользователя
	PasswordHash string  // Хеш пароля пользователя
	Role         string  // Роль пользователя
	Balance      float64 // Роль пользователя
}
