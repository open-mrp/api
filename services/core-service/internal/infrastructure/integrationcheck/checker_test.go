package integrationcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func TestCheck_ReadsTheProviderAnswer(t *testing.T) {
	t.Parallel()

	var gotPath, gotAuth string
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.RequestURI(), r.Header.Get("Authorization")
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	c := New(&Config{StripeBaseURL: srv.URL, ShippoBaseURL: srv.URL, HubspotBaseURL: srv.URL})

	got, apiErr := c.Check(context.Background(), constants.IntegrationCodeStripe, `{"private_key":"sk_test_1","publishable_key":"pk_test_1","webhook_secret":"whsec_1"}`)
	require.Nil(t, apiErr)
	assert.Equal(t, constants.IntegrationConnectionStatusConnected, got)
	assert.Equal(t, "/v1/balance", gotPath)
	assert.Equal(t, "Bearer sk_test_1", gotAuth)

	status = http.StatusUnauthorized
	got, apiErr = c.Check(context.Background(), constants.IntegrationCodeShippo, `{"api_key":"shippo_test_1"}`)
	require.Nil(t, apiErr)
	assert.Equal(t, constants.IntegrationConnectionStatusInvalidCredentials, got)
	assert.Equal(t, "ShippoToken shippo_test_1", gotAuth)

	status = http.StatusForbidden
	got, apiErr = c.Check(context.Background(), constants.IntegrationCodeHubspot, `{"access_token":"pat-1"}`)
	require.Nil(t, apiErr)
	assert.Equal(t, constants.IntegrationConnectionStatusInvalidCredentials, got)
	assert.Equal(t, "Bearer pat-1", gotAuth)

	status = http.StatusServiceUnavailable
	_, apiErr = c.Check(context.Background(), constants.IntegrationCodeHubspot, `{"access_token":"pat-1"}`)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeExternalServiceError, apiErr.Code)
}

func TestCheck_UnreachableProviderIsAnError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	c := New(&Config{StripeBaseURL: srv.URL})

	_, apiErr := c.Check(context.Background(), constants.IntegrationCodeStripe, `{"private_key":"sk_test_1"}`)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.CodeExternalServiceError, apiErr.Code)
}
