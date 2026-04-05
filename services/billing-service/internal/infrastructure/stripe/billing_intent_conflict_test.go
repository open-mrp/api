package stripe

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBillingIntentConflict(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		err    error
		wantID string
		wantOK bool
	}{
		{
			name:   "nil error",
			err:    nil,
			wantOK: false,
		},
		{
			name:   "unrelated error",
			err:    fmt.Errorf("connection refused"),
			wantOK: false,
		},
		{
			name:   "real stripe conflict error",
			err:    fmt.Errorf(`failed to create billing intent: {"code":"invalid_fields","status":400,"message":"Some fields in the request were invalid: 'pricing_plan_subscription_ids: One or more pricing plan subscriptions in this request are already reserved to be modified or deactivated through a billing intent. Pricing plan subscription(s) bpps_test_61UOAdttw0IzX2fO316UHtw6IgSQueoEsXV6K51aiWQC are reserved by billing intent bilint_test_61UOBcAHTPNYUy96C16UHtw6IgSQueoEsXV6K51ai5A0. To resolve this, cancel the conflicting billing intent(s), then try again.'"}`),
			wantID: "bilint_test_61UOBcAHTPNYUy96C16UHtw6IgSQueoEsXV6K51ai5A0",
			wantOK: true,
		},
		{
			name:   "live mode intent ID",
			err:    fmt.Errorf(`reserved by billing intent bilint_live_abc123XYZ. To resolve this`),
			wantID: "bilint_live_abc123XYZ",
			wantOK: true,
		},
		{
			name:   "reserved marker without bilint prefix",
			err:    fmt.Errorf(`reserved by billing intent something_else`),
			wantOK: false,
		},
		{
			name:   "intent ID at end of string",
			err:    fmt.Errorf(`reserved by billing intent bilint_test_abc123`),
			wantID: "bilint_test_abc123",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := parseBillingIntentConflict(tt.err)
			assert.Equal(t, tt.wantOK, gotOK)
			if tt.wantOK {
				assert.Equal(t, tt.wantID, gotID)
			}
		})
	}
}
