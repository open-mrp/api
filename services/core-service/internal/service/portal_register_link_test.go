package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// The order-online link targets the merchant's verified custom domain when it has one, else its slug on the shared
// portal host, and is left out when the merchant has no portal.
func TestPortalRegisterLink(t *testing.T) {
	t.Parallel()

	const accountID = "ac_seller"
	slug := "acme"
	verified := &domain.PortalDomain{Domain: "shop.example.net", Status: constants.PortalDomainStatusVerified}
	pending := &domain.PortalDomain{Domain: "shop.example.net", Status: constants.PortalDomainStatusPending}
	cases := []struct {
		name         string
		portalDomain *domain.PortalDomain
		slug         *string
		want         string
	}{
		{name: "a verified custom domain serves the portal without the slug", portalDomain: verified, want: "https://shop.example.net/auth/register"},
		{name: "otherwise the slug on the portal host", slug: &slug, want: "https://portal.example.com/acme/auth/register"},
		{name: "an unverified domain falls back to the slug", portalDomain: pending, slug: &slug, want: "https://portal.example.com/acme/auth/register"},
		{name: "no portal gives no link", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repos := factorymock.NewMockRepoFactory(ctrl)
			portalDomainRepo := repositorymock.NewMockPortalDomainRepo(ctrl)
			accountRepo := repositorymock.NewMockAccountRepo(ctrl)
			repos.EXPECT().NewPortalDomainRepo().Return(portalDomainRepo).AnyTimes()
			repos.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
			portalDomainRepo.EXPECT().GetByAccountID(gomock.Any(), accountID).Return(tc.portalDomain, nil)
			accountRepo.EXPECT().GetPortalSlug(gomock.Any(), accountID).Return(tc.slug, nil).MaxTimes(1)

			assert.Equal(t, tc.want, portalRegisterLink(context.Background(), repos, "https://portal.example.com/", accountID))
		})
	}
}
