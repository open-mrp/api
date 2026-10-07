package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	apierror "github.com/open-mrp/api/shared/errors"

	"go.uber.org/mock/gomock"
)

func accountUpdateCtx(accountID string) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_internal",
			AccountID:    &accountID,
			Permissions: map[string]bool{
				types.Permission{Domain: types.PermissionDomainAccount, Action: types.ActionUpdate}.String(): true,
			},
		},
	})
}

func changesByField(changes []audit.FieldChange) map[string]audit.FieldChange {
	out := make(map[string]audit.FieldChange, len(changes))
	for _, c := range changes {
		out[c.Field] = c
	}
	return out
}

// Branding and portal changes are recorded field by field under the names the update request uses,
// never as one struct carrying Go field names and timestamps that move on every save.
func TestAccountChanges_RecordsBrandingAndPortalFieldByField(t *testing.T) {
	t.Parallel()
	before := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	after := before.Add(time.Hour)
	email := "help@example.com"

	old := &domain.Account{
		ID:       "ac_1",
		Name:     "Acme",
		Branding: &domain.AccountBranding{ID: "acbr_1", SupportEmail: &email, PhoneNumber: new("555-0100"), UpdatedAt: before},
		Portal:   &domain.AccountPortal{ID: "acpo_1", Slug: "acme", UpdatedAt: before},
	}
	updated := &domain.Account{
		ID:        "ac_1",
		Name:      "Acme",
		Branding:  &domain.AccountBranding{ID: "acbr_1", SupportEmail: &email, PhoneNumber: new("555-0142"), UpdatedAt: after},
		Portal:    &domain.AccountPortal{ID: "acpo_1", Slug: "acme-inc", UpdatedAt: after},
		UpdatedAt: after,
	}

	got := changesByField(accountChanges(old, updated))
	if len(got) != 2 {
		t.Fatalf("got changes %v, want phone_number and slug only", got)
	}
	for field, want := range map[string][2]string{
		"phone_number": {`"555-0100"`, `"555-0142"`},
		"slug":         {`"acme"`, `"acme-inc"`},
	} {
		c, ok := got[field]
		if !ok {
			t.Errorf("no %s change in %v", field, got)
			continue
		}
		if string(c.OldValue) != want[0] || string(c.NewValue) != want[1] {
			t.Errorf("%s: %s -> %s, want %s -> %s", field, c.OldValue, c.NewValue, want[0], want[1])
		}
	}

	if same := accountChanges(old, old); len(same) != 0 {
		t.Errorf("an unchanged account records changes: %v", same)
	}
}

// An account that had no branding row records only the fields its first update set.
func TestAccountChanges_BrandingCreatedByTheUpdate(t *testing.T) {
	t.Parallel()
	old := &domain.Account{ID: "ac_1", Name: "Acme"}
	updated := &domain.Account{ID: "ac_1", Name: "Acme", Branding: &domain.AccountBranding{ID: "acbr_1", SupportEmail: new("help@example.com")}}

	got := changesByField(accountChanges(old, updated))
	c, ok := got["support_email"]
	if len(got) != 1 || !ok {
		t.Fatalf("got changes %v, want support_email only", got)
	}
	var oldValue any
	if err := json.Unmarshal(c.OldValue, &oldValue); err != nil || oldValue != nil {
		t.Errorf("support_email old value = %s, want null", c.OldValue)
	}
}

// Logos and favicons are served from a public CDN under the type they were stored with, so only real
// images are stored, under the type their bytes show.
func TestAccountSvc_BrandingUploadsStoreOnlyImages(t *testing.T) {
	t.Parallel()
	const accountID = "ac_1"
	ico := []byte("\x00\x00\x01\x00\x01\x00\x10\x10")

	uploads := []struct {
		name   string
		upload func(*accountSvcImpl, []byte) *apierror.APIError
		key    string
		accept map[string][]byte
		refuse map[string][]byte
	}{
		{
			name: "logo",
			upload: func(s *accountSvcImpl, b []byte) *apierror.APIError {
				return s.UploadAccountPhoto(accountUpdateCtx(accountID), accountID, b)
			},
			key:    accountID + "/logo.png",
			accept: map[string][]byte{"image/png": onePixelPNGHeader},
			refuse: map[string][]byte{"html": []byte("<html></html>"), "pdf": []byte("%PDF-1.4\n"), "empty": nil, "ico": ico},
		},
		{
			name: "favicon",
			upload: func(s *accountSvcImpl, b []byte) *apierror.APIError {
				return s.UploadAccountFavicon(accountUpdateCtx(accountID), accountID, b)
			},
			key:    accountID + "/favicon.png",
			accept: map[string][]byte{"image/png": onePixelPNGHeader, "image/x-icon": ico},
			refuse: map[string][]byte{"html": []byte("<html></html>"), "empty": nil},
		},
	}
	for _, tc := range uploads {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			accountRepo := repositorymock.NewMockAccountRepo(ctrl)
			accountRepo.EXPECT().UpdateBrandingLogoURL(gomock.Any(), accountID, tc.key).Return(nil).AnyTimes()
			accountRepo.EXPECT().UpdateBrandingFaviconURL(gomock.Any(), accountID, tc.key).Return(nil).AnyTimes()
			store := &recordingStore{}
			svc := &accountSvcImpl{accountRepo: accountRepo, s3Client: store, accountPhotosBucket: "account-photos"}

			for name, body := range tc.refuse {
				if apiErr := tc.upload(svc, body); apiErr == nil || apierror.GetHTTPStatusCode(apiErr.Code) != 400 {
					t.Errorf("%s: got %v, want a 400", name, apiErr)
				}
			}
			if len(store.uploads) != 0 {
				t.Fatalf("refused uploads must store nothing, got %v", store.uploads)
			}

			for contentType, body := range tc.accept {
				store.uploads = nil
				if apiErr := tc.upload(svc, body); apiErr != nil {
					t.Fatalf("%s: %v", contentType, apiErr)
				}
				if want := tc.key + " " + contentType; len(store.uploads) != 1 || store.uploads[0] != want {
					t.Errorf("uploads = %v, want [%s]", store.uploads, want)
				}
			}
		})
	}
}
