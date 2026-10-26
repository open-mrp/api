package addressep

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	"github.com/open-mrp/api/shared/validate"
)

func TestUpdateAddressClearsStateAndPostalCodeWithEmptyOrNull(t *testing.T) {
	t.Parallel()
	body := []byte(`{"state":"","postal_code":null}`)

	var req UpdateAddressRequest
	require.NoError(t, json.Unmarshal(body, &req))
	require.Nil(t, validate.RejectExplicitJSONNulls(body, &req))

	state := apirequest.ClearedAsEmptyString(req.State)
	postalCode := apirequest.ClearedAsEmptyString(req.PostalCode)
	require.NotNil(t, state)
	require.NotNil(t, postalCode)
	require.Equal(t, "", *state)
	require.Equal(t, "", *postalCode)
}

func TestUpdateAddressLeavesAnUnsentStateUnchanged(t *testing.T) {
	t.Parallel()
	var req UpdateAddressRequest
	require.NoError(t, json.Unmarshal([]byte(`{"locality":"George Town"}`), &req))

	require.Nil(t, apirequest.ClearedAsEmptyString(req.State))
	require.Nil(t, apirequest.ClearedAsEmptyString(req.PostalCode))
}
