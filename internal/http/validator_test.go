package http

import (
	"strings"
	"testing"
)

type TestStruct struct {
	Email    string `validate:"required,email"`
	Username string `validate:"required,min=3,max=50"`
	Password string `validate:"required,password_strength"`
	ISBN     string `validate:"omitempty,isbn"`
	Rating   int    `validate:"gte=1,lte=5"`
}

func TestValidateStruct_ValidInput(t *testing.T) {
	s := TestStruct{
		Email:    "test@example.com",
		Username: "testuser",
		Password: "Test123!@#",
		ISBN:     "9780123456789",
		Rating:   4,
	}

	errors := ValidateStruct(s)
	if len(errors) != 0 {
		t.Errorf("Expected no validation errors, got %d", len(errors))
	}
}

func TestValidateStruct_RequiredFields(t *testing.T) {
	s := TestStruct{}

	errors := ValidateStruct(s)
	if len(errors) == 0 {
		t.Error("Expected validation errors for required fields")
	}

	hasEmailError := false
	hasUsernameError := false
	for _, err := range errors {
		if err.Field == "email" && strings.Contains(err.Message, "required") {
			hasEmailError = true
		}
		if err.Field == "username" && strings.Contains(err.Message, "required") {
			hasUsernameError = true
		}
	}

	if !hasEmailError {
		t.Error("Expected email required error")
	}
	if !hasUsernameError {
		t.Error("Expected username required error")
	}
}

func TestValidateStruct_EmailFormat(t *testing.T) {
	s := TestStruct{
		Email: "invalid-email",
	}

	errors := ValidateStruct(s)
	hasEmailFormatError := false
	for _, err := range errors {
		if err.Field == "email" && strings.Contains(err.Message, "valid email") {
			hasEmailFormatError = true
		}
	}

	if !hasEmailFormatError {
		t.Error("Expected email format validation error")
	}
}

func TestValidateStruct_PasswordStrength(t *testing.T) {
	testCases := []struct {
		password string
		valid    bool
	}{
		{"Test123!@#", true},
		{"short", false},
		{"nouppercase123!@#", false},
		{"NOLOWERCASE123!@#", false},
		{"NoNumbers!@#", false},
		{"NoSpecial123", false},
	}

	for _, tc := range testCases {
		s := TestStruct{
			Email:    "test@example.com",
			Username: "testuser",
			Password: tc.password,
		}

		errors := ValidateStruct(s)
		hasPasswordError := false
		for _, err := range errors {
			if err.Field == "password" {
				hasPasswordError = true
				break
			}
		}

		if tc.valid && hasPasswordError {
			t.Errorf("Password %s should be valid but got error", tc.password)
		}
		if !tc.valid && !hasPasswordError {
			t.Errorf("Password %s should be invalid but no error", tc.password)
		}
	}
}

func TestValidateStruct_ISBN(t *testing.T) {
	testCases := []struct {
		isbn  string
		valid bool
	}{
		{"9780123456789", true},
		{"0123456789", true},
		{"012345678X", true},
		{"978-0-123456-78-9", true},
		{"invalid", false},
		{"12345", false},
		{"", true},
	}

	for _, tc := range testCases {
		s := TestStruct{
			Email:    "test@example.com",
			Username: "testuser",
			Password: "Test123!@#",
			ISBN:     tc.isbn,
			Rating:   4,
		}

		errors := ValidateStruct(s)
		hasISBNError := false
		for _, err := range errors {
			if err.Field == "isbn" || err.Field == "iSBN" {
				hasISBNError = true
				break
			}
		}

		if tc.valid && hasISBNError {
			t.Errorf("ISBN %s should be valid but got error: %v", tc.isbn, errors)
		}
		if !tc.valid && !hasISBNError {
			t.Errorf("ISBN %s should be invalid but no error. All errors: %v", tc.isbn, errors)
		}
	}
}

func TestValidateStruct_RatingRange(t *testing.T) {
	testCases := []struct {
		rating int
		valid  bool
	}{
		{1, true},
		{3, true},
		{5, true},
		{0, false},
		{6, false},
	}

	for _, tc := range testCases {
		s := TestStruct{
			Email:  "test@example.com",
			Rating: tc.rating,
		}

		errors := ValidateStruct(s)
		hasRatingError := false
		for _, err := range errors {
			if err.Field == "rating" {
				hasRatingError = true
				break
			}
		}

		if tc.valid && hasRatingError {
			t.Errorf("Rating %d should be valid but got error", tc.rating)
		}
		if !tc.valid && !hasRatingError {
			t.Errorf("Rating %d should be invalid but no error", tc.rating)
		}
	}
}
