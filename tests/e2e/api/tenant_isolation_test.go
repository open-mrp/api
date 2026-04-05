//go:build e2e

package api_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tenant isolation tests verify that resources created by one account are not
// accessible by another account. This is the highest-priority security concern
// for a multi-tenant SaaS platform.
//
// These tests use Tenant B (seeded in 0015_tenant_b_e2e.sql) — a completely
// independent account with its own API key. They verify cross-tenant access
// is denied with 404 (not 403, to avoid leaking resource existence).

// tenantBClient is a Client authenticated as tenant B.
var tenantBClient *Client

func getTenantBClient() *Client {
	if tenantBClient == nil {
		baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
		tenantBClient = NewClient(baseURL, SeedTenantBAPIKey, SeedTenantBAccountID)
	}
	return tenantBClient
}

// TestTenantIsolation_GetByID verifies that tenant B cannot GET a resource
// belonging to tenant A. The API should return 404, not 403.
func TestTenantIsolation_GetByID(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	// Create a resource in tenant A.
	created := createAndCleanup(t, customersPath, map[string]any{
		"name": uniqueName("e2e-iso-get"),
	})
	id := jsonField(created, "id")

	// Attempt to GET it from tenant B.
	statusCode, body, err := clientB.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, statusCode,
		"Tenant B should get 404 (not 403) for tenant A's resource: %s", string(body))
}

// TestTenantIsolation_UpdateByID verifies that tenant B cannot PATCH a resource
// belonging to tenant A.
func TestTenantIsolation_UpdateByID(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAndCleanup(t, customersPath, map[string]any{
		"name": uniqueName("e2e-iso-patch"),
	})
	id := jsonField(created, "id")

	statusCode, body, err := clientB.Patch(customersPath+"/"+id, map[string]any{
		"note": "cross-tenant update attempt",
	}, newIdempotencyKey())
	require.NoError(t, err)
	assert.Equal(t, 404, statusCode,
		"Tenant B should get 404 for PATCH on tenant A's resource: %s", string(body))
}

// TestTenantIsolation_DeleteByID verifies that tenant B cannot DELETE a resource
// belonging to tenant A.
func TestTenantIsolation_DeleteByID(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAndCleanup(t, customersPath, map[string]any{
		"name": uniqueName("e2e-iso-delete"),
	})
	id := jsonField(created, "id")

	statusCode, body, err := clientB.Delete(customersPath + "/" + id)
	require.NoError(t, err)
	assert.Equal(t, 404, statusCode,
		"Tenant B should get 404 for DELETE on tenant A's resource: %s", string(body))

	// Verify resource still exists in tenant A.
	getStatus, _, err := apiClient.GetListRaw(customersPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, getStatus, "Resource should still exist in tenant A after tenant B's delete attempt")
}

// TestTenantIsolation_ListDoesNotLeak verifies that tenant B's list results
// do not contain resources belonging to tenant A.
func TestTenantIsolation_ListDoesNotLeak(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	// Create a resource with a distinctive name in tenant A.
	distinctName := uniqueName("e2e-iso-leak")
	createAndCleanup(t, customersPath, map[string]any{
		"name": distinctName,
	})

	// List from tenant B with search for the distinctive name.
	list, _, err := clientB.GetList(customersPath, url.Values{"q": {distinctName}})
	require.NoError(t, err)
	assertEmptyListData(t, list.Data,
		"Tenant B's list should not contain tenant A's resource with name "+distinctName)
}

// TestTenantIsolation_AccountGroupIsolation verifies account groups are isolated.
func TestTenantIsolation_AccountGroupIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAndCleanup(t, "/v1/sales/account-groups", map[string]any{
		"name": uniqueName("e2e-iso-grp"),
		"type": "type_group",
	})
	id := jsonField(created, "id")

	statusCode, _, err := clientB.GetListRaw("/v1/sales/account-groups/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, statusCode, "Tenant B should not see tenant A's account group")
}

// TestTenantIsolation_APIKeyIsolation verifies API keys are isolated between tenants.
func TestTenantIsolation_APIKeyIsolation(t *testing.T) {
	t.Parallel()
	clientB := getTenantBClient()

	created := createAPIKeyAndCleanup(t, uniqueName("e2e-iso-key"))
	id := jsonField(jsonObject(created, "api_key_info"), "id")

	statusCode, _, err := clientB.GetListRaw(apiKeysPath+"/"+id, nil)
	require.NoError(t, err)
	assert.Equal(t, 404, statusCode, "Tenant B should not see tenant A's API key")
}
