//go:build e2e

package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const preview6APIVersion = "1.0.forge-preview.6"

// preview.7 drops a customer's `edi_status` and an invoice's `is_edi_sent`. A client still pinned to
// preview.6 that sends them is not refused for an unknown field: the values are dropped, since OpenMRP no
// longer stores them, and neither version returns them.
func TestVersionCompat_Preview6EdiFieldsAreDroppedNotRefused(t *testing.T) {
	t.Parallel()
	pinned := apiClient.WithAPIVersion(preview6APIVersion)

	body := validCustomerBody(uniqueName("e2e-compat-edi"))
	body["edi_status"] = "enabled"
	status, respBody, err := pinned.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	created := parseJSON(respBody)
	id := jsonField(created, "id")
	defer apiClient.Delete(customersPath + "/" + id)
	assert.NotContains(t, created, "edi_status")

	status, respBody, err = pinned.Patch(customersPath+"/"+id, map[string]any{"edi_status": "disabled", "note": "compat"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	assert.Equal(t, "compat", jsonField(parseJSON(respBody), "note"), "the rest of the update applies")

	status, respBody, err = apiClient.Post(customersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 400, status, "at preview.7 the field is unknown: %s", string(respBody))

	inv := invoiceNewCustomer(t)
	status, respBody, err = pinned.Patch(financeInvoicesPath+"/"+inv.invoiceID, map[string]any{"is_edi_sent": true, "note": "compat"}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 200, status, respBody)
	got := parseJSON(respBody)
	assert.Equal(t, "compat", jsonField(got, "note"))
	assert.NotContains(t, got, "is_edi_sent")
	assert.NotContains(t, got, "customer_is_edi_enabled")
}
