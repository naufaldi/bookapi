package http

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	
	validate.RegisterValidation("isbn", validateISBN)
	validate.RegisterValidation("password_strength", validatePasswordStrength)
}

func validateISBN(fl validator.FieldLevel) bool {
	isbn := fl.Field().String()
	isbn = strings.ReplaceAll(isbn, "-", "")
	isbn = strings.ReplaceAll(isbn, " ", "")
	
	if len(isbn) == 10 {
		matched, _ := regexp.MatchString(`^\d{9}[\dX]$`, isbn)
		return matched
	}
	if len(isbn) == 13 {
		matched, _ := regexp.MatchString(`^\d{13}$`, isbn)
		return matched
	}
	return false
}

func validatePasswordStrength(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}
	
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)
	
	return hasUpper && hasLower && hasNumber && hasSpecial
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ValidateStruct(s interface{}) []ValidationError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var errors []ValidationError
	for _, err := range err.(validator.ValidationErrors) {
		field := err.Field()
		tag := err.Tag()
		param := err.Param()

		var message string
		switch tag {
		case "required":
			message = fmt.Sprintf("%s is required", field)
		case "email":
			message = fmt.Sprintf("%s must be a valid email address", field)
		case "min":
			message = fmt.Sprintf("%s must be at least %s characters", field, param)
		case "max":
			message = fmt.Sprintf("%s must be at most %s characters", field, param)
		case "isbn":
			message = fmt.Sprintf("%s must be a valid ISBN (10 or 13 digits)", field)
		case "password_strength":
			message = fmt.Sprintf("%s must be at least 8 characters with uppercase, lowercase, number, and special character", field)
		case "gte", "lte":
			message = fmt.Sprintf("%s must be between %s", field, param)
		default:
			message = fmt.Sprintf("%s is invalid", field)
		}

		fieldName := strings.ToLower(field[:1]) + field[1:]
		errors = append(errors, ValidationError{
			Field:   fieldName,
			Message: message,
		})
	}

	return errors
}
