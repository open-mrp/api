package appctx

import (
	"context"
	"testing"

	"github.com/open-mrp/api/shared/constants"
)

// api-gateway/internal/cookie uses `ok` as the "platform is configured" gate and panics when it is
// false, so this pins what the gate actually proves: the key was set, not that the value is a real
// PlatformMode. A typo'd PLATFORM env var passes the gate and is treated as non-production.
func TestGetPlatformFromContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		ctx          context.Context
		expected     constants.PlatformMode
		expectOK     bool
		expectIsEnum bool
	}{
		{
			name:     "absent",
			ctx:      context.Background(),
			expectOK: false,
		},
		{
			name:     "stored as a plain string",
			ctx:      context.WithValue(context.Background(), platformKey, "production"),
			expectOK: false,
		},
		{
			name:     "empty mode",
			ctx:      WithPlatform(context.Background(), constants.PlatformMode("")),
			expected: constants.PlatformMode(""),
			expectOK: true,
		},
		{
			name:     "truncated mode",
			ctx:      WithPlatform(context.Background(), constants.PlatformMode("prod")),
			expected: constants.PlatformMode("prod"),
			expectOK: true,
		},
		{
			name:     "wrong case mode",
			ctx:      WithPlatform(context.Background(), constants.PlatformMode("PRODUCTION")),
			expected: constants.PlatformMode("PRODUCTION"),
			expectOK: true,
		},
		{
			name:         "valid mode",
			ctx:          WithPlatform(context.Background(), constants.PlatformModeProduction),
			expected:     constants.PlatformModeProduction,
			expectOK:     true,
			expectIsEnum: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			platform, ok := GetPlatformFromContext(tt.ctx)
			if ok != tt.expectOK {
				t.Errorf("expected ok=%v, got %v", tt.expectOK, ok)
			}
			if platform != tt.expected {
				t.Errorf("expected platform %q, got %q", tt.expected, platform)
			}
			if platform.IsValid() != tt.expectIsEnum {
				t.Errorf("expected IsValid()=%v for %q, got %v", tt.expectIsEnum, platform, platform.IsValid())
			}
			if tt.expectOK && !tt.expectIsEnum && platform.IsProduction() {
				t.Errorf("expected %q to not be treated as production", platform)
			}
		})
	}
}
