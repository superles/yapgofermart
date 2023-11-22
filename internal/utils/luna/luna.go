package luna

import (
	"errors"
	"regexp"
	"strconv"
)

func Valid(cardNumber string) (bool, error) {
	match, err := regexp.Match("\\d+", []byte(cardNumber))
	if err != nil {
		return false, err
	}
	if !match {
		return false, errors.New("неверный формат")
	}
	// Преобразование строки в массив цифр
	var digits []int
	for _, char := range cardNumber {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			// Обработка ошибки при конвертации строки в число
			return false, errors.New("неверный формат")
		}
		digits = append(digits, digit)
	}

	// Обратный проход по массиву цифр для применения алгоритма Луна
	for i := len(digits) - 2; i >= 0; i -= 2 {
		digits[i] *= 2
		if digits[i] > 9 {
			digits[i] -= 9
		}
	}

	// Вычисление суммы всех цифр
	sum := 0
	for _, digit := range digits {
		sum += digit
	}

	// Проверка на делимость на 10
	return sum%10 == 0, nil
}
