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
		})
	}
}
