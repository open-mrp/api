package apirequest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/validate"
)

func TestAddressSnapshotInput_Validate(t *testing.T) {
	t.Parallel()
	assert.Nil(t, AddressSnapshotInput{AddressID: field.Some("ad_1")}.Validate("ship_to"))
	assert.Nil(t, AddressSnapshotInput{Name: field.Some("Dock"), Country: field.Some("US")}.Validate("ship_to"))

	both := AddressSnapshotInput{AddressID: field.Some("ad_1"), City: field.Some("Columbus")}.Validate("ship_to")
	require.NotNil(t, both)
	assert.Equal(t, "ship_to", both.Param)

	missing := AddressSnapshotInput{Line1: field.Some("1 Main St")}.Validate("bill_to")
	require.NotNil(t, missing)
	assert.Equal(t, apierror.CodeValidationFailed, missing.Code)
	require.Len(t, missing.Errors, 2)
	assert.Equal(t, "bill_to.name", missing.Errors[0].Param)
	assert.Equal(t, "bill_to.country", missing.Errors[1].Param)
}

func TestAddressSnapshotInput_ToProto(t *testing.T) {
	t.Parallel()
	saved := AddressSnapshotInput{AddressID: field.Some("ad_1")}.ToProto()
	assert.Equal(t, "ad_1", saved.GetAddressId())
	assert.Empty(t, saved.GetName())

	inline := AddressSnapshotInput{Name: field.Some("Dock"), Line1: field.Some("1 Main St"), Country: field.Some("US")}.ToProto()
	assert.Nil(t, inline.AddressId)
	assert.Equal(t, "Dock", inline.GetName())
	assert.Equal(t, "1 Main St", inline.GetLine_1())
	assert.Nil(t, inline.Line_2)
	assert.Equal(t, "US", inline.GetCountry())
}

func TestEmailRecipientInput(t *testing.T) {
	t.Parallel()
	in := []EmailRecipientInput{{AccountUserID: "acus_1", Documents: []constants.RecipientDocument{constants.RecipientDocumentInvoice, constants.RecipientDocumentShipment}}}
	assert.Nil(t, validate.Validate(&in[0]))
	out := EmailRecipientsToProto(in)
	require.Len(t, out, 1)
	assert.Equal(t, "acus_1", out[0].GetAccountUserId())
	assert.Equal(t, []string{"invoice", "shipment"}, out[0].GetDocuments())

	assert.NotNil(t, validate.Validate(&EmailRecipientInput{AccountUserID: "acus_1"}), "documents are required")
}
