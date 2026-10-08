package mediator

import (
	"context"
	"strings"
	"testing"

	publishermock "github.com/open-mrp/api/services/auth-service/internal/domain/mock/publisher"
	"github.com/open-mrp/api/services/auth-service/internal/testutil"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/messaging"

	"go.uber.org/mock/gomock"
)

// A portal customer's one-click login goes to the merchant's slug on the portal host; an operator's goes to the dashboard.
func TestSendAlreadyRegisteredEmail_LoginURLHost(t *testing.T) {
	t.Parallel()

	slug := "acme"
	cases := []struct {
		name string
		slug *string
		want string
	}{
		{name: "portal customer", slug: &slug, want: "https://portal.example.com/acme" + string(constants.DashboardPathMagicLogin) + "?t="},
		{name: "operator", want: "https://app.example.com" + string(constants.DashboardPathMagicLogin) + "?t="},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			publisher := publishermock.NewMockNotificationPublisher(ctrl)
			var loginURL string
			publisher.EXPECT().
				PublishSendEmail(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, data messaging.EmailSendData) *apierror.APIError {
					loginURL, _ = data.Params["LoginURL"].(string)
					return nil
				}).
				Times(1)

			med := &userMedImpl{
				jwtSecret:             testutil.JWTSecret,
				frontendURL:           "https://app.example.com",
				portalURL:             "https://portal.example.com",
				notificationPublisher: publisher,
			}
			email := "user@example.com"
			med.SendAlreadyRegisteredEmail(context.Background(), &types.User{ID: testutil.EntityIDUser, Email: &email}, tc.slug)

			if !strings.HasPrefix(loginURL, tc.want) {
				t.Fatalf("login URL %q does not start with %q", loginURL, tc.want)
			}
		})
	}
}
