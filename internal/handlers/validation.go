package handlers

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	MaxNameLength = 50
	MinNameLength = 1
	MaxAmount     = 999_999_999
	MinAmount     = 1
	MaxDayOfMonth = 31
	MinDayOfMonth = 1
)

func ValidateExpenseName(name string) error {
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return fmt.Errorf("название  не может быть пустым")
	}

	if len(name) < MinNameLength {
		return fmt.Errorf("название слишком короткое (мин 1 символ)")
	}

	if len(name) > MaxNameLength {
		return fmt.Errorf("название слишком длинное ( макс %d символов)", MaxNameLength)
	}

	for _, ch := range name {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != ' ' && ch != '-' && ch != '_' {
			return fmt.Errorf("недопустимые символы в названии")
		}
	}

	return nil
}

func ValidateAmount(amount int64) error {
	if amount < MinAmount {
		return fmt.Errorf("сумма должна быть не менее %d рублей", MinAmount)
	}
	if amount > MaxAmount {
		return fmt.Errorf("сумма не должна превышать %d рублей", MaxAmount)
	}
	return nil
}

func ValidateDayOfMonth(day int) error {
	if day < MinDayOfMonth || day > MaxDayOfMonth {
		return fmt.Errorf("введите число от %d до %d", MinDayOfMonth, MaxDayOfMonth)
	}
	return nil
}
