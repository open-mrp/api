package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	publishermock "github.com/open-mrp/api/services/core-service/internal/domain/mock/publisher"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/crypto"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	passwordTestAccountID     = "acc_pw_test"
	passwordTestActorID       = "usr_pw_admin"
	passwordTestAccountUserID = "au_pw_target"
	passwordTestTargetUserID  = "usr_pw_target"
	passwordTestRequesterPass = "Requester-Pass1!"
	passwordTestNewPass       = "New-Password1!"
)

func passwordTestCtx() context.Context {
	accountID := passwordTestAccountID
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           passwordTestActorID,
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions:  map[string]bool{"team_users:update": true},
		},
	})
}

func TestUpdateAccountUserPassword_TargetEligibility(t *testing.T) {
	t.Parallel()

	email := "someone@example.com"
	empty := ""
	scanner := string(constants.RoleTypeScanner)
	custom := string(constants.RoleTypeCustom)
	admin := string(constants.RoleTypeAdmin)

	cases := []struct {
		name        string
		email       *string
		roleType    *string
		wantUpdated bool
	}{
		{name: "scanner without email", email: nil, roleType: &scanner, wantUpdated: true},
		{name: "custom role without email", email: nil, roleType: &custom, wantUpdated: true},
		{name: "no role without email", email: nil, roleType: nil, wantUpdated: true},
		{name: "empty email is treated as no email", email: &empty, roleType: &custom, wantUpdated: true},
		{name: "user with email is rejected", email: &email, roleType: &custom, wantUpdated: false},
		{name: "scanner with email is rejected", email: &email, roleType: &scanner, wantUpdated: false},
		{name: "admin without email is rejected", email: nil, roleType: &admin, wantUpdated: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			requesterHash, err := crypto.HashBcrypt(passwordTestRequesterPass)
			require.NoError(t, err)

			userRepo := repositorymock.NewMockUserRepo(ctrl)
			accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
			repoFactory := factorymock.NewMockRepoFactory(ctrl)
			repoFactory.EXPECT().NewUserRepo().Return(userRepo).AnyTimes()
			repoFactory.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
			repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

			idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)
			mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
			mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotencyMed}).AnyTimes()

			idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
				Return(&domain.IdempotencyKey{TypeID: "idk_pw", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
			idempotencyMed.EXPECT().CacheErrorResponse(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()

			userRepo.EXPECT().GetHashedPassword(gomock.Any(), passwordTestActorID).Return(requesterHash, nil)
			accountUserRepo.EXPECT().GetDetailByAccountAndID(gomock.Any(), passwordTestAccountID, passwordTestAccountUserID, gomock.Any()).
				Return(&domain.AccountUserDetail{
					ID:       passwordTestAccountUserID,
					UserID:   passwordTestTargetUserID,
					Email:    tc.email,
					RoleType: tc.roleType,
				}, nil)

			if tc.wantUpdated {
				userRepo.EXPECT().UpdatePassword(gomock.Any(), passwordTestTargetUserID, gomock.Any()).Return(nil)
				idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			}

			svc := NewAccountUserSvc(&AccountUserSvcConfig{
				Repos:                 repoFactory,
				MediatorFactory:       mediatorFactory,
				TxManager:             &stubTxManager{factory: repoFactory},
				NotificationPublisher: publishermock.NewMockNotificationPublisher(ctrl),
				BillingPublisher:      publishermock.NewMockBillingPublisher(ctrl),
				S3Client:              &capturingObjectStore{},
				PlatformMode:          constants.PlatformModeTest,
			})

			apiErr := svc.UpdateAccountUserPassword(passwordTestCtx(), passwordTestAccountUserID, passwordTestRequesterPass, passwordTestNewPass)
			if tc.wantUpdated {
				require.Nil(t, apiErr)
				return
			}
			require.NotNil(t, apiErr)
			require.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
		})
	}
}
