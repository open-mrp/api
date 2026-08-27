package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/ptrutil"

	"go.uber.org/mock/gomock"
)

func accountReadCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			Permissions: map[string]bool{
				types.Permission{Domain: types.PermissionDomainAccount, Action: types.ActionRead}.String(): true,
			},
		},
	})
}

// account_branding.logo_url stores the object key the upload wrote, so the resource's logo_url/favicon_url fields advertise a URL while carrying something no client can load. Every read path that hands out branding has to sign it — GetAccount and BatchGetAccountsByIDs both feed the same account resource, and only one of them signing is the shape of bug that ships.
func TestAccountReads_SignBrandingAssetKeys(t *testing.T) {
	const accountID = "ac_1"
	branding := func() *domain.AccountBranding {
		logoKey, faviconKey := accountID+"/logo.png", accountID+"/favicon.png"
		return &domain.AccountBranding{ID: "abr_1", LogoURL: &logoKey, FaviconURL: &faviconKey}
	}

	cases := []struct {
		name string
		read func(*testing.T, *accountSvcImpl) *domain.Account
	}{
		{
			name: "GetAccount",
			read: func(t *testing.T, svc *accountSvcImpl) *domain.Account {
				account, apiErr := svc.GetAccount(accountReadCtx(accountID), accountID)
				if apiErr != nil {
					t.Fatalf("GetAccount: %v", apiErr)
				}
				return account
			},
		},
		{
			name: "BatchGetAccountsByIDs",
			read: func(t *testing.T, svc *accountSvcImpl) *domain.Account {
				accounts, apiErr := svc.BatchGetAccountsByIDs(accountReadCtx(accountID), []string{accountID})
				if apiErr != nil {
					t.Fatalf("BatchGetAccountsByIDs: %v", apiErr)
				}
				if len(accounts) != 1 {
					t.Fatalf("got %d accounts, want 1", len(accounts))
				}
				return accounts[0]
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			accountRepo := repositorymock.NewMockAccountRepo(ctrl)
			accountRepo.EXPECT().GetByID(gomock.Any(), accountID).Return(&domain.Account{ID: accountID, Branding: branding()}, nil).AnyTimes()
			accountRepo.EXPECT().GetByIDs(gomock.Any(), []string{accountID}).Return([]*domain.Account{{ID: accountID, Branding: branding()}}, nil).AnyTimes()

			store := &stubBrandingStore{presignedURL: "https://s3.example.com/signed"}
			svc := &accountSvcImpl{accountRepo: accountRepo, s3Client: store, accountPhotosBucket: "account-photos"}

			account := tc.read(t, svc)
			if got := ptrutil.Deref(account.Branding.LogoURL); got != "https://s3.example.com/signed" {
				t.Errorf("logo_url = %q, want a signed URL", got)
			}
			if got := ptrutil.Deref(account.Branding.FaviconURL); got != "https://s3.example.com/signed" {
				t.Errorf("favicon_url = %q, want a signed URL", got)
			}
			// No CDN configured (dev/e2e): fall back to presigning. A branding URL rides in long-lived HTML (the portal favicon <link>), so a short signature 403s once the browser reuses the cached URL past its expiry. Sign for the SigV4 maximum instead.
			if store.presignExpiry != brandingPresignExpiry {
				t.Errorf("presign expiry = %s, want %s", store.presignExpiry, brandingPresignExpiry)
			}
		})
	}
}

// With a CDN configured, branding keys must be handed out as stable "<base>/<key>" URLs and never presigned — a presigned URL 403s once its signature (or the pod's STS session) expires while the portal favicon <link> still references it.
func TestAccountReads_ServeBrandingFromCDNWhenConfigured(t *testing.T) {
	const accountID = "ac_1"
	ctrl := gomock.NewController(t)
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	branding := &domain.AccountBranding{ID: "abr_1", LogoURL: ptr(accountID + "/logo.png"), FaviconURL: ptr(accountID + "/favicon.png")}
	accountRepo.EXPECT().GetByID(gomock.Any(), accountID).Return(&domain.Account{ID: accountID, Branding: branding}, nil).AnyTimes()

	store := &stubBrandingStore{presignedURL: "https://s3.example.com/signed"}
	svc := &accountSvcImpl{accountRepo: accountRepo, s3Client: store, accountPhotosBucket: "account-photos", assetCDNBaseURL: "https://cdn.augno.com"}

	account, apiErr := svc.GetAccount(accountReadCtx(accountID), accountID)
	if apiErr != nil {
		t.Fatalf("GetAccount: %v", apiErr)
	}
	if got := ptrutil.Deref(account.Branding.LogoURL); got != "https://cdn.augno.com/ac_1/logo.png" {
		t.Errorf("logo_url = %q, want the CDN URL", got)
	}
	if got := ptrutil.Deref(account.Branding.FaviconURL); got != "https://cdn.augno.com/ac_1/favicon.png" {
		t.Errorf("favicon_url = %q, want the CDN URL", got)
	}
	if store.presignKey != "" {
		t.Errorf("presigned %q, want no presign call when a CDN is configured", store.presignKey)
	}
}
