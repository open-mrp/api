package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type deletedScopeHarness struct {
	ctrl      *gomock.Controller
	repos     *factorymock.MockRepoFactory
	mediators *factorymock.MockMediatorFactory
	deleted   *repositorymock.MockDeletedRecordRepo
}

func newDeletedScopeHarness(t *testing.T) *deletedScopeHarness {
	ctrl := gomock.NewController(t)
	h := &deletedScopeHarness{
		ctrl:      ctrl,
		repos:     factorymock.NewMockRepoFactory(ctrl),
		mediators: factorymock.NewMockMediatorFactory(ctrl),
		deleted:   repositorymock.NewMockDeletedRecordRepo(ctrl),
	}
	h.repos.EXPECT().NewDeletedRecordRepo().Return(h.deleted).AnyTimes()
	h.mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()
	return h
}

func deletedScopeCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test",
			AccountID:    &accountID,
			RoleType:     &adminCode,
		},
	})
}

func deletedScopeNotFound() *apierror.APIError {
	return apierror.NewResourceNotFoundError("Not found.")
}

// assertOnlyTheOwnerLearnsItWasDeleted deletes a record that is already gone, as the account it was deleted from and as another account; del makes the record's lookup answer not found for accountID and calls the delete.
func assertOnlyTheOwnerLearnsItWasDeleted(t *testing.T, resourceType constants.DeletedRecordResourceType, id string, del func(h *deletedScopeHarness, accountID string) *apierror.APIError) {
	t.Helper()

	t.Run("the owner is told it is gone", func(t *testing.T) {
		h := newDeletedScopeHarness(t)
		h.deleted.EXPECT().ExistsInAccount(gomock.Any(), resourceType, id, "ac_owner").Return(true, nil)

		apiErr := del(h, "ac_owner")
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeResourceGone, apiErr.Code)
	})

	t.Run("another tenant is told it was never there", func(t *testing.T) {
		h := newDeletedScopeHarness(t)
		h.deleted.EXPECT().ExistsInAccount(gomock.Any(), resourceType, id, "ac_other").Return(false, nil)

		apiErr := del(h, "ac_other")
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	})
}
