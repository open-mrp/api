package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
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

// audits returns the audit events published, with the identity each is attributed to.
func (r *subscriptionOutboxRepo) audits(t *testing.T) ([]audit.PublishedEvent, []*types.Identity) {
	t.Helper()
	var events []audit.PublishedEvent
	var identities []*types.Identity
	for _, msg := range r.created {
		if msg.RoutingKey != string(contracts.PlatformEventAuditLogged) {
			continue
		}
		var event audit.PublishedEvent
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &event))
		events = append(events, event)
		identities = append(identities, msg.Payload.Identity)
	}
	return events, identities
}

type subscriptionFixture struct {
	svc             *accountSvcImpl
	accountRepo     *repositorymock.MockAccountRepo
	accountUserRepo *repositorymock.MockAccountUserRepo
	outbox          *subscriptionOutboxRepo
}

func newSubscriptionFixture(t *testing.T) *subscriptionFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	accountRepo := repositorymock.NewMockAccountRepo(ctrl)
	accountUserRepo := repositorymock.NewMockAccountUserRepo(ctrl)
	outbox := &subscriptionOutboxRepo{}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAccountRepo().Return(accountRepo).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(accountUserRepo).AnyTimes()
	repos.EXPECT().NewAccountRelationRepo().Return(repositorymock.NewMockAccountRelationRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewRolePermissionRepo().Return(repositorymock.NewMockRolePermissionRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewRoleRepo().Return(repositorymock.NewMockRoleRepo(ctrl)).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	svc := &accountSvcImpl{accountRepo: accountRepo, accountUserRepo: accountUserRepo, repos: repos, outboxRepo: outbox, txManager: &stubTxManager{factory: repos}}
	return &subscriptionFixture{svc: svc, accountRepo: accountRepo, accountUserRepo: accountUserRepo, outbox: outbox}
}

func (f *subscriptionFixture) expectStatusUpdate() {
	f.accountRepo.EXPECT().UpdateSubscription(gomock.Any(), "ac_1", gomock.Any(), "", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
}

// Billing applies Stripe webhooks under a system identity, so the change is audited as Stripe's.
func TestUpdateAccountSubscription_AuditsWebhookChangeAsStripe(t *testing.T) {
	f := newSubscriptionFixture(t)
	f.expectStatusUpdate()
	pastDue := "past_due"
	ctx := appctx.WithSystemIdentity(context.Background(), constants.SystemActorStripe, "ac_1")

	apiErr := f.svc.UpdateAccountSubscription(ctx, "ac_1", &pastDue, "", nil, nil, nil, nil, nil, nil, nil, nil)

	require.Nil(t, apiErr, "a webhook-driven status change must persist")
	events, identities := f.outbox.audits(t)
	require.Len(t, events, 1)
	require.Equal(t, constants.ObjectTypeAccount, events[0].ResourceType)
	require.Equal(t, "ac_1", events[0].ResourceID)
	require.True(t, identities[0].IsSystem())
	require.Equal(t, "system:stripe", identities[0].Actor.ID)
}

func TestUpdateAccountSubscription_AuditsCallerInitiatedChange(t *testing.T) {
	f := newSubscriptionFixture(t)
	f.expectStatusUpdate()
	pastDue := "past_due"
	accountID := "ac_1"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{ID: "us_1", AccountID: &accountID, RelationType: types.IdentityRelationTypeInternal},
	})

	apiErr := f.svc.UpdateAccountSubscription(ctx, accountID, &pastDue, "", nil, nil, nil, nil, nil, nil, nil, nil)

	require.Nil(t, apiErr)
	events, identities := f.outbox.audits(t)
	require.Len(t, events, 1, "a change a caller made must still be audited")
	require.Equal(t, "us_1", identities[0].Actor.ID)
}

// Every caller of this internal RPC carries an identity; one that doesn't is a bug, and the change must not land unaudited.
func TestUpdateAccountSubscription_RefusesAnUnattributedChange(t *testing.T) {
	f := newSubscriptionFixture(t)
	f.expectStatusUpdate()
	pastDue := "past_due"

	apiErr := f.svc.UpdateAccountSubscription(context.Background(), "ac_1", &pastDue, "", nil, nil, nil, nil, nil, nil, nil, nil)

	require.NotNil(t, apiErr)
}

// expectPlanChange stubs a webhook moving ac_1 from pro to the given plan, whose seat limit is seats.
func (f *subscriptionFixture) expectPlanChange(planCode string, seats *int32) {
	f.accountRepo.EXPECT().GetPlanCode(gomock.Any(), "ac_1").Return(constants.PlanCode("pro"), nil).Times(1)
	f.accountRepo.EXPECT().GetPlanTypeIDByCode(gomock.Any(), planCode).Return("pt_1", nil).Times(1)
	f.accountRepo.EXPECT().UpdateSubscription(gomock.Any(), "ac_1", gomock.Any(), planCode, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	f.accountRepo.EXPECT().GetSeatLimitByPlanCode(gomock.Any(), planCode).Return(seats, nil).Times(1)
}

// Reactivating users the plan has room for again changes who can sign in, so each one is audited, attributed to Stripe and naming the plan.
func TestUpdateAccountSubscription_AuditsUsersReactivatedByAPlanChange(t *testing.T) {
	f := newSubscriptionFixture(t)
	f.expectPlanChange("enterprise", nil)
	f.accountUserRepo.EXPECT().LockSeatStatuses(gomock.Any(), "ac_1").Return(map[string]constants.AccountUserStatus{
		"acus_a": constants.AccountUserStatusActive,
		"acus_b": constants.AccountUserStatusDisabled,
	}, nil).Times(1)
	f.accountUserRepo.EXPECT().ReactivateUsers(gomock.Any(), "ac_1", gomock.Any()).Return(int64(1), nil).Times(1)
	f.accountUserRepo.EXPECT().LockSeatStatuses(gomock.Any(), "ac_1").Return(map[string]constants.AccountUserStatus{
		"acus_a": constants.AccountUserStatusActive,
		"acus_b": constants.AccountUserStatusActive,
	}, nil).Times(1)
	ctx := appctx.WithSystemIdentity(context.Background(), constants.SystemActorStripe, "ac_1")

	require.Nil(t, f.svc.UpdateAccountSubscription(ctx, "ac_1", nil, "enterprise", nil, nil, nil, nil, nil, nil, nil, nil))

	events, identities := f.outbox.audits(t)
	require.Len(t, events, 2, "the account's plan change and the one reactivated user")
	seat := events[1]
	require.Equal(t, constants.ObjectTypeAccountUser, seat.ResourceType)
	require.Equal(t, "acus_b", seat.ResourceID)
	require.Len(t, seat.Changes, 1)
	require.Equal(t, "status_code", seat.Changes[0].Field)
	require.JSONEq(t, `"disabled"`, string(seat.Changes[0].OldValue))
	require.JSONEq(t, `"active"`, string(seat.Changes[0].NewValue))
	require.Equal(t, "enterprise", seat.Metadata["plan_code"])
	require.Equal(t, "system:stripe", identities[1].Actor.ID)
}

// A webhook downgrade has no user to keep active, so it deactivates nobody — and audits nobody.
func TestUpdateAccountSubscription_WebhookDowngradeDeactivatesNobody(t *testing.T) {
	f := newSubscriptionFixture(t)
	seats := int32(1)
	f.expectPlanChange("starter", &seats)
	statuses := map[string]constants.AccountUserStatus{"acus_a": constants.AccountUserStatusActive, "acus_b": constants.AccountUserStatusActive}
	f.accountUserRepo.EXPECT().LockSeatStatuses(gomock.Any(), "ac_1").Return(statuses, nil).Times(2)
	f.accountUserRepo.EXPECT().CountActive(gomock.Any(), "ac_1").Return(int64(2), nil).Times(1)
	ctx := appctx.WithSystemIdentity(context.Background(), constants.SystemActorStripe, "ac_1")

	require.Nil(t, f.svc.UpdateAccountSubscription(ctx, "ac_1", nil, "starter", nil, nil, nil, nil, nil, nil, nil, nil))

	events, _ := f.outbox.audits(t)
	require.Len(t, events, 1, "only the account's plan change")
	require.Equal(t, constants.ObjectTypeAccount, events[0].ResourceType)
}

// A user's downgrade deactivates the least-recently-used others, and each deactivation is audited as the user's.
func TestUpdateAccountSubscription_AuditsUsersDeactivatedByAUsersDowngrade(t *testing.T) {
	f := newSubscriptionFixture(t)
	seats := int32(1)
	f.expectPlanChange("starter", &seats)
	f.accountUserRepo.EXPECT().LockSeatStatuses(gomock.Any(), "ac_1").Return(map[string]constants.AccountUserStatus{
		"acus_me": constants.AccountUserStatusActive, "acus_other": constants.AccountUserStatusActive,
	}, nil).Times(1)
	f.accountUserRepo.EXPECT().CountActive(gomock.Any(), "ac_1").Return(int64(2), nil).Times(1)
	f.accountUserRepo.EXPECT().DeactivateExcept(gomock.Any(), "ac_1", "us_me", int32(1)).Return(int64(1), nil).Times(1)
	f.accountUserRepo.EXPECT().EnsureActive(gomock.Any(), "ac_1", "us_me").Return(nil).Times(1)
	f.accountUserRepo.EXPECT().LockSeatStatuses(gomock.Any(), "ac_1").Return(map[string]constants.AccountUserStatus{
		"acus_me": constants.AccountUserStatusActive, "acus_other": constants.AccountUserStatusDisabled,
	}, nil).Times(1)
	accountID := "ac_1"
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor:  &types.IdentityActor{ID: "us_me", AccountID: &accountID, RelationType: types.IdentityRelationTypeInternal},
	})

	require.Nil(t, f.svc.UpdateAccountSubscription(ctx, accountID, nil, "starter", nil, nil, nil, nil, nil, nil, nil, nil))

	events, identities := f.outbox.audits(t)
	require.Len(t, events, 2)
	require.Equal(t, "acus_other", events[1].ResourceID)
	require.JSONEq(t, `"disabled"`, string(events[1].Changes[0].NewValue))
	require.Equal(t, "us_me", identities[1].Actor.ID)
}
