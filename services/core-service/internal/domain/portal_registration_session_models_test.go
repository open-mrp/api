package domain

import (
	"testing"
	"time"

	"github.com/augno/api/shared/constants"
)

func TestDeriveStatus(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-1 * time.Hour)
	stale := now.Add(-constants.PortalRegistrationSessionTTL - time.Hour)
	completedAt := now.Add(-30 * time.Minute)
	abandonedAt := now.Add(-30 * time.Minute)

	cases := []struct {
		name    string
		session PortalRegistrationSession
		want    constants.PortalRegistrationStatus
	}{
		{
			name:    "completed takes precedence even when created long ago",
			session: PortalRegistrationSession{CreatedAt: stale, CompletedAt: &completedAt},
			want:    constants.PortalRegistrationStatusCompleted,
		},
		{
			name:    "abandoned takes precedence over expiry",
			session: PortalRegistrationSession{CreatedAt: stale, AbandonedAt: &abandonedAt},
			want:    constants.PortalRegistrationStatusAbandoned,
		},
		{
			name:    "incomplete and recent is in_progress",
			session: PortalRegistrationSession{CreatedAt: recent},
			want:    constants.PortalRegistrationStatusInProgress,
		},
		{
			name:    "incomplete and past the TTL is expired",
			session: PortalRegistrationSession{CreatedAt: stale},
			want:    constants.PortalRegistrationStatusExpired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.session.DeriveStatus(now); got != tc.want {
				t.Errorf("DeriveStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}
