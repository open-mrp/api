package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// internalAdminCtx builds an identity ctx whose actor is an internal admin
// targeted at `targetAccountID`. Admins bypass CheckHasPermission, simplifying
// the harness for permission-orthogonal tests.
func internalAdminCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test",
			AccountID:    &targetAccountID,
			RoleType:     &adminCode,
		},
	})
}

// SAFETY: This test guards against a cross-account data leak. Callers can
// only ever read their own account; BatchGetAccountsByIDs must filter input
// IDs to just the caller's target account before the repo is consulted.
// Regression here would let any caller fetch arbitrary accounts by ID via
// the api-gateway owner.account include.
func TestBatchGetAccountsByIDs_FiltersToCallerAccount(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const targetAcct = "acct_target"
	mockRepo := repositorymock.NewMockAccountRepo(ctrl)
	// Repo must be invoked with ONLY the target account ID, never the
	// arbitrary ids the caller smuggled in.
	mockRepo.EXPECT().
		GetByIDs(gomock.Any(), []string{targetAcct}).
		Return(nil, nil)

	// The smuggled ids are checked for an account_relation; with none, they are
	// dropped and never reach GetByIDs.
	mockRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	mockRelationRepo.EXPECT().HasRelation(gomock.Any(), targetAcct, "acct_other_1").Return(false, nil)
	mockRelationRepo.EXPECT().HasRelation(gomock.Any(), targetAcct, "acct_other_2").Return(false, nil)

	svc := &accountSvcImpl{accountRepo: mockRepo, accountRelationRepo: mockRelationRepo}

	ctx := internalAdminCtx(targetAcct)
	_, apiErr := svc.BatchGetAccountsByIDs(ctx, []string{
		"acct_other_1",
		targetAcct,
		"acct_other_2",
	})
	require.Nil(t, apiErr)
}

// When the input contains no IDs that match the caller's target account,
// the repo must NOT be called at all (preventing a probe-style leak via
// repository-side errors).
func TestBatchGetAccountsByIDs_NoMatchingIDs_ShortCircuits(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const targetAcct = "acct_target"
	mockRepo := repositorymock.NewMockAccountRepo(ctrl)
	// Zero EXPECT calls — repo must not be touched.

	// Both smuggled ids have no relation to the caller, so after filtering the
	// allowed set is empty and the account repo is never consulted.
	mockRelationRepo := repositorymock.NewMockAccountRelationRepo(ctrl)
	mockRelationRepo.EXPECT().HasRelation(gomock.Any(), targetAcct, "acct_other_1").Return(false, nil)
	mockRelationRepo.EXPECT().HasRelation(gomock.Any(), targetAcct, "acct_other_2").Return(false, nil)

	svc := &accountSvcImpl{accountRepo: mockRepo, accountRelationRepo: mockRelationRepo}
	ctx := internalAdminCtx(targetAcct)
	result, apiErr := svc.BatchGetAccountsByIDs(ctx, []string{"acct_other_1", "acct_other_2"})
	require.Nil(t, apiErr)
	require.Nil(t, result)
}

// Empty input ID slice is a no-op; no repo call, no error.
func TestBatchGetAccountsByIDs_EmptyInput(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repositorymock.NewMockAccountRepo(ctrl)
	svc := &accountSvcImpl{accountRepo: mockRepo}
	ctx := internalAdminCtx("acct_target")
	result, apiErr := svc.BatchGetAccountsByIDs(ctx, nil)
	require.Nil(t, apiErr)
	require.Nil(t, result)
}

// Missing identity in ctx must fail invariant rather than silently passing.
func TestBatchGetAccountsByIDs_NoIdentity_Fails(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repositorymock.NewMockAccountRepo(ctrl)
	svc := &accountSvcImpl{accountRepo: mockRepo}
	_, apiErr := svc.BatchGetAccountsByIDs(context.Background(), []string{"acct_x"})
	require.NotNil(t, apiErr)
}
