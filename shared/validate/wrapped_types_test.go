package validate

import (
	"testing"

	"github.com/open-mrp/api/shared/field"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type wrappedAddress struct {
	Name    string `json:"name" validate:"required"`
	Country string `json:"country" validate:"required,max=2"`
}

type wrappedAddressRequest struct {
	BillTo field.Optional[wrappedAddress] `json:"bill_to,omitzero"`
}

func init() {
	RegisterWrappedTypes(field.Optional[wrappedAddress]{})
}

func TestValidate_ChecksTheFieldsInsideASetOptionalStruct(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Validate(&wrappedAddressRequest{}), "an unset wrapper has nothing to check")
	assert.Nil(t, Validate(&wrappedAddressRequest{BillTo: field.Some(wrappedAddress{Name: "Dock", Country: "US"})}))

	apiErr := Validate(&wrappedAddressRequest{BillTo: field.Some(wrappedAddress{Name: "Dock", Country: "USA"})})
	require.NotNil(t, apiErr)
	assert.Equal(t, "bill_to.country", apiErr.Param)

	apiErr = Validate(&wrappedAddressRequest{BillTo: field.Some(wrappedAddress{Country: "US"})})
	require.NotNil(t, apiErr)
	assert.Equal(t, "bill_to.name", apiErr.Param)
}
