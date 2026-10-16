package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

// Only a production deployment may hold a live Shippo key, and never for a sandbox account.
func TestValidateShippoCredentials_LiveKeysOnlyInProduction(t *testing.T) {
	const live, test = `{"api_key":"shippo_live_x"}`, `{"api_key":"shippo_test_x"}`
	tests := []struct {
		name      string
		mode      constants.PlatformMode
		isSandbox bool
		creds     string
		wantOK    bool
	}{
		{"production account in production takes a live key", constants.PlatformModeProduction, false, live, true},
		{"production account in production refuses a test key", constants.PlatformModeProduction, false, test, false},
		{"sandbox in production takes a test key", constants.PlatformModeProduction, true, test, true},
		{"sandbox in production refuses a live key", constants.PlatformModeProduction, true, live, false},
		{"development refuses a live key", constants.PlatformModeDevelopment, false, live, false},
		{"development takes a test key", constants.PlatformModeDevelopment, false, test, true},
		{"test mode refuses a live key", constants.PlatformModeTest, false, live, false},
		{"test mode takes a test key", constants.PlatformModeTest, false, test, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &accountIntegrationSvcImpl{platformMode: tc.mode}
			apiErr := svc.validateShippoCredentials(trace.SpanFromContext(context.Background()), tc.creds, tc.isSandbox)
			if tc.wantOK {
				assert.Nil(t, apiErr)
			} else {
				assert.NotNil(t, apiErr)
			}
		})
	}
}

type fixedChecker struct {
	status constants.IntegrationConnectionStatus
	err    *apierror.APIError
}

func (c fixedChecker) Check(context.Context, constants.IntegrationCode, string) (constants.IntegrationConnectionStatus, *apierror.APIError) {
	return c.status, c.err
}

// A provider that rejects the credentials is the caller's to fix (422 on credentials); one that gives no answer
// fails the request without saving.
func TestCheckWithProvider(t *testing.T) {
	ctx := context.Background()

	svc := &accountIntegrationSvcImpl{checker: fixedChecker{status: constants.IntegrationConnectionStatusConnected}}
	status, apiErr := svc.checkWithProvider(ctx, constants.IntegrationCodeShippo, `{}`)
	assert.Nil(t, apiErr)
	assert.Equal(t, constants.IntegrationConnectionStatusConnected, status)

	svc.checker = fixedChecker{status: constants.IntegrationConnectionStatusInvalidCredentials}
	_, apiErr = svc.checkWithProvider(ctx, constants.IntegrationCodeStripe, `{}`)
	if assert.NotNil(t, apiErr) {
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
		assert.Equal(t, "credentials", apiErr.Param)
		assert.Equal(t, "Stripe rejected these credentials.", apiErr.Errors[0].Message)
	}

	svc.checker = fixedChecker{err: apierror.NewExternalServiceError(nil, "down")}
	_, apiErr = svc.checkWithProvider(ctx, constants.IntegrationCodeHubspot, `{}`)
	if assert.NotNil(t, apiErr) {
		assert.Equal(t, apierror.CodeExternalServiceError, apiErr.Code)
	}
}
