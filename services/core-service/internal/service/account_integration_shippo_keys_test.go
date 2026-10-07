package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/constants"

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
