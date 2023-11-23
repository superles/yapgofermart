package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const saltSize = 16

// GenerateSalt генерирует случайную соль
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// HashPasswordWithRandomSalt хэширует пароль с использованием random соли и SHA256
func HashPasswordWithRandomSalt(password string) (string, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return "", err
	}
	return HashPassword(password, salt), nil
}

// HashPassword хеширует пароль с использованием соли и SHA256
func HashPassword(password string, salt []byte) string {
	hash := sha256.New()
	hash.Write(salt)
	hash.Write([]byte(password))
	hashed := hash.Sum(nil)
	saltEncoded := hex.EncodeToString(salt)
	return fmt.Sprintf("%s%s", saltEncoded, hex.EncodeToString(hashed))
}

// ValidatePassword проверяет, соответствует ли хешированный пароль и исходный пароль
func ValidatePassword(hashedPassword, password string) (bool, error) {
	decoded, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return false, err
	}
	salt := decoded[:saltSize]
	expectedHash := HashPassword(password, salt)
	return hashedPassword == expectedHash, nil
}
