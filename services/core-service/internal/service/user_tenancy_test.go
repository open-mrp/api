package service

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"go.uber.org/mock/gomock"
)

const (
	tenancyAccountID = "ac_tenancy"
	tenancyActorID   = "us_actor"
	tenancyOtherID   = "us_elsewhere"
)

var onePixelPNGHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

// recordingStore is an object store that records what was uploaded.
type recordingStore struct {
	stubBrandingStore
	uploads []string
}

func (s *recordingStore) Upload(_ context.Context, _, key string, _ io.Reader, contentType string) *apierror.APIError {
	s.uploads = append(s.uploads, key+" "+contentType)
	return nil
}

func teamAdminCtx() context.Context {
	accountID := tenancyAccountID
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           tenancyActorID,
			AccountID:    &accountID,
			Permissions: map[string]bool{
				types.Permission{Domain: types.PermissionDomainTeamUsers, Action: types.ActionRead}.String():   true,
				types.Permission{Domain: types.PermissionDomainTeamUsers, Action: types.ActionUpdate}.String(): true,
			},
		},
	})
}

// userInAnotherTenancy wires a user who exists but belongs to no account the caller acts in.
func userInAnotherTenancy(ctrl *gomock.Controller) *factorymock.MockRepoFactory {
	email := "someone@elsewhere.example"
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	userRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(nil, apierror.NewResourceNotFoundError("Resource not found.")).AnyTimes()
	userRepo.EXPECT().FindByEmail(gomock.Any(), email).Return(&domain.UserRecord{ID: tenancyOtherID, Email: &email}, nil).AnyTimes()

	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	accountUserRepo.EXPECT().FindByAccountAndUserID(gomock.Any(), tenancyOtherID, tenancyAccountID).
		Return(nil, apierror.NewResourceNotFoundError("Resource not found.")).AnyTimes()

	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewUserRepo().Return(userRepo).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
	return repos
}

// The team permission lets the caller manage users in their own account; a user in any other account is
// not found, whether named by ID or email, and the refusal does not carry the email.
func TestUserSvc_AUserInAnotherTenancyIsNotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	store := &recordingStore{}
	svc := &userSvcImpl{repos: userInAnotherTenancy(ctrl), s3Client: store}

	_, apiErr := svc.GetUser(teamAdminCtx(), "someone@elsewhere.example")
	assertUserNotFound(t, "GetUser by email", apiErr)

	_, apiErr = svc.UpdateUser(teamAdminCtx(), tenancyOtherID, domain.UpdateUserParams{Name: new("Hijacked")})
	assertUserNotFound(t, "UpdateUser", apiErr)

	apiErr = svc.UploadUserPhoto(teamAdminCtx(), tenancyOtherID, onePixelPNGHeader)
	assertUserNotFound(t, "UploadUserPhoto", apiErr)
	if len(store.uploads) != 0 {
		t.Errorf("a refused upload must store nothing, got %v", store.uploads)
	}
}

func assertUserNotFound(t *testing.T, call string, apiErr *apierror.APIError) {
	t.Helper()
	if apiErr == nil || !apierror.IsNotFound(apiErr) {
		t.Errorf("%s: got %s, want resource_not_found", call, apierror.Describe(apiErr))
		return
	}
	if strings.Contains(apiErr.PublicMessage, "@") {
		t.Errorf("%s: the refusal must not carry the email: %q", call, apiErr.PublicMessage)
	}
}

// A user's own record needs no membership lookup: acting in the account already proves it.
func TestUserSvc_TheCallerIsAlwaysFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	userRepo.EXPECT().FindByID(gomock.Any(), tenancyActorID).Return(&domain.UserRecord{ID: tenancyActorID}, nil)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewUserRepo().Return(userRepo).AnyTimes()
	svc := &userSvcImpl{repos: repos}

	user, apiErr := svc.GetUser(teamAdminCtx(), tenancyActorID)
	if apiErr != nil || user.ID != tenancyActorID {
		t.Fatalf("GetUser(self) = %v, %v", user, apiErr)
	}
}

// The photo is stored under the type its bytes show, and anything that is not an image is refused
// before it reaches the bucket.
func TestUserSvc_UploadUserPhotoStoresOnlyImages(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	userRepo := repositorymock.NewMockUserRepo(ctrl)
	userRepo.EXPECT().UpdateImageURL(gomock.Any(), tenancyActorID, gomock.Any()).Return(nil)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewUserRepo().Return(userRepo).AnyTimes()
	store := &recordingStore{}
	svc := &userSvcImpl{repos: repos, s3Client: store, userPhotosBucket: "user-photos"}

	for name, body := range map[string][]byte{"html": []byte("<html><script>alert(1)</script></html>"), "empty": nil} {
		if apiErr := svc.UploadUserPhoto(teamAdminCtx(), tenancyActorID, body); apiErr == nil || apierror.GetHTTPStatusCode(apiErr.Code) != 400 {
			t.Errorf("%s: got %v, want a 400", name, apiErr)
		}
	}
	if len(store.uploads) != 0 {
		t.Fatalf("refused uploads must store nothing, got %v", store.uploads)
	}

	if apiErr := svc.UploadUserPhoto(teamAdminCtx(), tenancyActorID, onePixelPNGHeader); apiErr != nil {
		t.Fatalf("UploadUserPhoto(png): %v", apiErr)
	}
	if want := tenancyAccountID + "/" + tenancyActorID + ".png image/png"; len(store.uploads) != 1 || store.uploads[0] != want {
		t.Errorf("uploads = %v, want [%s]", store.uploads, want)
	}
}
