//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deletions and lookups the suite had never called.
//
// Grouped because they share a shape rather than a domain: each is the last operation on a
// resource, and each has the same two failure modes — succeeding on something that does not
// exist, and succeeding on something that is still in use.

const (
	batchesPath          = "/v1/operations/batches"
	supplierMaterialsFmt = suppliersPath + "/%s/materials/%s"
)

// ──────────────────────────────────────────────
// Shipping cases
// ──────────────────────────────────────────────

// packedShipmentCase returns a shipping case from a freshly packed shipment. Packing a pick with one case is what creates them.
func packedShipmentCase(t *testing.T) (shipmentID string, caseID string) {
	t.Helper()

	shipment := packedShipment(t)
	shipmentID = jsonField(shipment, "id")

	status, body, err := apiClient.GetListRaw(shipmentsPath+"/"+shipmentID, url.Values{"include": {"shipping_cases"}})
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	cases := jsonListData(parseJSON(body), "shipping_cases")
	require.NotEmpty(t, cases, "packing with one case must produce a shipping case: %s", string(body))
	first, ok := cases[0].(map[string]any)
	require.True(t, ok)
	caseID = jsonField(first, "id")
	require.NotEmpty(t, caseID)
	return shipmentID, caseID
}

func TestShippingCases_DeleteRemovesTheCase(t *testing.T) {
	t.Parallel()

	_, caseID := packedShipmentCase(t)

	status, body, err := apiClient.Delete(shippingCasesPath + "/" + caseID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	assert.Contains(t, []int{200, 204}, status, "delete should succeed: %s", string(body))

	getStatus, getBody, err := apiClient.GetListRaw(shippingCasesPath+"/"+caseID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus, "a deleted shipping case must be gone: %s", string(getBody))
}

func TestShippingCases_DeleteUnknownCaseIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(shippingCasesPath + "/shcs_doesnotexist0")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown shipping case must 404: %s", string(body))
}

// A case that has never been labelled answers with a null URL rather than an error: the resource documents the link as absent until a carrier has produced one, so the case existing and the label existing are deliberately separate questions.
func TestShippingCases_LabelIsNullUntilOneIsGenerated(t *testing.T) {
	t.Parallel()

	_, caseID := packedShipmentCase(t)

	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/"+caseID+"/label", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "label must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)

	parsed := parseJSON(body)
	assert.Equal(t, "shipping_case_label_url", jsonField(parsed, "object"))
	assert.Empty(t, jsonField(parsed, "url"), "an unlabelled case has no link yet: %s", string(body))
}

func TestShippingCases_LabelForUnknownCaseIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(shippingCasesPath+"/shcs_doesnotexist0/label", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a label for an unknown case must 404: %s", string(body))
}

// ──────────────────────────────────────────────
// Suppliers
// ──────────────────────────────────────────────

func createSupplier(t *testing.T) string {
	t.Helper()

	created := createAndCleanup(t, suppliersPath, map[string]any{
		"name":   uniqueName("e2e-supplier"),
		"number": uniqueName("SUP"),
	})
	id := jsonField(created, "id")
	require.NotEmpty(t, id)
	return id
}

func TestSuppliers_DeleteRemovesTheSupplier(t *testing.T) {
	t.Parallel()

	supplierID := createSupplier(t)

	status, body, err := apiClient.Delete(suppliersPath + "/" + supplierID)
	require.NoError(t, err)
	require.Less(t, status, 500, "delete must not 5xx: %s", string(body))
	assert.Contains(t, []int{200, 204}, status, "delete should succeed: %s", string(body))

	getStatus, getBody, err := apiClient.GetListRaw(suppliersPath+"/"+supplierID, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, getStatus, "a deleted supplier must be gone: %s", string(getBody))
}

func TestSuppliers_DeleteUnknownSupplierIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(suppliersPath + "/ac_doesnotexist00000")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown supplier must 404: %s", string(body))
}

func TestSupplierMaterials_DeleteUnknownMaterialIs404(t *testing.T) {
	t.Parallel()

	supplierID := createSupplier(t)

	status, body, err := apiClient.Delete(suppliersPath + "/" + supplierID + "/materials/spml_doesnotexist")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown supplier material must 404: %s", string(body))
}

// A supplier material belongs to one supplier; reaching it through another must not resolve.
func TestSupplierMaterials_MaterialFromAnotherSupplierIsNotAddressable(t *testing.T) {
	t.Parallel()

	otherSupplierID := createSupplier(t)

	status, body, err := apiClient.Delete(suppliersPath + "/" + otherSupplierID + "/materials/" + SeedSupplierMaterialID)
	require.NoError(t, err)
	require.Less(t, status, 500, "must not 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a material from another supplier must not be deletable: %s", string(body))
}

// ──────────────────────────────────────────────
// Batches
// ──────────────────────────────────────────────

// The flow graph is the batch's genealogy, so it must at minimum contain the batch it was asked about.
func TestBatches_FlowContainsTheBatch(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(batchesPath+"/"+SeedBatchID+"/flow", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "flow must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	assert.Contains(t, string(body), SeedBatchID, "the flow must contain the batch it was asked about")
}

func TestBatches_FlowForUnknownBatchIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.GetListRaw(batchesPath+"/bt_doesnotexist00000/flow", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "a flow for an unknown batch must 404: %s", string(body))
}

func TestBatches_DeleteUnknownBatchIs404(t *testing.T) {
	t.Parallel()

	status, body, err := apiClient.Delete(batchesPath + "/bt_doesnotexist00000")
	require.NoError(t, err)
	require.Less(t, status, 500, "must 404 rather than 5xx: %s", string(body))
	assert.Equal(t, 404, status, "deleting an unknown batch must 404: %s", string(body))
}
