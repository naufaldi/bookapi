package crypto

import "testing"

func TestValidatePasswordStrength_ValidPasswords(t *testing.T) {
	validPasswords := []string{
		"Test123!@#",
		"Password1$",
		"SecureP@ss1",
		"Str0ng#Pass",
		"Valid123!",
	}

	for _, password := range validPasswords {
		err := ValidatePasswordStrength(password)
		if err != nil {
			t.Errorf("Password %s should be valid but got error: %v", password, err)
		}
	}
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	shortPasswords := []string{
		"Test1!",
		"Pass1",
		"Abc12",
	}

	for _, password := range shortPasswords {
		err := ValidatePasswordStrength(password)
		if err != ErrPasswordTooShort {
			t.Errorf("Expected ErrPasswordTooShort for %s, got %v", password, err)
		}
	}
}

func TestValidatePasswordStrength_NoUpperCase(t *testing.T) {
	passwords := []string{
		"test123!@#",
		"password1$",
	}

	for _, password := range passwords {
		err := ValidatePasswordStrength(password)
		if err != ErrPasswordNoUpper {
			t.Errorf("Expected ErrPasswordNoUpper for %s, got %v", password, err)
		}
	}
}

func TestValidatePasswordStrength_NoLowerCase(t *testing.T) {
	passwords := []string{
		"TEST123!@#",
		"PASSWORD1$",
	}

	for _, password := range passwords {
		err := ValidatePasswordStrength(password)
		if err != ErrPasswordNoLower {
			t.Errorf("Expected ErrPasswordNoLower for %s, got %v", password, err)
		}
	}
}

func TestValidatePasswordStrength_NoNumber(t *testing.T) {
	passwords := []string{
		"TestPass!@#",
		"Password$",
	}

	for _, password := range passwords {
		err := ValidatePasswordStrength(password)
		if err != ErrPasswordNoNumber {
			t.Errorf("Expected ErrPasswordNoNumber for %s, got %v", password, err)
		}
	}
}

func TestValidatePasswordStrength_NoSpecialChar(t *testing.T) {
	passwords := []string{
		"TestPass123",
		"Password1",
	}

	for _, password := range passwords {
		err := ValidatePasswordStrength(password)
		if err != ErrPasswordNoSpecialChar {
			t.Errorf("Expected ErrPasswordNoSpecialChar for %s, got %v", password, err)
		}
	}
}
