package signingpin

import (
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pin   string
		valid bool
	}{
		{"2580", true},
		{"13579", true},
		{"10293847", true},
		{"0912", true},
		{"1123", true},
		{"9870", true},
		{"", false},
		{"123", false},
		{"123456789", false},
		{"12a4", false},
		{"１２３４", false},
		{" 2580", false},
		{"-258", false},
		{"0000", false},
		{"1111", false},
		{"99999999", false},
		{"1234", false},
		{"0123", false},
		{"12345678", false},
		{"4321", false},
		{"98765", false},
		{"3210", false},
		{"8901", true},
		{"2109", true},
	}

	for _, tt := range tests {
		t.Run(tt.pin, func(t *testing.T) {
			t.Parallel()
			apiErr := Validate(tt.pin, "pin")
			if tt.valid {
				assert.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
			assert.Equal(t, "pin", apiErr.Param)
		})
	}
}
