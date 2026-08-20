package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/augno/api/services/auth-service/pkg/types"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/augno/api/shared/errors"
)

// Wires just the account-user repo the shipped_by lookup touches, so each test owns its own
// expectation rather than inheriting the label harness's permissive default.
func newShippedByFixture(t *testing.T, ctrl *gomock.Controller) (*shipmentSvcImpl, *repositorymock.MockAccountUserRepo) {
	t.Helper()

	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()

	return &shipmentSvcImpl{repos: repoFactory}, accountUserRepo
}

// Builds an identity of the given actor type, or one with no actor at all when actorID is empty.
func shippedByIdentity(actorType types.IdentityActorType, actorID, accountID string) *types.Identity {
	identity := &types.Identity{
		Type:   actorType,
		Target: &types.IdentityTarget{AccountID: accountID},
	}
	if actorID != "" {
		identity.Actor = &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           actorID,
			AccountID:    &accountID,
		}
	}
	return identity
}

func TestResolveShippedByID_UserResolvesToAccountUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, accountUserRepo := newShippedByFixture(t, ctrl)

	accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), testLabelAccountID, "usr_internal").
		Return("acus_shipper", nil)

	got, apiErr := svc.resolveShippedByID(context.Background(),
		shippedByIdentity(types.IdentityActorTypeUser, "usr_internal", testLabelAccountID), testLabelAccountID)

	require.Nil(t, apiErr)
	assert.Equal(t, "acus_shipper", got, "a user's ship is credited to their account user")
}

// A user whose account-user link is missing cannot be credited, and legacy rejects the ship outright
// rather than recording an anonymous one.
func TestResolveShippedByID_UnlinkedUserIsRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, accountUserRepo := newShippedByFixture(t, ctrl)

	accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), testLabelAccountID, "usr_internal").
		Return("", apierror.NewResourceNotFoundError("Account user not found."))

	got, apiErr := svc.resolveShippedByID(context.Background(),
		shippedByIdentity(types.IdentityActorTypeUser, "usr_internal", testLabelAccountID), testLabelAccountID)

	require.NotNil(t, apiErr)
	assert.Empty(t, got)
}

// An API key has no account user to credit, so it ships unattributed instead of being turned away —
// machine callers are first class here in a way the legacy dashboard never had to support.
func TestResolveShippedByID_APIKeyShipsUnattributed(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, accountUserRepo := newShippedByFixture(t, ctrl)

	accountUserRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), testLabelAccountID, "apky_test").
		Return("", apierror.NewResourceNotFoundError("Account user not found."))

	got, apiErr := svc.resolveShippedByID(context.Background(),
		shippedByIdentity(types.IdentityActorTypeAPIKey, "apky_test", testLabelAccountID), testLabelAccountID)

	require.Nil(t, apiErr, "an API key must still be able to ship")
	assert.Empty(t, got, "with no account user, shipped_by stays unset")
}

func TestResolveShippedByID_NoActorSkipsTheLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _ := newShippedByFixture(t, ctrl)

	// No ResolveAccountUserID expectation: an actorless identity must not reach the repository.
	got, apiErr := svc.resolveShippedByID(context.Background(),
		shippedByIdentity(types.IdentityActorTypeUnauthenticated, "", testLabelAccountID), testLabelAccountID)

	require.Nil(t, apiErr)
	assert.Empty(t, got)
}
