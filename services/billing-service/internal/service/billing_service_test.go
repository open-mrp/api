package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	clientmock "github.com/augno/api/services/billing-service/internal/domain/mock/client"
	factorymock "github.com/augno/api/services/billing-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/billing-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/billing-service/internal/domain/mock/repository"
	servicemock "github.com/augno/api/services/billing-service/internal/domain/mock/service"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"

	authtypes "github.com/augno/api/services/auth-service/pkg/types"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type BillingSvcTestSuite struct {
	suite.Suite
	billingSvc         domain.BillingSvc
	repoFactory        *factorymock.MockRepoFactory
	pricingPlanRepo    *repositorymock.MockPricingPlanRepo
	accountUsageRepo   *repositorymock.MockAccountUsageRepo
	tokenBillingRepo   *repositorymock.MockAgentTokenBillingRepo
	idempotencyKeyRepo *repositorymock.MockIdempotencyKeyRepo
	stripeClient       *clientmock.MockStripeClient
	coreClient         *servicemock.MockCoreClient
	notificationClient *clientmock.MockNotificationClient
	idempotencyMed     *mediatormock.MockIdempotencyMed
	ctrl               *gomock.Controller
}

func (s *BillingSvcTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.pricingPlanRepo = repositorymock.NewMockPricingPlanRepo(s.ctrl)
	s.accountUsageRepo = repositorymock.NewMockAccountUsageRepo(s.ctrl)
	s.tokenBillingRepo = repositorymock.NewMockAgentTokenBillingRepo(s.ctrl)
	s.idempotencyKeyRepo = repositorymock.NewMockIdempotencyKeyRepo(s.ctrl)
	s.stripeClient = clientmock.NewMockStripeClient(s.ctrl)
	s.coreClient = servicemock.NewMockCoreClient(s.ctrl)
	s.notificationClient = clientmock.NewMockNotificationClient(s.ctrl)
	s.idempotencyMed = mediatormock.NewMockIdempotencyMed(s.ctrl)

	s.repoFactory = factorymock.NewMockRepoFactory(s.ctrl)
	s.repoFactory.EXPECT().NewPricingPlanRepo().Return(s.pricingPlanRepo).AnyTimes()
	s.repoFactory.EXPECT().NewAccountUsageRepo().Return(s.accountUsageRepo).AnyTimes()
	s.repoFactory.EXPECT().NewAgentTokenBillingRepo().Return(s.tokenBillingRepo).AnyTimes()
	s.repoFactory.EXPECT().NewIdempotencyKeyRepo().Return(s.idempotencyKeyRepo).AnyTimes()

	s.billingSvc = NewBillingSvc(&BillingSvcConfig{
		Repos:              s.repoFactory,
		StripeClient:       s.stripeClient,
		CoreClient:         s.coreClient,
		FrontendURL:        "https://app.example.com",
		NotificationClient: s.notificationClient,
		IdempotencyMed:     s.idempotencyMed,
	})
}

func (s *BillingSvcTestSuite) TearDownSuite() {
	s.ctrl.Finish()
}

func TestBillingSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BillingSvcTestSuite))
}

// --- helpers ---

var anyCtx = gomock.Any()

func testIdentityCtx() context.Context {
	return appctx.WithIdentity(context.Background(), &authtypes.Identity{Type: authtypes.IdentityActorTypeUser})
}

// testUserActorCtx carries a fully populated user actor, for the paths that need to know which
// person is acting rather than merely that the caller is authenticated.
func testUserActorCtx() context.Context {
	return appctx.WithIdentity(context.Background(), &authtypes.Identity{
		Type:  authtypes.IdentityActorTypeUser,
		Actor: &authtypes.IdentityActor{ID: "usr_1"},
	})
}

// testAPIKeyActorCtx carries an API-key actor, which names no person.
func testAPIKeyActorCtx() context.Context {
	return appctx.WithIdentity(context.Background(), &authtypes.Identity{
		Type:  authtypes.IdentityActorTypeAPIKey,
		Actor: &authtypes.IdentityActor{ID: "apky_1"},
	})
}

// expectIdempotency sets up the standard idempotency expectations for a mutating method:
// UpsertIdempotencyKey returns a fresh key at RecoveryPointStarted, and
// CacheSuccessResponse / SetResponse succeed.
func (s *BillingSvcTestSuite) expectIdempotency() {
	s.idempotencyMed.EXPECT().UpsertIdempotencyKey(anyCtx, gomock.Any()).Return(&domain.IdempotencyKey{
		TypeID:        "sikey_test",
		RecoveryPoint: string(domain.RecoveryPointStarted),
	}, nil).AnyTimes()
	s.idempotencyMed.EXPECT().CacheSuccessResponse(anyCtx, "sikey_test", gomock.Any()).Return(nil).AnyTimes()
	s.idempotencyMed.EXPECT().CacheErrorResponse(anyCtx, "sikey_test", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError {
			return apiErr
		},
	).AnyTimes()
	s.idempotencyKeyRepo.EXPECT().SetResponse(anyCtx, "sikey_test", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

func testPricingPlan(planCode string, stripePlanID *string) *domain.PricingPlan {
	return &domain.PricingPlan{
		ID:                  1,
		CreatedAt:           time.Now(),
		TypeID:              "plan_type_" + planCode,
		Name:                planCode + " Plan",
		PlanTypeCode:        planCode,
		PricePerSeat:        10.0,
		SeatMinimum:         new(1),
		StripePricingPlanID: stripePlanID,
	}
}

func testStripePricingPlan() *domain.StripePricingPlan {
	return &domain.StripePricingPlan{
		ID:                    "spp_123",
		LiveVersion:           "v1",
		LicenseFeeComponentID: "comp_license",
	}
}

// --- SubscribeToPricingPlan tests ---

// expectSubscribeLookup sets up the common early-lookup expectations for SubscribeToPricingPlan:
// GetPlanByCode, GetAccountByStripeCustomerID, and GetAccountSubscriptionInfo.
// Returns the subscription info so tests can customize the billing state.
func (s *BillingSvcTestSuite) expectSubscribeLookup(stripeCustomerID, planCode string, plan *domain.PricingPlan, subInfo *domain.AccountSubscriptionInfo) {
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, planCode).Return(plan, nil)
	s.coreClient.EXPECT().GetAccountByStripeCustomerID(anyCtx, stripeCustomerID).Return("acct_1", planCode, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(subInfo, nil)
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_HappyPath() {
	s.expectIdempotency()
	planCode := "starter"
	stripeCustomerID := "cus_abc"
	plan := testPricingPlan(planCode, new("spp_123"))
	stripePlan := testStripePricingPlan()

	// No existing profile/cadence/subscription — creates new ones
	s.expectSubscribeLookup(stripeCustomerID, planCode, plan, &domain.AccountSubscriptionInfo{})

	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, stripeCustomerID, "subscribe_"+stripeCustomerID+"_profile").Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", "subscribe_"+stripeCustomerID+"_cadence").Return("bc_1", nil)

	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, planCode).Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), "subscribe_"+stripeCustomerID+"_intent").Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 1000}, nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(1000), "usd", stripeCustomerID, "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_1"},
	}, nil)

	servicingStatus := "active"
	collectionStatus := "current"
	profileID := "bp_1"
	cadenceID := "bc_1"
	subID := "pps_1"
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, "subscribe_persist_"+stripeCustomerID, "acct_1",
		nil, "", nil, nil, nil, &profileID, &cadenceID, &subID, &servicingStatus, &collectionStatus,
	).Return(nil)

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), stripeCustomerID, planCode)
	s.Nil(apiErr)
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_HappyPath_ReusesExistingProfileCadence() {
	s.expectIdempotency()
	planCode := "starter"
	stripeCustomerID := "cus_abc"
	plan := testPricingPlan(planCode, new("spp_123"))
	stripePlan := testStripePricingPlan()

	// Profile/cadence already exist from SetupBillingProfile — reused, no Stripe create calls
	s.expectSubscribeLookup(stripeCustomerID, planCode, plan, &domain.AccountSubscriptionInfo{
		BillingProfileID: new("bp_existing"),
		BillingCadenceID: new("bc_existing"),
	})

	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, planCode).Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_existing", gomock.Any(), "subscribe_"+stripeCustomerID+"_intent").Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 1000}, nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(1000), "usd", stripeCustomerID, "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_1"},
	}, nil)

	servicingStatus := "active"
	collectionStatus := "current"
	profileID := "bp_existing"
	cadenceID := "bc_existing"
	subID := "pps_1"
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, "subscribe_persist_"+stripeCustomerID, "acct_1",
		nil, "", nil, nil, nil, &profileID, &cadenceID, &subID, &servicingStatus, &collectionStatus,
	).Return(nil)

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), stripeCustomerID, planCode)
	s.Nil(apiErr)
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_AlreadySubscribed_ShortCircuits() {
	s.expectIdempotency()
	planCode := "starter"
	plan := testPricingPlan(planCode, new("spp_123"))

	// Subscription already exists — should return nil immediately
	s.expectSubscribeLookup("cus_abc", planCode, plan, &domain.AccountSubscriptionInfo{
		PricingPlanSubscriptionID: new("pps_existing"),
	})

	// No Stripe calls expected

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", planCode)
	s.Nil(apiErr)
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_FreePlan_NoOp() {
	s.expectIdempotency()
	plan := testPricingPlan("free", nil)

	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "free").Return(plan, nil)

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "free")
	s.Nil(apiErr)
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_CreateBillingProfileFails() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))

	s.expectSubscribeLookup("cus_abc", "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_abc", gomock.Any()).Return("", fmt.Errorf("stripe error"))

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "starter")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "billing profile")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_CreateBillingCadenceFails() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))

	s.expectSubscribeLookup("cus_abc", "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_abc", gomock.Any()).Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", gomock.Any()).Return("", fmt.Errorf("stripe error"))

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "starter")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "billing cadence")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_CommitFails() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))
	stripePlan := testStripePricingPlan()

	s.expectSubscribeLookup("cus_abc", "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_abc", gomock.Any()).Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", gomock.Any()).Return("bc_1", nil)
	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 1000}, nil)
	piID := "pi_1"
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(1000), "usd", "cus_abc", gomock.Any()).Return(piID, nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", &piID, gomock.Any()).Return(nil, fmt.Errorf("commit failed"))
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_1").Return(nil)

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "starter")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "commit billing intent")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_PlanNotFound() {
	s.expectIdempotency()
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "nonexistent").Return(nil, apierror.NewResourceNotFoundError("plan not found"))

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "nonexistent")
	s.NotNil(apiErr)
}

// --- SwitchPlan tests ---

func (s *BillingSvcTestSuite) TestSwitchPlan_ModifyExistingSubscription() {
	s.expectIdempotency()
	accountID := "acct_1"
	planID := "plan_type_pro"
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID:          new("bc_1"),
		PricingPlanSubscriptionID: new("pps_old"),
	}, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(3, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, actions []domain.BillingIntentAction, _ string) (string, error) {
			s.Equal("modify", actions[0].Type)
			s.Equal("pps_old", actions[0].SubscriptionID)
			return "bi_1", nil
		})
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 2000}, nil)
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, accountID).Return(new("cus_1"), nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(2000), "usd", "cus_1", "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_new"},
	}, nil)
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "pro", nil, nil, nil, nil, nil, new("pps_new"), new("active"), new("current"),
	).Return(nil)

	result, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), accountID, planID)
	s.Nil(apiErr)
	s.True(result.Success)
	s.NotNil(result.IntentID)
}

func (s *BillingSvcTestSuite) TestSwitchPlan_SubscribeNewPlan() {
	s.expectIdempotency()
	accountID := "acct_1"
	planID := "plan_type_starter"
	plan := testPricingPlan("starter", new("spp_starter"))
	stripePlan := testStripePricingPlan()

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_starter").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(1, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, actions []domain.BillingIntentAction, _ string) (string, error) {
			s.Equal("subscribe", actions[0].Type)
			return "bi_1", nil
		})
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 1000}, nil)
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, accountID).Return(new("cus_1"), nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(1000), "usd", "cus_1", "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_1"},
	}, nil)
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "starter", nil, nil, nil, nil, nil, new("pps_1"), new("active"), new("current"),
	).Return(nil)

	result, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), accountID, planID)
	s.Nil(apiErr)
	s.True(result.Success)
}

func (s *BillingSvcTestSuite) TestSwitchPlan_DowngradeToFree() {
	s.expectIdempotency()
	accountID := "acct_1"
	planID := "plan_type_free"
	plan := testPricingPlan("free", nil)

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID:          new("bc_1"),
		PricingPlanSubscriptionID: new("pps_old"),
	}, nil)

	// Deactivate existing subscription
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, actions []domain.BillingIntentAction, _ string) (string, error) {
			s.Equal("deactivate", actions[0].Type)
			s.Equal("pps_old", actions[0].SubscriptionID)
			return "bi_deact", nil
		})
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_deact").Return(&domain.BillingIntentReservation{}, nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_deact", (*string)(nil), gomock.Any()).Return(&domain.BillingIntentCommitResult{}, nil)

	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "free", nil, nil, nil, nil, nil, nil, nil, nil,
	).Return(nil)

	result, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), accountID, planID)
	s.Nil(apiErr)
	s.True(result.Success)
}

func (s *BillingSvcTestSuite) TestSwitchPlan_PlanNotFound() {
	s.expectIdempotency()
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "bad_plan").Return(nil, apierror.NewResourceNotFoundError("plan not found"))

	_, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), "acct_1", "bad_plan")
	s.NotNil(apiErr)
}

func (s *BillingSvcTestSuite) TestSwitchPlan_NoCadence() {
	s.expectIdempotency()
	plan := testPricingPlan("pro", new("spp_pro"))

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_pro").Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{}, nil)

	_, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), "acct_1", "plan_type_pro")
	s.NotNil(apiErr)
	s.Contains(apiErr.PublicMessage, "billing cadence")
}

// --- PreviewPlanChange tests ---

func (s *BillingSvcTestSuite) TestPreviewPlanChange_HappyPath() {
	accountID := "acct_1"
	planID := "plan_type_pro"
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(2, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_preview", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_preview").Return(&domain.BillingIntentReservation{
		IntentID:  "bi_preview",
		NetAmount: 2000,
		LineItems: []domain.BillingIntentLineItem{{Description: "Pro plan", Amount: 2000}},
	}, nil)
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_preview").Return(nil)

	preview, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), accountID, planID)
	s.Nil(apiErr)
	s.Equal(int64(2000), preview.NetAmount)
	s.Equal("$20.00", preview.FormattedNetAmount)
	s.Len(preview.LineItems, 1)
}

func (s *BillingSvcTestSuite) TestPreviewPlanChange_FreePlan() {
	plan := testPricingPlan("free", nil)

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_free").Return(plan, nil)

	preview, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), "acct_1", "plan_type_free")
	s.Nil(apiErr)
	s.Equal(int64(0), preview.NetAmount)
}

func (s *BillingSvcTestSuite) TestPreviewPlanChange_ReserveFails() {
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_pro").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, "acct_1").Return(1, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(nil, fmt.Errorf("reserve failed"))
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_1").Return(nil)

	_, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), "acct_1", "plan_type_pro")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "reserve billing intent")
}

func (s *BillingSvcTestSuite) TestPreviewPlanChange_ConflictVoidAndRetrySucceeds() {
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_pro").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, "acct_1").Return(2, nil)

	// First call returns conflict, second succeeds.
	conflictErr := &domain.ErrBillingIntentConflict{
		ConflictingIntentID: "bilint_test_stale",
		Err:                 fmt.Errorf("stripe error"),
	}
	gomock.InOrder(
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("", conflictErr),
		s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bilint_test_stale").Return(nil),
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_retry", nil),
	)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_retry").Return(&domain.BillingIntentReservation{
		IntentID:  "bi_retry",
		NetAmount: 1500,
		LineItems: []domain.BillingIntentLineItem{{Description: "Pro plan", Amount: 1500}},
	}, nil)
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_retry").Return(nil)

	preview, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), "acct_1", "plan_type_pro")
	s.Nil(apiErr)
	s.Equal(int64(1500), preview.NetAmount)
}

func (s *BillingSvcTestSuite) TestPreviewPlanChange_ConflictVoidFailsButRetrySucceeds() {
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_pro").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, "acct_1").Return(1, nil)

	conflictErr := &domain.ErrBillingIntentConflict{
		ConflictingIntentID: "bilint_test_stale",
		Err:                 fmt.Errorf("stripe error"),
	}
	// Void fails (intent already committed), but retry succeeds because conflict cleared.
	gomock.InOrder(
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("", conflictErr),
		s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bilint_test_stale").Return(fmt.Errorf("intent already committed")),
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_retry", nil),
	)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_retry").Return(&domain.BillingIntentReservation{
		IntentID:  "bi_retry",
		NetAmount: 1000,
		LineItems: []domain.BillingIntentLineItem{{Description: "Pro plan", Amount: 1000}},
	}, nil)
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_retry").Return(nil)

	preview, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), "acct_1", "plan_type_pro")
	s.Nil(apiErr)
	s.Equal(int64(1000), preview.NetAmount)
}

func (s *BillingSvcTestSuite) TestPreviewPlanChange_ConflictRetryAlsoFails_FallsBackToLocalEstimate() {
	plan := testPricingPlan("pro", new("spp_pro"))
	currentPlan := testPricingPlan("starter", new("spp_starter"))
	stripePlan := testStripePricingPlan()

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, "acct_1").Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID: new("bc_1"),
	}, nil)
	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, "plan_type_pro").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, "acct_1").Return(1, nil)

	conflictErr := &domain.ErrBillingIntentConflict{
		ConflictingIntentID: "bilint_test_stale",
		Err:                 fmt.Errorf("stripe error"),
	}
	// Both void and retry fail — falls back to local estimate.
	gomock.InOrder(
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("", conflictErr),
		s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bilint_test_stale").Return(fmt.Errorf("void failed")),
		s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("", fmt.Errorf("still conflicting")),
	)
	// Fallback fetches current plan for local proration.
	s.accountUsageRepo.EXPECT().GetAccountNameAndPlanCode(anyCtx, "acct_1").Return("Test Account", "starter", nil)
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(currentPlan, nil)

	preview, apiErr := s.billingSvc.PreviewPlanChange(context.Background(), "acct_1", "plan_type_pro")
	s.Nil(apiErr)
	s.True(preview.IsEstimate)
	s.Len(preview.LineItems, 2)
	// pro plan: $10/seat * 1 seat = $10.00 (1000 cents), starter: same = 1000 cents, net = 0
	s.Equal(int64(0), preview.NetAmount)
}

// --- EnsureBillingCustomer tests ---

func (s *BillingSvcTestSuite) TestEnsureBillingCustomer_ExistingCustomer() {
	s.expectIdempotency()
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, "acct_1").Return(new("cus_existing"), nil)

	result, apiErr := s.billingSvc.EnsureBillingCustomer(testIdentityCtx(), "acct_1")
	s.Nil(apiErr)
	s.Equal("cus_existing", result.StripeCustomerID)
	s.False(result.Created)
}

func (s *BillingSvcTestSuite) TestEnsureBillingCustomer_CreatesNew() {
	s.expectIdempotency()
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, "acct_1").Return(nil, nil)
	s.accountUsageRepo.EXPECT().GetAccountNameAndPlanCode(anyCtx, "acct_1").Return("Test Account", "starter", nil)
	s.accountUsageRepo.EXPECT().GetAdminEmailByAccountID(anyCtx, "acct_1").Return("admin@test.com", nil)
	s.stripeClient.EXPECT().CreateCustomer(anyCtx, "admin@test.com", "Test Account", "ensure_customer_acct_1", gomock.Any()).
		Return(&domain.StripeCustomer{ID: "cus_new"}, nil)
	s.accountUsageRepo.EXPECT().UpdateStripeCustomerIDByAccountID(anyCtx, "cus_new", "acct_1").Return(nil)

	result, apiErr := s.billingSvc.EnsureBillingCustomer(testIdentityCtx(), "acct_1")
	s.Nil(apiErr)
	s.Equal("cus_new", result.StripeCustomerID)
	s.True(result.Created)
}

// --- GetAccountUsage tests ---

func (s *BillingSvcTestSuite) TestGetAccountUsage_HappyPath() {
	accountID := "acct_1"
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)

	s.accountUsageRepo.EXPECT().GetLimitsByAccountID(anyCtx, accountID).Return([]domain.PlanLimit{
		{Key: "seats_maximum", Value: new(10)},
		{Key: "sandboxes_maximum", Value: new(5)},
		{Key: "invoices_maximum", Value: new(100)},
		{Key: "batches_maximum", Value: new(50)},
	}, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(3, nil)
	s.accountUsageRepo.EXPECT().CountSandboxesByAccountID(anyCtx, accountID).Return(1, nil)

	servicingStatus := "active"
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		ServicingStatus:              &servicingStatus,
		SubscriptionCurrentPeriodEnd: &periodEnd,
	}, nil)
	s.accountUsageRepo.EXPECT().CountInvoicesByAccountID(anyCtx, accountID, gomock.Any()).Return(10, nil)
	s.accountUsageRepo.EXPECT().CountBatchesByAccountID(anyCtx, accountID, gomock.Any()).Return(2, nil)
	// Agent spend resolves the plan's rate card from its Stripe pricing plan; with no pricing plan the spend is 0 without any Stripe call.
	s.accountUsageRepo.EXPECT().GetAccountStripePricingPlanID(anyCtx, accountID).Return(nil, nil)

	usage, apiErr := s.billingSvc.GetAccountUsage(context.Background(), accountID)
	s.Nil(apiErr)
	s.Equal(3, usage.Seats.Current)
	s.Equal(10, *usage.Seats.Limit)
	s.Equal(1, usage.Sandboxes.Current)
	s.NotNil(usage.Subscription)
	s.Equal("active", usage.Subscription.ServicingStatus)
	s.Equal(int64(0), usage.EstimatedAgentSpendCents)
}

func (s *BillingSvcTestSuite) TestGetAccountUsage_FreePlanNoSubscription() {
	accountID := "acct_free"

	s.accountUsageRepo.EXPECT().GetLimitsByAccountID(anyCtx, accountID).Return([]domain.PlanLimit{
		{Key: "seats_maximum", Value: new(2)},
	}, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(1, nil)
	s.accountUsageRepo.EXPECT().CountSandboxesByAccountID(anyCtx, accountID).Return(0, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{}, nil)
	s.accountUsageRepo.EXPECT().CountInvoicesByAccountID(anyCtx, accountID, gomock.Any()).Return(0, nil)
	s.accountUsageRepo.EXPECT().CountBatchesByAccountID(anyCtx, accountID, gomock.Any()).Return(0, nil)
	// Free plan has no Stripe pricing plan, so agent spend is 0 without a Stripe call.
	s.accountUsageRepo.EXPECT().GetAccountStripePricingPlanID(anyCtx, accountID).Return(nil, nil)

	usage, apiErr := s.billingSvc.GetAccountUsage(context.Background(), accountID)
	s.Nil(apiErr)
	s.Nil(usage.Subscription)
	s.Equal(int64(0), usage.EstimatedAgentSpendCents)
}

func (s *BillingSvcTestSuite) TestGetAccountUsage_AgentSpendFromRateCard() {
	accountID := "acct_paid"
	now := time.Now().UTC()
	periodEnd := now.Add(30 * 24 * time.Hour)

	s.accountUsageRepo.EXPECT().GetLimitsByAccountID(anyCtx, accountID).Return([]domain.PlanLimit{
		{Key: "seats_maximum", Value: new(10)},
	}, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(3, nil)
	s.accountUsageRepo.EXPECT().CountSandboxesByAccountID(anyCtx, accountID).Return(1, nil)

	servicingStatus := "active"
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		ServicingStatus:              &servicingStatus,
		SubscriptionCurrentPeriodEnd: &periodEnd,
	}, nil)
	s.accountUsageRepo.EXPECT().CountInvoicesByAccountID(anyCtx, accountID, gomock.Any()).Return(0, nil)
	s.accountUsageRepo.EXPECT().CountBatchesByAccountID(anyCtx, accountID, gomock.Any()).Return(0, nil)

	// The account's plan carries a Stripe pricing plan; the resolved plan supplies the rate card (token pricing) plus the display name and base fee.
	s.accountUsageRepo.EXPECT().GetAccountStripePricingPlanID(anyCtx, accountID).Return(new("spp_123"), nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(&domain.StripePricingPlan{
		ID: "spp_123", LiveVersion: "v1", RateCardID: "rcd_pro",
		DisplayName: "Founder", BaseFeeCents: 100, BaseFeeInterval: "month",
	}, nil)
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, accountID).Return(new("cus_paid"), nil)
	s.stripeClient.EXPECT().GetAgentTokenSpendCents(anyCtx, "cus_paid", "rcd_pro", gomock.Any()).Return(int64(4231), nil)

	usage, apiErr := s.billingSvc.GetAccountUsage(context.Background(), accountID)
	s.Nil(apiErr)
	s.Equal(int64(4231), usage.EstimatedAgentSpendCents)
	s.Equal("Founder", usage.PlanName)
	s.Equal(int64(100), usage.BaseFeeCents)
	s.Equal("month", usage.BaseFeeInterval)
}

// --- SetupBillingProfile tests ---

func (s *BillingSvcTestSuite) TestSetupBillingProfile_HappyPath() {
	s.expectIdempotency()
	accountID := "acct_1"

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{}, nil)
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, accountID).Return(new("cus_1"), nil)
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_1", gomock.Any()).Return("bp_new", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_new", gomock.Any()).Return("bc_new", nil)
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "", nil, nil, nil, new("bp_new"), new("bc_new"), nil, nil, nil,
	).Return(nil)

	result, apiErr := s.billingSvc.SetupBillingProfile(testIdentityCtx(), accountID)
	s.Nil(apiErr)
	s.Equal("bp_new", result.ProfileID)
	s.Equal("bc_new", result.CadenceID)
}

func (s *BillingSvcTestSuite) TestSetupBillingProfile_AlreadyHasProfile() {
	s.expectIdempotency()
	accountID := "acct_1"

	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingProfileID: new("bp_existing"),
		BillingCadenceID: new("bc_existing"),
	}, nil)

	result, apiErr := s.billingSvc.SetupBillingProfile(testIdentityCtx(), accountID)
	s.Nil(apiErr)
	s.Equal("bp_existing", result.ProfileID)
	s.Equal("bc_existing", result.CadenceID)
}

// --- GetPlanByCode tests ---

func (s *BillingSvcTestSuite) TestGetPlanByCode_HappyPath() {
	plan := testPricingPlan("starter", new("spp_123"))

	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)
	s.pricingPlanRepo.EXPECT().GetPlanLimitsByTypeID(anyCtx, plan.TypeID).Return([]domain.PlanLimit{
		{Key: "seats_maximum", Value: new(10)},
	}, nil)

	result, apiErr := s.billingSvc.GetPlanByCode(context.Background(), "starter")
	s.Nil(apiErr)
	s.Equal("starter Plan", result.Name)
	s.Len(result.Limits, 1)
}

// --- CreateBillingPortalSession tests ---

func (s *BillingSvcTestSuite) TestCreateBillingPortalSession_HappyPath() {
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, "acct_1").Return(new("cus_1"), nil)
	s.stripeClient.EXPECT().CreateBillingPortalSession(anyCtx, "cus_1", gomock.Any()).
		Return(&domain.StripeBillingPortalSession{URL: "https://billing.stripe.com/session"}, nil)

	url, apiErr := s.billingSvc.CreateBillingPortalSession(context.Background(), "acct_1")
	s.Nil(apiErr)
	s.Equal("https://billing.stripe.com/session", url)
}

func (s *BillingSvcTestSuite) TestCreateBillingPortalSession_NoCustomer() {
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, "acct_1").Return(nil, nil)

	_, apiErr := s.billingSvc.CreateBillingPortalSession(context.Background(), "acct_1")
	s.NotNil(apiErr)
	s.Contains(apiErr.PublicMessage, "no Stripe customer")
}

// --- RequestEnterpriseUpgrade tests ---

func (s *BillingSvcTestSuite) TestRequestEnterpriseUpgrade_HappyPath() {
	input := domain.RequestEnterpriseUpgradeInput{
		AccountID: "acct_1",
		ActorID:   "usr_1",
		ActorName: "John Doe",
	}
	plan := testPricingPlan("pro", new("spp_pro"))

	s.accountUsageRepo.EXPECT().GetAccountNameAndPlanCode(anyCtx, "acct_1").Return("Test Corp", "pro", nil)
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "pro").Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetUserEmailByID(anyCtx, "usr_1").Return("john@test.com", nil, nil)
	s.notificationClient.EXPECT().SendEnterpriseRequest(anyCtx, "acct_1", "Test Corp", "pro Plan", "John Doe", "john@test.com").Return(nil)

	result, apiErr := s.billingSvc.RequestEnterpriseUpgrade(testUserActorCtx(), input)
	s.Nil(apiErr)
	s.True(result.Success)
}

// The inquiry carries the requester's name and email to sales, so a caller that names no person
// is refused up front rather than failing later on a user lookup that was never going to match.
func (s *BillingSvcTestSuite) TestRequestEnterpriseUpgrade_APIKeyIsRefused() {
	_, apiErr := s.billingSvc.RequestEnterpriseUpgrade(testAPIKeyActorCtx(), domain.RequestEnterpriseUpgradeInput{
		AccountID: "acct_1",
		ActorID:   "apky_1",
		ActorName: "A Key",
	})

	s.NotNil(apiErr)
	s.Contains(apiErr.PublicMessage, "must be a user")
}

// --- SubscribeToPricingPlan recoverability tests ---

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_GetAccountByStripeCustomerIDFails_ReturnsError() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))
	stripeCustomerID := "cus_abc"

	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)

	// Account lookup fails early — no Stripe mutations
	s.coreClient.EXPECT().GetAccountByStripeCustomerID(anyCtx, stripeCustomerID).
		Return("", "", apierror.NewInternalError(fmt.Errorf("db timeout"), "lookup failed"))

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), stripeCustomerID, "starter")
	s.NotNil(apiErr, "must return error so caller can retry")
	s.Contains(apiErr.InternalMessage, "look up account")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_UpdateAccountSubscriptionFailsAfterCommit_ReturnsError() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))
	stripePlan := testStripePricingPlan()
	stripeCustomerID := "cus_abc"

	s.expectSubscribeLookup(stripeCustomerID, "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, stripeCustomerID, gomock.Any()).Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", gomock.Any()).Return("bc_1", nil)
	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 1000}, nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(1000), "usd", stripeCustomerID, "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_1"},
	}, nil)

	// Post-commit: persist billing IDs fails
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), "acct_1", nil, "", nil, nil, nil, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(apierror.NewInternalError(fmt.Errorf("db timeout"), "persist failed"))

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), stripeCustomerID, "starter")
	s.NotNil(apiErr, "must return error so caller knows billing IDs were not persisted")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_ReserveFails_IntentVoided() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))
	stripePlan := testStripePricingPlan()

	s.expectSubscribeLookup("cus_abc", "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_abc", gomock.Any()).Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", gomock.Any()).Return("bc_1", nil)
	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(nil, fmt.Errorf("reserve failed"))
	s.stripeClient.EXPECT().VoidBillingIntent(anyCtx, "bi_1").Return(nil)

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "starter")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "reserve billing intent")
}

func (s *BillingSvcTestSuite) TestSubscribeToPricingPlan_CreateBillingIntentFails_NoStripeMutation() {
	s.expectIdempotency()
	plan := testPricingPlan("starter", new("spp_123"))
	stripePlan := testStripePricingPlan()

	s.expectSubscribeLookup("cus_abc", "starter", plan, &domain.AccountSubscriptionInfo{})
	s.stripeClient.EXPECT().CreateBillingProfile(anyCtx, "cus_abc", gomock.Any()).Return("bp_1", nil)
	s.stripeClient.EXPECT().CreateBillingCadence(anyCtx, "bp_1", gomock.Any()).Return("bc_1", nil)
	// ProfileCreated phase re-reads plan
	s.pricingPlanRepo.EXPECT().GetPlanByCode(anyCtx, "starter").Return(plan, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_123").Return(stripePlan, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("", fmt.Errorf("intent creation failed"))

	// No Reserve/Commit/Void calls expected — profile+cadence exist but no intent is safe

	apiErr := s.billingSvc.SubscribeToPricingPlan(testIdentityCtx(), "cus_abc", "starter")
	s.NotNil(apiErr)
	s.Contains(apiErr.InternalMessage, "billing intent")
}

// --- SwitchPlan recoverability tests ---

func (s *BillingSvcTestSuite) TestSwitchPlan_UpdateAccountSubscriptionFailsAfterCommit_ReturnsError() {
	s.expectIdempotency()
	accountID := "acct_1"
	planID := "plan_type_pro"
	plan := testPricingPlan("pro", new("spp_pro"))
	stripePlan := testStripePricingPlan()

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID:          new("bc_1"),
		PricingPlanSubscriptionID: new("pps_old"),
	}, nil)
	s.stripeClient.EXPECT().GetPricingPlan(anyCtx, "spp_pro").Return(stripePlan, nil)
	s.accountUsageRepo.EXPECT().CountUsersByAccountID(anyCtx, accountID).Return(3, nil)
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_1", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_1").Return(&domain.BillingIntentReservation{IntentID: "bi_1", NetAmount: 2000}, nil)
	s.accountUsageRepo.EXPECT().GetStripeCustomerIDByAccountID(anyCtx, accountID).Return(new("cus_1"), nil)
	s.stripeClient.EXPECT().CreatePaymentIntent(anyCtx, int64(2000), "usd", "cus_1", "https://app.example.com/dashboard/account?tab=billing").Return("pi_1", nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_1", new("pi_1"), gomock.Any()).Return(&domain.BillingIntentCommitResult{
		PricingPlanSubscriptionIDs: []string{"pps_new"},
	}, nil)

	// Post-commit: persist fails
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "pro", nil, nil, nil, nil, nil, new("pps_new"), new("active"), new("current"),
	).Return(apierror.NewInternalError(fmt.Errorf("db timeout"), "persist failed"))

	_, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), accountID, planID)
	s.NotNil(apiErr, "must return error so caller knows subscription is active in Stripe but plan not updated")
}

func (s *BillingSvcTestSuite) TestSwitchPlan_DowngradeDeactivationSucceeds_PersistFails_ReturnsError() {
	s.expectIdempotency()
	accountID := "acct_1"
	planID := "plan_type_free"
	plan := testPricingPlan("free", nil)

	s.pricingPlanRepo.EXPECT().GetPlanByTypeID(anyCtx, planID).Return(plan, nil)
	s.accountUsageRepo.EXPECT().GetAccountSubscriptionInfo(anyCtx, accountID).Return(&domain.AccountSubscriptionInfo{
		BillingCadenceID:          new("bc_1"),
		PricingPlanSubscriptionID: new("pps_old"),
	}, nil)

	// Deactivation succeeds
	s.stripeClient.EXPECT().CreateBillingIntent(anyCtx, "bc_1", gomock.Any(), gomock.Any()).Return("bi_deact", nil)
	s.stripeClient.EXPECT().ReserveBillingIntent(anyCtx, "bi_deact").Return(&domain.BillingIntentReservation{}, nil)
	s.stripeClient.EXPECT().CommitBillingIntent(anyCtx, "bi_deact", (*string)(nil), gomock.Any()).Return(&domain.BillingIntentCommitResult{}, nil)

	// Post-deactivation: persist plan change fails
	s.coreClient.EXPECT().UpdateAccountSubscription(
		anyCtx, gomock.Any(), accountID, nil, "free", nil, nil, nil, nil, nil, nil, nil, nil,
	).Return(apierror.NewInternalError(fmt.Errorf("db timeout"), "persist failed"))

	_, apiErr := s.billingSvc.SwitchPlan(testIdentityCtx(), accountID, planID)
	s.NotNil(apiErr, "must return error — old sub deactivated but plan not updated; webhook will mark as canceled")
}
