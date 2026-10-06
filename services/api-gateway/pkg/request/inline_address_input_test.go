package apirequest

import (
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inlineAddressRequest struct {
	BillToAddress field.Optional[InlineAddressInput] `json:"bill_to_address,omitzero"`
}

func decodeInlineAddressRequest(t *testing.T, body string) inlineAddressRequest {
	t.Helper()
	var req inlineAddressRequest
	require.NoError(t, json.Unmarshal([]byte(body), &req))
	return req
}

// A field inside the inline address is reported by its path in the body, so the caller can find it.
func TestInlineAddressInput_ValidationNamesTheNestedField(t *testing.T) {
	t.Parallel()
	req := decodeInlineAddressRequest(t, `{"bill_to_address":{"name":"Dock","country":"US","email":"not-an-email"}}`)

	apiErr := validate.Validate(&req)

	require.NotNil(t, apiErr)
	assert.Equal(t, "bill_to_address.email", apiErr.Param)
}

func TestInlineAddressToProto(t *testing.T) {
	t.Parallel()

	assert.Nil(t, InlineAddressToProto(field.None[InlineAddressInput]()), "an absent inline address sends nothing")

	got := InlineAddressToProto(decodeInlineAddressRequest(t, `{"bill_to_address":{"id":"ad_dock","type":"drop_ship","phone":null,"street_line_2":"Dock 4"}}`).BillToAddress)
	require.NotNil(t, got)
	assert.Equal(t, "ad_dock", got.GetId())
	assert.True(t, got.GetIsDropShip())
	assert.True(t, got.GetPhone().GetClear(), "null clears the phone")
	assert.Nil(t, got.Email, "an omitted field is left alone")
	assert.Equal(t, "Dock 4", got.GetStreetLine_2().GetValue())
	assert.Nil(t, got.Name)
	assert.Nil(t, got.Country)
}
