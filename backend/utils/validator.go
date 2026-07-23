package utils

import (
	"github.com/go-playground/validator/v10"
)

// Global validator instance
var Validate = validator.New()

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
