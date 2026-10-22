package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, status int, body string) *client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/key", r.URL.Path)
		assert.Equal(t, "Bearer sk-or-v1-test", r.Header.Get("Authorization"))
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	c, err := NewClient(&ClientConfig{BaseURL: server.URL})
	require.NoError(t, err)
	return c.(*client)
}

func TestGetKeyUsage(t *testing.T) {
	c := serve(t, http.StatusOK, `{"data":{"usage":212.6,"usage_daily":1.25,"usage_weekly":8.4,"usage_monthly":31.90000001,"limit":null,"limit_remaining":null}}`)

	usage, apiErr := c.GetKeyUsage(context.Background(), "sk-or-v1-test")

	require.Nil(t, apiErr)
	assert.Equal(t, "1.25", usage.SpentToday)
	assert.Equal(t, "8.4", usage.SpentThisWeek)
	assert.Equal(t, "31.9", usage.SpentThisMonth)
	assert.Equal(t, "212.6", usage.SpentTotal)
	assert.Nil(t, usage.KeyLimit)
	assert.Nil(t, usage.KeyLimitRemaining)
}

func TestGetKeyUsage_KeyLimit(t *testing.T) {
	c := serve(t, http.StatusOK, `{"data":{"usage":10,"usage_daily":0,"usage_weekly":0,"usage_monthly":10,"limit":25,"limit_remaining":15}}`)

	usage, apiErr := c.GetKeyUsage(context.Background(), "sk-or-v1-test")

	require.Nil(t, apiErr)
	require.NotNil(t, usage.KeyLimit)
	assert.Equal(t, "25", *usage.KeyLimit)
	assert.Equal(t, "15", *usage.KeyLimitRemaining)
}

func TestGetKeyUsage_RejectedKeyIsTheCallersToFix(t *testing.T) {
	c := serve(t, http.StatusUnauthorized, `{"error":{"message":"No auth credentials found"}}`)

	_, apiErr := c.GetKeyUsage(context.Background(), "sk-or-v1-test")

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
}

func TestGetKeyUsage_ServerErrorIsTransient(t *testing.T) {
	c := serve(t, http.StatusBadGateway, `upstream down`)

	_, apiErr := c.GetKeyUsage(context.Background(), "sk-or-v1-test")

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeExternalSvcError, apiErr.Code)
	assert.True(t, apiErr.IsTransient)
}
