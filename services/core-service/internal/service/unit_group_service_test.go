package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func unitGroupInternalIdentityCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"unit_groups:update": true,
			},
		},
	})
}

func TestUpsertUnitGroupUnit_IdempotencyStarted_CachesSuccess(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	unitGroupRepo := repositorymock.NewMockUnitGroupRepo(ctrl)
	unitQueryRepo := repositorymock.NewMockUnitQueryRepo(ctrl)
	repoFactory := factorymock.NewMockRepoFactory(ctrl)
	repoFactory.EXPECT().NewUnitGroupRepo().Return(unitGroupRepo).AnyTimes()
	repoFactory.EXPECT().NewUnitQueryRepo().Return(unitQueryRepo).AnyTimes()

	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: idempotencyMed,
	}).AnyTimes()

	idempotencyMed.EXPECT().
		UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{
			TypeID:        "idk_test",
			RecoveryPoint: string(domain.RecoveryPointStarted),
		}, nil).
		Times(1)

	accountID := "ac_test123"
	aid := accountID
	existing := &domain.UnitGroupFull{
		ID:        "ug_test",
		Type:      "weight",
		AccountID: &aid,
	}
	unitGroupRepo.EXPECT().
		Get(gomock.Any(), domain.GetUnitGroupParams{AccountID: accountID, UnitGroupID: "ug_test"}).
		Return(existing, nil)

	unitQueryRepo.EXPECT().
		Find(gomock.Any(), accountID, "un_child").
		Return(&domain.LightUnit{Type: "weight"}, nil)

	out := &domain.UnitGroupUnit{
		ID:                 "ugu_new",
		UnitGroupID:        "ug_test",
		UnitID:             "un_child",
		DiscountPercentage: "10",
		DiscountFixed:      "0",
	}
	unitGroupRepo.EXPECT().
		UpsertUnitGroupUnit(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, p domain.UpsertUnitGroupUnitParams) (*domain.UnitGroupUnit, *apierror.APIError) {
			assert.NotEmpty(t, id)
			assert.Equal(t, "ug_test", p.UnitGroupID)
			assert.Equal(t, "un_child", p.UnitID)
			return out, nil
		})

	idempotencyMed.EXPECT().
		CacheSuccessResponse(gomock.Any(), "idk_test", out).
		Return(nil).
		Times(1)

	svc := NewUnitGroupSvc(&UnitGroupSvcConfig{
		Repos:           repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: repoFactory},
	})

	ctx := idempotencyCtx(unitGroupInternalIdentityCtx(accountID))
	result, apiErr := svc.UpsertUnitGroupUnit(ctx, domain.UpsertUnitGroupUnitParams{
		UnitGroupID:        "ug_test",
		UnitID:             "un_child",
		DiscountPercentage: "10",
		DiscountFixed:      "0",
		IsVisible:          true,
	})

	assert.Nil(t, apiErr)
	assert.Equal(t, out, result)
}
