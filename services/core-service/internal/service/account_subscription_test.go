package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/messaging"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type subscriptionOutboxRepo struct {
	created []messaging.OutboxMessageInput
}

func (r *subscriptionOutboxRepo) Create(_ context.Context, in messaging.OutboxMessageInput) (int64, error) {
	r.created = append(r.created, in)
	return int64(len(r.created)), nil
}

func newSubscriptionFixture(t *testing.T) (*accountSvcImpl, *repositorymock.MockAccountRepo, *subscriptionOutboxRepo) {
	t.Helper()
	ctrl := gomock.NewController(t)
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	outbox := &subscriptionOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(repositorymock.NewMockAccountUserRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewAccountRelationRepo().Return(repositorymock.NewMockAccountRelationRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewRolePermissionRepo().Return(repositorymock.NewMockRolePermissionRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewRoleRepo().Return(repositorymock.NewMockRoleRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	accountRepo.EXPECT().UpdateSubscription(gomock.Any(), "ac_1", gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)

	return &accountSvcImpl{accountRepo: accountRepo, repos: repos, txManager: &stubTxManager{factory: repos}}, accountRepo, outbox
}

// Billing applies Stripe webhooks with no caller identity: nobody made the change. The status must still be saved.
func TestUpdateAccountSubscription_SavesWebhookStatusChangeWithoutIdentity(t *testing.T) {
	svc, _, outbox := newSubscriptionFixture(t)
	pastDue := "past_due"

	apiErr := svc.UpdateAccountSubscription(context.Background(), "ac_1", &pastDue, "", nil, nil, nil, nil, nil, nil, nil, nil)

	require.Nil(t, apiErr, "a webhook-driven status change must persist")
	require.Empty(t, outbox.created, "there is no caller to attribute an audit event to")
}

func TestUpdateAccountSubscription_AuditsCallerInitiatedChange(t *testing.T) {
	svc, _, outbox := newSubscriptionFixture(t)
	pastDue := "past_due"
	accountID := "ac_1"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{ID: "us_1", AccountID: &accountID, RelationType: types.IdentityRelationTypeInternal},
	})

	apiErr := svc.UpdateAccountSubscription(ctx, accountID, &pastDue, "", nil, nil, nil, nil, nil, nil, nil, nil)

	require.Nil(t, apiErr)
	require.Len(t, outbox.created, 1, "a change a caller made must still be audited")
}
