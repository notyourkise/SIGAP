package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type passwordPolicyTest struct {
	Password string `validate:"required,min=8,max=100,strongpassword"`
}

func TestStrongPasswordValidator(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "valid: lengkap besar kecil angka simbol", password: "AdminPass123!", valid: true},
		{name: "valid: 8 karakter dengan seluruh syarat", password: "Ab1@defg", valid: true},
		{name: "invalid: kurang dari 8 karakter", password: "Ab1@x", valid: false},
		{name: "invalid: tanpa huruf besar", password: "adminpass123!", valid: false},
		{name: "invalid: tanpa huruf kecil", password: "ADMINPASS123!", valid: false},
		{name: "invalid: tanpa angka", password: "AdminPass@@", valid: false},
		{name: "invalid: tanpa karakter khusus", password: "AdminPass123", valid: false},
		{name: "invalid: kosong", password: "", valid: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateStruct(passwordPolicyTest{Password: tc.password})
			if tc.valid {
				assert.Empty(t, errs)
			} else {
				assert.NotEmpty(t, errs, "password '%s' seharusnya ditolak", tc.password)
			}
		})
	}
}
