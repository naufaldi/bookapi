package auth

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

var (
	ErrPasswordTooShort      = errors.New("password must be at least 8 characters")
	ErrPasswordNoUpper       = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower       = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber      = errors.New("password must contain at least one number")
	ErrPasswordNoSpecialChar = errors.New("password must contain at least one special character")
)

func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	if !hasUpper {
		return ErrPasswordNoUpper
	}

	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	if !hasLower {
		return ErrPasswordNoLower
	}

	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	if !hasNumber {
		return ErrPasswordNoNumber
	}

	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	if !hasSpecial {
		return ErrPasswordNoSpecialChar
	}

	return nil
}