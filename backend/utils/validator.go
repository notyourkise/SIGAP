package utils

import (
	"github.com/go-playground/validator/v10"
)

// Global validator instance
var Validate = validator.New()

// init mendaftarkan validasi custom kebijakan password (Fase 2.9):
// password harus mengandung minimal 1 huruf besar, 1 huruf kecil,
// 1 angka, dan 1 karakter khusus.
func init() {
	_ = Validate.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		pw := fl.Field().String()
		var hasUpper, hasLower, hasDigit, hasSpecial bool
		for _, r := range pw {
			switch {
			case r >= 'A' && r <= 'Z':
				hasUpper = true
			case r >= 'a' && r <= 'z':
				hasLower = true
			case r >= '0' && r <= '9':
				hasDigit = true
			default:
				hasSpecial = true
			}
		}
		return hasUpper && hasLower && hasDigit && hasSpecial
	})
}

// ValidationError mendefinisikan format error validasi per field
type ValidationError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value,omitempty"`
}

// ValidateStruct memvalidasi struct dan mengembalikan slice ValidationError jika terdapat error
func ValidateStruct(s interface{}) []ValidationError {
	var errors []ValidationError
	err := Validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ValidationError
			element.Field = err.Field()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, element)
		}
	}
	return errors
}
