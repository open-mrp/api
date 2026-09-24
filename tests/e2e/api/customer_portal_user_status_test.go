//go:build e2e

package api_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Disabling a customer's portal user must cut that user off from the merchant. The user's token targets the merchant without an OpenMRP-Actor-Account header, which resolves access through the customer relation rather than through the customer account's membership check.
func TestCustomerPortalUser_DisabledLosesMerchantAccess(t *testing.T) {
	t.Parallel()
	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	// The merchant provisions its customer's portal users; the customer administers them from inside its own account.
	merchant := NewClient(baseURL, SeedAPIKey, SeedCustomerAccountID)
	customerAdmin := NewClient(baseURL, SeedCustomerAPIKey, SeedCustomerAccountID)
	username := uniqueName("e2e-portal-user")
	password := "PortalPass123!"

	status, body, err := merchant.Post(accountUsersPath, map[string]any{
		"username": username,
		"password": password,
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	accountUserID := jsonField(parseJSON(body), "id")
	defer func() { _, _, _ = merchant.Put(accountUsersPath+"/"+accountUserID+"/actions/remove", nil) }()

	portalUser := loginAsUser(t, username, password, SeedAccountID)
	merchantOrders := func() int {
		status, _, err := portalUser.GetListRaw(salesOrdersPath, nil)
		require.NoError(t, err)
		return status
	}
	require.Equal(t, http.StatusOK, merchantOrders(), "an active portal user should reach the merchant")

	status, body, err = customerAdmin.Put(accountUsersPath+"/"+accountUserID+"/actions/disable", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	// Revocation reaches the credential cache through the audit-event stream, so it is prompt but not synchronous.
	require.Eventually(t, func() bool { return merchantOrders() == http.StatusForbidden }, 5*time.Second, 100*time.Millisecond,
		"a disabled portal user must lose access to the merchant")
}
