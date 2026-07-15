package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var billingSvcTracer = tracing.GetTracer("billing-service.service")

type BillingSvcConfig struct {
	// Repos (required) is the repository factory for billing persistence.
	Repos domain.RepoFactory

	// StripeClient (required) is the Stripe API client.
	StripeClient domain.StripeClient

	// CoreClient (required) is the core-service client used to resolve accounts.
	CoreClient domain.CoreClient

	// FrontendURL (required) is the dashboard base URL used in Stripe redirect and portal links.
	FrontendURL string

	// NotificationClient (required) sends billing-related notifications.
	NotificationClient domain.NotificationClient

	// IdempotencyMed (required) deduplicates billing operations.
	IdempotencyMed domain.IdempotencyMed
}

type billingSvcImpl struct {
	repos              domain.RepoFactory
	stripeClient       domain.StripeClient
	coreClient         domain.CoreClient
	frontendURL        string
	notificationClient domain.NotificationClient
	idempotencyMed     domain.IdempotencyMed

	agentSpendCacheMu sync.Mutex
	agentSpendCache   map[string]agentSpendCacheEntry

	planCacheMu sync.Mutex
	planCache   map[string]planCacheEntry
}

// agentSpendCacheTTL bounds how stale the displayed/cap-enforced agent spend can be. Short because the underlying Stripe read is a couple of API calls and the value drives both the dashboard figure and cap enforcement.
const agentSpendCacheTTL = 60 * time.Second

// planResolveTTL bounds how long a resolved Stripe pricing plan (rate card, display name, base fee) is reused. Long because a plan changes only when pricing is reconfigured, and re-resolving costs several Stripe calls.
const planResolveTTL = time.Hour

type agentSpendCacheEntry struct {
	cents     int64
	expiresAt time.Time
}

type planCacheEntry struct {
	plan      *domain.StripePricingPlan
	expiresAt time.Time
}

func (c *BillingSvcConfig) validate() error {
	if c.Repos == nil {
		return fmt.Errorf("billing service: repos is required")
	}
	if c.StripeClient == nil {
		return fmt.Errorf("billing service: stripe client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("billing service: core client is required")
	}
	if c.FrontendURL == "" {
		return fmt.Errorf("billing service: frontend URL is required")
	}
	if c.NotificationClient == nil {
		return fmt.Errorf("billing service: notification client is required")
	}
	if c.IdempotencyMed == nil {
		return fmt.Errorf("billing service: idempotency mediator is required")
	}
	return nil
}

func NewBillingSvc(config *BillingSvcConfig) domain.BillingSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &billingSvcImpl{
		repos:              config.Repos,
		stripeClient:       config.StripeClient,
		coreClient:         config.CoreClient,
		frontendURL:        config.FrontendURL,
		notificationClient: config.NotificationClient,
		idempotencyMed:     config.IdempotencyMed,
		agentSpendCache:    make(map[string]agentSpendCacheEntry),
		planCache:          make(map[string]planCacheEntry),
	}
}

func (s *billingSvcImpl) GetPlanByCode(ctx context.Context, planCode string) (*domain.PricingPlan, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_plan_by_code")
	defer span.End()

	repo := s.repos.NewPricingPlanRepo()

	plan, apiErr := repo.GetPlanByCode(ctx, planCode)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limits, apiErr := repo.GetPlanLimitsByTypeID(ctx, plan.TypeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	plan.Limits = limits

	return plan, nil
}

func (s *billingSvcImpl) ListPricingPlans(ctx context.Context, input domain.ListPricingPlansInput) (*domain.ListPricingPlansResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.list_pricing_plans")
	defer span.End()

	repo := s.repos.NewPricingPlanRepo()

	plans, pageInfo, apiErr := repo.ListPricingPlans(ctx, input.Cursor, input.Limit, input.Query)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	for i := range plans {
		limits, apiErr := repo.GetPlanLimitsByTypeID(ctx, plans[i].TypeID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		plans[i].Limits = limits
	}

	return &domain.ListPricingPlansResult{
		Plans:    plans,
		PageInfo: pageInfo,
	}, nil
}

func (s *billingSvcImpl) GetAccountUsage(ctx context.Context, accountID string) (*domain.AccountUsage, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_account_usage")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	limits, apiErr := repo.GetLimitsByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limitMap := make(map[string]*int, len(limits))
	for _, l := range limits {
		limitMap[l.Key] = l.Value
	}

	userCount, apiErr := repo.CountUsersByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	sandboxCount, apiErr := repo.CountSandboxesByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	usage := &domain.AccountUsage{
		Seats:     domain.UsageItem{Current: userCount, Limit: limitMap["seats_maximum"]},
		Sandboxes: domain.UsageItem{Current: sandboxCount, Limit: limitMap["sandboxes_maximum"]},
	}

	// Use v2 status from account record
	if subInfo.ServicingStatus != nil {
		usage.Subscription = &domain.SubscriptionInfoResult{
			ServicingStatus:  *subInfo.ServicingStatus,
			CollectionStatus: ptrutil.Deref(subInfo.CollectionStatus),
		}
	} else if subInfo.SubscriptionStatus != nil {
		// Fallback to v1 status during migration transition
		usage.Subscription = &domain.SubscriptionInfoResult{
			ServicingStatus:  *subInfo.SubscriptionStatus,
			CollectionStatus: "current",
		}
	}

	// Determine billing period start for per-period usage counts
	periodStart := agentSpendPeriodStart(subInfo)

	invoiceCount, apiErr := repo.CountInvoicesByAccountID(ctx, accountID, periodStart)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	batchCount, apiErr := repo.CountBatchesByAccountID(ctx, accountID, periodStart)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	usage.Invoices = domain.UsageItem{Current: invoiceCount, Limit: limitMap["invoices_maximum"]}
	usage.Batches = domain.UsageItem{Current: batchCount, Limit: limitMap["batches_maximum"]}

	spend, plan := s.currentAgentSpend(ctx, accountID, periodStart)
	usage.EstimatedAgentSpendCents = spend
	// Surface the plan name and base fee the customer is actually billed, from the same resolved Stripe plan.
	if plan != nil {
		usage.PlanName = plan.DisplayName
		usage.BaseFeeCents = plan.BaseFeeCents
		usage.BaseFeeInterval = plan.BaseFeeInterval
	}

	return usage, nil
}

// currentAgentSpend returns the marked-up token spend the account will be billed in Stripe for the current period (cached briefly) alongside the resolved Stripe pricing plan (for display). It resolves the account's pricing plan once: the plan supplies both the rate card that prices metered usage and the display name/base fee. Spend is 0 with a nil plan when the account has no Stripe pricing plan; on a transient Stripe/customer failure the last known good spend is preferred over 0.
func (s *billingSvcImpl) currentAgentSpend(ctx context.Context, accountID string, periodStart time.Time) (int64, *domain.StripePricingPlan) {
	s.agentSpendCacheMu.Lock()
	cached, hasCached := s.agentSpendCache[accountID]
	s.agentSpendCacheMu.Unlock()

	repo := s.repos.NewAccountUsageRepo()

	pricingPlanID, planErr := repo.GetAccountStripePricingPlanID(ctx, accountID)
	if planErr != nil {
		return s.agentSpendFallback(cached, hasCached), nil
	}
	if pricingPlanID == nil || *pricingPlanID == "" {
		// Plan has no Stripe pricing plan (e.g. free); genuinely zero agent spend, no plan details.
		return 0, nil
	}

	plan := s.resolvePlan(ctx, *pricingPlanID)

	if hasCached && time.Now().Before(cached.expiresAt) {
		return cached.cents, plan
	}

	if plan == nil || plan.RateCardID == "" {
		// Pricing plan has no token rate card, or Stripe was transiently unavailable resolving it. Prefer the last known good value over flipping the figure to 0.
		return s.agentSpendFallback(cached, hasCached), plan
	}

	customerID, custErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
	if custErr != nil || customerID == nil || *customerID == "" {
		return s.agentSpendFallback(cached, hasCached), plan
	}

	cents, err := s.stripeClient.GetAgentTokenSpendCents(ctx, *customerID, plan.RateCardID, periodStart)
	if err != nil {
		slog.Error("failed to compute agent token spend from Stripe",
			"account_id", accountID, "error", err.Error())
		return s.agentSpendFallback(cached, hasCached), plan
	}

	s.agentSpendCacheMu.Lock()
	s.agentSpendCache[accountID] = agentSpendCacheEntry{cents: cents, expiresAt: time.Now().Add(agentSpendCacheTTL)}
	s.agentSpendCacheMu.Unlock()
	return cents, plan
}

// resolvePlan resolves and caches a Stripe pricing plan's billing details (rate card id for token pricing, display name, and base fee) from its id. Cached per pricing plan for planResolveTTL since a plan changes only when pricing is reconfigured. Returns nil when Stripe is transiently unavailable (not cached, so it retries next call).
func (s *billingSvcImpl) resolvePlan(ctx context.Context, pricingPlanID string) *domain.StripePricingPlan {
	s.planCacheMu.Lock()
	if entry, ok := s.planCache[pricingPlanID]; ok && time.Now().Before(entry.expiresAt) {
		s.planCacheMu.Unlock()
		return entry.plan
	}
	s.planCacheMu.Unlock()

	plan, err := s.stripeClient.GetPricingPlan(ctx, pricingPlanID)
	if err != nil {
		slog.Error("failed to resolve Stripe pricing plan",
			"pricing_plan_id", pricingPlanID, "error", err.Error())
		return nil
	}

	s.planCacheMu.Lock()
	s.planCache[pricingPlanID] = planCacheEntry{plan: plan, expiresAt: time.Now().Add(planResolveTTL)}
	s.planCacheMu.Unlock()
	return plan
}

// agentSpendFallback returns the last known good spend when a fresh read fails, so a transient Stripe error doesn't flip the displayed figure to $0.
func (s *billingSvcImpl) agentSpendFallback(cached agentSpendCacheEntry, hasCached bool) int64 {
	if hasCached {
		return cached.cents
	}
	return 0
}

// GetAgentSpendCents returns the account's marked-up token spend for the current billing period. Shares the cached computation and period boundary used by GetAccountUsage so the cap agent-service enforces matches the dashboard figure.
func (s *billingSvcImpl) GetAgentSpendCents(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_agent_spend_cents")
	defer span.End()

	subInfo, apiErr := s.repos.NewAccountUsageRepo().GetAccountSubscriptionInfo(ctx, accountID)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	cents, _ := s.currentAgentSpend(ctx, accountID, agentSpendPeriodStart(subInfo))
	return cents, nil
}

// GetAgentTokenRates returns the marked-up per-token rates from the account's plan rate card so a caller can price in-flight usage against the cap with the same rates Stripe bills. Empty when the account has no pricing plan or rate card.
func (s *billingSvcImpl) GetAgentTokenRates(ctx context.Context, accountID string) ([]domain.TokenRate, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_agent_token_rates")
	defer span.End()

	pricingPlanID, apiErr := s.repos.NewAccountUsageRepo().GetAccountStripePricingPlanID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if pricingPlanID == nil || *pricingPlanID == "" {
		return nil, nil
	}

	plan := s.resolvePlan(ctx, *pricingPlanID)
	if plan == nil || plan.RateCardID == "" {
		return nil, nil
	}

	rates, err := s.stripeClient.GetRateCardTokenRates(ctx, plan.RateCardID)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "failed to fetch rate card token rates"))
	}
	return rates, nil
}

// agentSpendPeriodStart returns the start of the account's current billing period: one month before the subscription period end, or the calendar month start when there is no active subscription.
func agentSpendPeriodStart(subInfo *domain.AccountSubscriptionInfo) time.Time {
	if subInfo != nil && subInfo.SubscriptionCurrentPeriodEnd != nil {
		return subInfo.SubscriptionCurrentPeriodEnd.AddDate(0, -1, 0)
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (s *billingSvcImpl) CreateBillingPortalSession(ctx context.Context, accountID string) (string, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.create_billing_portal_session")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
	if apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}
	if customerID == nil {
		return "", apierror.NewValidationError("Account has no Stripe customer.")
	}

	returnURL := fmt.Sprintf("%s%s", s.frontendURL, constants.DashboardPathBillingPortal)
	session, err := s.stripeClient.CreateBillingPortalSession(ctx, *customerID, returnURL)
	if err != nil {
		span.RecordError(err)
		return "", apierror.NewInternalError(err, "failed to create billing portal session")
	}

	return session.URL, nil
}

func (s *billingSvcImpl) PreviewPlanChange(ctx context.Context, accountID string, planID string) (*domain.PlanChangePreview, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.preview_plan_change")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	planRepo := s.repos.NewPricingPlanRepo()
	plan, apiErr := planRepo.GetPlanByTypeID(ctx, planID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	if plan.StripePricingPlanID == nil {
		// Free plan — no billing intent needed
		return &domain.PlanChangePreview{
			NetAmount:                  0,
			FormattedNetAmount:         "$0.00",
			MonthlyBillAmount:          0,
			FormattedMonthlyBillAmount: "$0.00",
		}, nil
	}

	if subInfo.BillingCadenceID == nil {
		return nil, apierror.NewValidationError("Account has no billing cadence. Call SetupBillingProfile first.")
	}

	// Determine action type
	var action domain.BillingIntentAction
	stripePlan, err := s.stripeClient.GetPricingPlan(ctx, *plan.StripePricingPlanID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to fetch pricing plan")
	}

	// Build component configurations with current seat count for accurate preview
	userCount, _ := repo.CountUsersByAccountID(ctx, accountID)
	var compConfigs []domain.ComponentConfiguration
	if stripePlan.LicenseFeeComponentID != "" {
		qty := userCount
		if plan.SeatMinimum != nil && *plan.SeatMinimum > qty {
			qty = *plan.SeatMinimum
		}
		if qty < 1 {
			qty = 1
		}
		compConfigs = append(compConfigs, domain.ComponentConfiguration{
			PricingPlanComponentID: stripePlan.LicenseFeeComponentID,
			Quantity:               qty,
		})
	}

	if subInfo.PricingPlanSubscriptionID != nil {
		action = domain.BillingIntentAction{
			Type:                    "modify",
			PricingPlanID:           *plan.StripePricingPlanID,
			PricingPlanVersion:      stripePlan.LiveVersion,
			SubscriptionID:          *subInfo.PricingPlanSubscriptionID,
			ComponentConfigurations: compConfigs,
		}
	} else {
		action = domain.BillingIntentAction{
			Type:                    "subscribe",
			PricingPlanID:           *plan.StripePricingPlanID,
			PricingPlanVersion:      stripePlan.LiveVersion,
			ComponentConfigurations: compConfigs,
		}
	}

	// Create intent, reserve, extract preview, void
	intentID, err := s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, "")

	// If a stale preview intent blocks us, void it (best-effort) and retry once. The void may fail if the intent is already committed/voided by another process, but the retry may still succeed if the conflict has cleared.
	var conflict *domain.ErrBillingIntentConflict
	if errors.As(err, &conflict) {
		slog.WarnContext(ctx, "voiding conflicting preview billing intent",
			"conflicting_intent_id", conflict.ConflictingIntentID,
			"account_id", accountID,
		)
		if voidErr := s.stripeClient.VoidBillingIntent(ctx, conflict.ConflictingIntentID); voidErr != nil {
			slog.WarnContext(ctx, "failed to void conflicting billing intent, will retry create anyway",
				"intent_id", conflict.ConflictingIntentID, "error", voidErr)
		}
		intentID, err = s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, "")
	}

	if err != nil {
		// Billing intent creation failed (e.g., subscription locked by another intent).
		// Fall back to local proration estimate so the user still gets a preview.
		slog.WarnContext(ctx, "billing intent unavailable, using local proration estimate",
			"account_id", accountID, "error", err)
		return s.estimateProrationLocally(ctx, repo, planRepo, plan, accountID, userCount)
	}

	reservation, err := s.stripeClient.ReserveBillingIntent(ctx, intentID)
	if err != nil {
		_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to reserve billing intent for preview")
	}

	// Void the intent — we only needed the preview
	if voidErr := s.stripeClient.VoidBillingIntent(ctx, intentID); voidErr != nil {
		slog.WarnContext(ctx, "failed to void preview billing intent", "intent_id", intentID, "error", voidErr)
	}

	lineItems := make([]domain.PlanChangePreviewLineItem, len(reservation.LineItems))
	for i, li := range reservation.LineItems {
		lineItems[i] = domain.PlanChangePreviewLineItem(li)
	}

	// Estimate monthly bill from plan pricing
	monthlyBill := estimateMonthlyBill(plan, userCount)

	return &domain.PlanChangePreview{
		NetAmount:                  reservation.NetAmount,
		FormattedNetAmount:         formatAmount(reservation.NetAmount),
		MonthlyBillAmount:          monthlyBill,
		FormattedMonthlyBillAmount: formatAmount(monthlyBill),
		LineItems:                  lineItems,
	}, nil
}

// estimateProrationLocally calculates a proration preview without Stripe billing intents. Used as a fallback when the subscription is locked by another intent.
func (s *billingSvcImpl) estimateProrationLocally(
	ctx context.Context,
	repo domain.AccountUsageRepo,
	planRepo domain.PricingPlanRepo,
	newPlan *domain.PricingPlan,
	accountID string,
	userCount int,
) (*domain.PlanChangePreview, *apierror.APIError) {
	monthlyBill := estimateMonthlyBill(newPlan, userCount)

	// Try to compute a proration credit from the current plan.
	_, currentPlanCode, err := repo.GetAccountNameAndPlanCode(ctx, accountID)
	if err != nil {
		// Can't determine current plan — return estimate with no proration.
		return &domain.PlanChangePreview{
			NetAmount:                  monthlyBill,
			FormattedNetAmount:         formatAmount(monthlyBill),
			MonthlyBillAmount:          monthlyBill,
			FormattedMonthlyBillAmount: formatAmount(monthlyBill),
			IsEstimate:                 true,
		}, nil
	}

	currentPlan, apiErr := planRepo.GetPlanByCode(ctx, currentPlanCode)
	if apiErr != nil {
		return &domain.PlanChangePreview{
			NetAmount:                  monthlyBill,
			FormattedNetAmount:         formatAmount(monthlyBill),
			MonthlyBillAmount:          monthlyBill,
			FormattedMonthlyBillAmount: formatAmount(monthlyBill),
			IsEstimate:                 true,
		}, nil
	}

	currentMonthly := estimateMonthlyBill(currentPlan, userCount)
	netAmount := monthlyBill - currentMonthly

	var lineItems []domain.PlanChangePreviewLineItem
	if currentMonthly > 0 {
		lineItems = append(lineItems, domain.PlanChangePreviewLineItem{
			Description: fmt.Sprintf("Unused time on %s plan", currentPlan.Name),
			Amount:      -currentMonthly,
		})
	}
	lineItems = append(lineItems, domain.PlanChangePreviewLineItem{
		Description: fmt.Sprintf("Remaining time on %s plan", newPlan.Name),
		Amount:      monthlyBill,
	})

	return &domain.PlanChangePreview{
		NetAmount:                  netAmount,
		FormattedNetAmount:         formatAmount(netAmount),
		MonthlyBillAmount:          monthlyBill,
		FormattedMonthlyBillAmount: formatAmount(monthlyBill),
		LineItems:                  lineItems,
		IsEstimate:                 true,
	}, nil
}

func estimateMonthlyBill(plan *domain.PricingPlan, userCount int) int64 {
	var unitAmount int64
	if plan.PricePerMonth != nil && *plan.PricePerMonth > 0 {
		unitAmount = int64(*plan.PricePerMonth * 100)
	} else {
		unitAmount = int64(plan.PricePerSeat * 100)
	}

	quantity := int64(userCount)
	if plan.SeatMinimum != nil && int64(*plan.SeatMinimum) > quantity {
		quantity = int64(*plan.SeatMinimum)
	}
	if quantity < 1 {
		quantity = 1
	}

	return unitAmount * quantity
}

func (s *billingSvcImpl) RequestEnterpriseUpgrade(ctx context.Context, input domain.RequestEnterpriseUpgradeInput) (*domain.RequestEnterpriseUpgradeResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.request_enterprise_upgrade")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	accountName, planCode, apiErr := repo.GetAccountNameAndPlanCode(ctx, input.AccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	planRepo := s.repos.NewPricingPlanRepo()
	plan, apiErr := planRepo.GetPlanByCode(ctx, planCode)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	requesterEmail, _, apiErr := repo.GetUserEmailByID(ctx, input.ActorID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.notificationClient.SendEnterpriseRequest(ctx, input.AccountID, accountName, plan.Name, input.ActorName, requesterEmail)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.RequestEnterpriseUpgradeResult{Success: true}, nil
}

func (s *billingSvcImpl) EnsureBillingCustomer(ctx context.Context, accountID string) (*domain.EnsureBillingCustomerResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.ensure_billing_customer")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	iKey, apiErr := s.idempotencyMed.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := domain.RecoveryPoint(iKey.RecoveryPoint)

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.EnsureBillingCustomerResult](ctx, iKey.ResponseCode, iKey.ResponseBody)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			if cached.Error != nil {
				return nil, cached.Error
			}
			return cached.Data, nil

		case domain.RecoveryPointStarted:
			repo := s.repos.NewAccountUsageRepo()

			customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}
			if customerID != nil {
				result := &domain.EnsureBillingCustomerResult{
					StripeCustomerID: *customerID,
					Created:          false,
				}
				if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, result); cacheErr != nil {
					return nil, cacheErr
				}
				return result, nil
			}

			accountName, _, apiErr := repo.GetAccountNameAndPlanCode(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}

			adminEmail, apiErr := repo.GetAdminEmailByAccountID(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}
			if adminEmail == "" {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, apierror.NewValidationError("No admin user found for account."))
			}

			stripeIdemKey := fmt.Sprintf("ensure_customer_%s", accountID)
			metadata := map[string]string{
				"accountID":   accountID,
				"accountName": accountName,
			}

			cust, err := s.stripeClient.CreateCustomer(ctx, adminEmail, accountName, stripeIdemKey, metadata)
			if err != nil {
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to create Stripe customer")
			}

			if apiErr := repo.UpdateStripeCustomerIDByAccountID(ctx, cust.ID, accountID); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}

			result := &domain.EnsureBillingCustomerResult{
				StripeCustomerID: cust.ID,
				Created:          true,
			}
			if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, result); cacheErr != nil {
				return nil, cacheErr
			}
			return result, nil

		default:
			return nil, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid recovery point: %s", recoveryPoint)))
		}
	}
}

func (s *billingSvcImpl) CreateRegistrationCustomer(ctx context.Context, email, name, idempotencyKey string, metadata map[string]string) (*domain.EnsureBillingCustomerResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.create_registration_customer")
	defer span.End()

	cust, err := s.stripeClient.CreateCustomer(ctx, email, name, idempotencyKey, metadata)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to create Stripe customer")
	}

	return &domain.EnsureBillingCustomerResult{
		StripeCustomerID: cust.ID,
		Created:          true,
	}, nil
}

// setupBillingProfileState holds intermediate state for SetupBillingProfile recovery.
type setupBillingProfileState struct {
	ProfileID string `json:"profileID"`
	CadenceID string `json:"cadenceID"`
}

func (s *billingSvcImpl) SetupBillingProfile(ctx context.Context, accountID string) (*domain.BillingProfileResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.setup_billing_profile")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	iKey, apiErr := s.idempotencyMed.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := domain.RecoveryPoint(iKey.RecoveryPoint)
	var state setupBillingProfileState

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.BillingProfileResult](ctx, iKey.ResponseCode, iKey.ResponseBody)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			if cached.Error != nil {
				return nil, cached.Error
			}
			return cached.Data, nil

		case domain.RecoveryPointStarted:
			repo := s.repos.NewAccountUsageRepo()

			subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}
			if subInfo.BillingProfileID != nil && subInfo.BillingCadenceID != nil {
				result := &domain.BillingProfileResult{
					ProfileID: *subInfo.BillingProfileID,
					CadenceID: *subInfo.BillingCadenceID,
				}
				if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, result); cacheErr != nil {
					return nil, cacheErr
				}
				return result, nil
			}

			customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}
			if customerID == nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, apierror.NewValidationError("Account has no Stripe customer. Call EnsureBillingCustomer first."))
			}

			stripeIdemKey := fmt.Sprintf("billing_profile_%s", accountID)

			profileID, err := s.stripeClient.CreateBillingProfile(ctx, *customerID, stripeIdemKey+"_profile")
			if err != nil {
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to create billing profile")
			}

			cadenceID, err := s.stripeClient.CreateBillingCadence(ctx, profileID, stripeIdemKey+"_cadence")
			if err != nil {
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to create billing cadence")
			}

			state = setupBillingProfileState{ProfileID: profileID, CadenceID: cadenceID}
			stateBody, jsonErr := json.Marshal(state)
			if jsonErr != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal intermediate state."))
			}
			if setErr := s.repos.NewIdempotencyKeyRepo().SetResponse(ctx, iKey.TypeID, 200, stateBody, domain.RecoveryPointProfileCreated); setErr != nil {
				return nil, setErr
			}
			recoveryPoint = domain.RecoveryPointProfileCreated
			continue

		case domain.RecoveryPointProfileCreated:
			if state.ProfileID == "" {
				cached, err := idempotency.UnmarshalCachedResponse[setupBillingProfileState](ctx, iKey.ResponseCode, iKey.ResponseBody)
				if err != nil {
					return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				state = *cached.Data
			}

			updateKey := fmt.Sprintf("setup_profile_%s", accountID)
			if apiErr := s.coreClient.UpdateAccountSubscription(ctx, updateKey, accountID, nil, "", nil, nil, nil, &state.ProfileID, &state.CadenceID, nil, nil, nil); apiErr != nil {
				return nil, apiErr
			}

			result := &domain.BillingProfileResult{
				ProfileID: state.ProfileID,
				CadenceID: state.CadenceID,
			}
			if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, result); cacheErr != nil {
				return nil, cacheErr
			}
			return result, nil

		default:
			return nil, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid recovery point: %s", recoveryPoint)))
		}
	}
}

// switchPlanState holds intermediate state for SwitchPlan recovery.
type switchPlanState struct {
	PricingPlanSubID *string `json:"pricingPlanSubID,omitempty"`
	IntentID         *string `json:"intentID,omitempty"`
	PlanCode         string  `json:"planCode"`
}

func (s *billingSvcImpl) SwitchPlan(ctx context.Context, accountID string, planID string) (*domain.SwitchPlanResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.switch_plan")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	iKey, apiErr := s.idempotencyMed.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	recoveryPoint := domain.RecoveryPoint(iKey.RecoveryPoint)
	var state switchPlanState

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[domain.SwitchPlanResult](ctx, iKey.ResponseCode, iKey.ResponseBody)
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			if cached.Error != nil {
				return nil, cached.Error
			}
			return cached.Data, nil

		case domain.RecoveryPointStarted:
			repo := s.repos.NewAccountUsageRepo()

			planRepo := s.repos.NewPricingPlanRepo()
			plan, apiErr := planRepo.GetPlanByTypeID(ctx, planID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}

			subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
			if apiErr != nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}

			planCode := plan.PlanTypeCode

			// Paid → Free: deactivate existing pricing plan subscription
			if planCode == string(constants.PlanCodeFree) {
				if subInfo.PricingPlanSubscriptionID != nil && subInfo.BillingCadenceID != nil {
					action := domain.BillingIntentAction{
						Type:           "deactivate",
						SubscriptionID: *subInfo.PricingPlanSubscriptionID,
					}
					intentID, err := s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, fmt.Sprintf("switch_%s_%s", accountID, planID))
					var conflict *domain.ErrBillingIntentConflict
					if errors.As(err, &conflict) {
						slog.WarnContext(ctx, "voiding conflicting billing intent before deactivation",
							"conflicting_intent_id", conflict.ConflictingIntentID,
							"account_id", accountID,
						)
						if voidErr := s.stripeClient.VoidBillingIntent(ctx, conflict.ConflictingIntentID); voidErr != nil {
							slog.WarnContext(ctx, "failed to void conflicting billing intent, will retry create anyway",
								"intent_id", conflict.ConflictingIntentID, "error", voidErr)
						}
						intentID, err = s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, fmt.Sprintf("switch_%s_%s_%d", accountID, planID, time.Now().UnixMilli()))
					}
					if err != nil && strings.Contains(err.Error(), "invalid_pricing_plan_subscription_status_for_cancel") {
						// Subscription already canceled on Stripe — skip deactivation.
						slog.WarnContext(ctx, "subscription already canceled on Stripe, skipping deactivation",
							"account_id", accountID,
							"subscription_id", *subInfo.PricingPlanSubscriptionID,
						)
					} else if err != nil {
						span.RecordError(err)
						return nil, apierror.NewInternalError(err, "failed to create deactivate billing intent")
					} else {
						if _, err := s.stripeClient.ReserveBillingIntent(ctx, intentID); err != nil {
							span.RecordError(err)
							return nil, apierror.NewInternalError(err, "failed to reserve deactivate billing intent")
						}
						if _, err := s.stripeClient.CommitBillingIntent(ctx, intentID, nil, *subInfo.BillingCadenceID); err != nil {
							span.RecordError(err)
							return nil, apierror.NewInternalError(err, "failed to commit deactivate billing intent")
						}
					}
				}

				state = switchPlanState{PlanCode: planCode}
				stateBody, jsonErr := json.Marshal(state)
				if jsonErr != nil {
					return nil, tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal intermediate state."))
				}
				if setErr := s.repos.NewIdempotencyKeyRepo().SetResponse(ctx, iKey.TypeID, 200, stateBody, domain.RecoveryPointIntentCommitted); setErr != nil {
					return nil, setErr
				}
				recoveryPoint = domain.RecoveryPointIntentCommitted
				continue
			}

			// Free → Paid or Paid → Paid: need billing profile + cadence
			if subInfo.BillingCadenceID == nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, apierror.NewValidationError("Account has no billing cadence. Call SetupBillingProfile first."))
			}
			if plan.StripePricingPlanID == nil {
				return nil, s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, apierror.NewValidationError("Target plan has no pricing plan ID configured."))
			}

			stripePlan, err := s.stripeClient.GetPricingPlan(ctx, *plan.StripePricingPlanID)
			if err != nil {
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to fetch pricing plan")
			}

			userCount, _ := repo.CountUsersByAccountID(ctx, accountID)
			var compConfigs []domain.ComponentConfiguration
			if stripePlan.LicenseFeeComponentID != "" {
				qty := userCount
				if plan.SeatMinimum != nil && *plan.SeatMinimum > qty {
					qty = *plan.SeatMinimum
				}
				if qty < 1 {
					qty = 1
				}
				compConfigs = append(compConfigs, domain.ComponentConfiguration{
					PricingPlanComponentID: stripePlan.LicenseFeeComponentID,
					Quantity:               qty,
				})
			}

			var action domain.BillingIntentAction
			if subInfo.PricingPlanSubscriptionID != nil {
				action = domain.BillingIntentAction{
					Type:                    "modify",
					PricingPlanID:           *plan.StripePricingPlanID,
					PricingPlanVersion:      stripePlan.LiveVersion,
					SubscriptionID:          *subInfo.PricingPlanSubscriptionID,
					ComponentConfigurations: compConfigs,
				}
			} else {
				action = domain.BillingIntentAction{
					Type:                    "subscribe",
					PricingPlanID:           *plan.StripePricingPlanID,
					PricingPlanVersion:      stripePlan.LiveVersion,
					ComponentConfigurations: compConfigs,
				}
			}

			intentKey := fmt.Sprintf("switch_%s_%s_%d", accountID, planID, time.Now().UnixMilli())
			intentID, err := s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, intentKey+"_intent")
			var conflict *domain.ErrBillingIntentConflict
			if errors.As(err, &conflict) {
				slog.WarnContext(ctx, "voiding conflicting billing intent before plan switch",
					"conflicting_intent_id", conflict.ConflictingIntentID,
					"account_id", accountID,
				)
				if voidErr := s.stripeClient.VoidBillingIntent(ctx, conflict.ConflictingIntentID); voidErr != nil {
					slog.WarnContext(ctx, "failed to void conflicting billing intent, will retry create anyway",
						"intent_id", conflict.ConflictingIntentID, "error", voidErr)
				}
				retryKey := fmt.Sprintf("switch_%s_%s_%d", accountID, planID, time.Now().UnixMilli())
				intentID, err = s.stripeClient.CreateBillingIntent(ctx, *subInfo.BillingCadenceID, []domain.BillingIntentAction{action}, retryKey+"_intent")
			}
			if err != nil {
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to create billing intent")
			}

			reservation, err := s.stripeClient.ReserveBillingIntent(ctx, intentID)
			if err != nil {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to reserve billing intent")
			}

			var paymentIntentID *string
			if reservation.NetAmount > 0 {
				customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
				if apiErr != nil {
					_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
					return nil, tracing.Trace(span, apiErr)
				}
				if customerID == nil {
					_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
					return nil, apierror.NewValidationError("Account has no Stripe customer.")
				}
				returnURL := fmt.Sprintf("%s%s", s.frontendURL, constants.DashboardPathBillingPortal)
				piID, piErr := s.stripeClient.CreatePaymentIntent(ctx, reservation.NetAmount, "usd", *customerID, returnURL)
				if piErr != nil {
					_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
					span.RecordError(piErr)
					return nil, apierror.NewInternalError(piErr, "failed to create payment intent")
				}
				paymentIntentID = &piID
			}

			commitResult, err := s.stripeClient.CommitBillingIntent(ctx, intentID, paymentIntentID, *subInfo.BillingCadenceID)
			if err != nil {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				span.RecordError(err)
				return nil, apierror.NewInternalError(err, "failed to commit billing intent")
			}

			if commitResult == nil || len(commitResult.PricingPlanSubscriptionIDs) == 0 {
				return nil, apierror.NewInternalError(nil, "commit succeeded but no subscription ID returned")
			}
			pricingPlanSubID := &commitResult.PricingPlanSubscriptionIDs[0]

			state = switchPlanState{PricingPlanSubID: pricingPlanSubID, IntentID: &intentID, PlanCode: planCode}
			stateBody, jsonErr := json.Marshal(state)
			if jsonErr != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal intermediate state."))
			}
			if setErr := s.repos.NewIdempotencyKeyRepo().SetResponse(ctx, iKey.TypeID, 200, stateBody, domain.RecoveryPointIntentCommitted); setErr != nil {
				return nil, setErr
			}
			recoveryPoint = domain.RecoveryPointIntentCommitted
			continue

		case domain.RecoveryPointIntentCommitted:
			if state.PlanCode == "" {
				cached, err := idempotency.UnmarshalCachedResponse[switchPlanState](ctx, iKey.ResponseCode, iKey.ResponseBody)
				if err != nil {
					return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				state = *cached.Data
			}

			if state.PlanCode == string(constants.PlanCodeFree) {
				coreIdemKey := fmt.Sprintf("switch_plan_%s_%s", accountID, planID)
				if switchErr := s.coreClient.UpdateAccountSubscription(ctx, coreIdemKey, accountID, nil, state.PlanCode, nil, nil, nil, nil, nil, nil, nil, nil); switchErr != nil {
					return nil, switchErr
				}
			} else {
				servicingStatus := "active"
				collectionStatus := "current"
				coreIdemKey := fmt.Sprintf("switch_plan_%s_%s", accountID, planID)
				if switchErr := s.coreClient.UpdateAccountSubscription(ctx, coreIdemKey, accountID, nil, state.PlanCode, nil, nil, nil, nil, nil, state.PricingPlanSubID, &servicingStatus, &collectionStatus); switchErr != nil {
					return nil, switchErr
				}
			}

			result := &domain.SwitchPlanResult{
				Success:  true,
				IntentID: state.IntentID,
			}
			if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, result); cacheErr != nil {
				return nil, cacheErr
			}
			return result, nil

		default:
			return nil, tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid recovery point: %s", recoveryPoint)))
		}
	}
}

// subscribeToPricingPlanState holds intermediate state for SubscribeToPricingPlan recovery.
type subscribeToPricingPlanState struct {
	AccountID        string  `json:"accountID"`
	ProfileID        string  `json:"profileID"`
	CadenceID        string  `json:"cadenceID"`
	PricingPlanSubID *string `json:"pricingPlanSubID,omitempty"`
}

func (s *billingSvcImpl) SubscribeToPricingPlan(ctx context.Context, stripeCustomerID, planCode string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.subscribe_to_pricing_plan")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return tracing.Trace(span, apierror.NewInvariantViolationError("Identity not found in context."))
	}
	iKey, apiErr := s.idempotencyMed.UpsertIdempotencyKey(ctx, identity)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	recoveryPoint := domain.RecoveryPoint(iKey.RecoveryPoint)
	var state subscribeToPricingPlanState

	for {
		switch recoveryPoint {
		case domain.RecoveryPointFinished:
			cached, err := idempotency.UnmarshalCachedResponse[struct{}](ctx, iKey.ResponseCode, iKey.ResponseBody)
			if err != nil {
				return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
			}
			if cached.Error != nil {
				return cached.Error
			}
			return nil

		case domain.RecoveryPointStarted:
			repo := s.repos.NewPricingPlanRepo()
			usageRepo := s.repos.NewAccountUsageRepo()

			plan, apiErr := repo.GetPlanByCode(ctx, planCode)
			if apiErr != nil {
				return s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}

			if plan.StripePricingPlanID == nil {
				if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, struct{}{}); cacheErr != nil {
					return cacheErr
				}
				return nil
			}

			accountID, _, lookupErr := s.coreClient.GetAccountByStripeCustomerID(ctx, stripeCustomerID)
			if lookupErr != nil {
				return apierror.NewInternalError(lookupErr.Unwrap(), "failed to look up account for billing subscription")
			}

			subInfo, apiErr := usageRepo.GetAccountSubscriptionInfo(ctx, accountID)
			if apiErr != nil {
				return s.idempotencyMed.CacheErrorResponse(ctx, iKey.TypeID, tracing.Trace(span, apiErr))
			}
			if subInfo.PricingPlanSubscriptionID != nil {
				if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, struct{}{}); cacheErr != nil {
					return cacheErr
				}
				return nil
			}

			var profileID, cadenceID string
			if subInfo.BillingProfileID != nil && subInfo.BillingCadenceID != nil {
				profileID = *subInfo.BillingProfileID
				cadenceID = *subInfo.BillingCadenceID
			} else {
				stripeIdemKey := fmt.Sprintf("subscribe_%s", stripeCustomerID)

				var err error
				profileID, err = s.stripeClient.CreateBillingProfile(ctx, stripeCustomerID, stripeIdemKey+"_profile")
				if err != nil {
					span.RecordError(err)
					return apierror.NewInternalError(err, "failed to create billing profile")
				}

				cadenceID, err = s.stripeClient.CreateBillingCadence(ctx, profileID, stripeIdemKey+"_cadence")
				if err != nil {
					span.RecordError(err)
					return apierror.NewInternalError(err, "failed to create billing cadence")
				}
			}

			state = subscribeToPricingPlanState{AccountID: accountID, ProfileID: profileID, CadenceID: cadenceID}
			stateBody, jsonErr := json.Marshal(state)
			if jsonErr != nil {
				return tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal intermediate state."))
			}
			if setErr := s.repos.NewIdempotencyKeyRepo().SetResponse(ctx, iKey.TypeID, 200, stateBody, domain.RecoveryPointProfileCreated); setErr != nil {
				return setErr
			}
			recoveryPoint = domain.RecoveryPointProfileCreated
			continue

		case domain.RecoveryPointProfileCreated:
			if state.AccountID == "" {
				cached, err := idempotency.UnmarshalCachedResponse[subscribeToPricingPlanState](ctx, iKey.ResponseCode, iKey.ResponseBody)
				if err != nil {
					return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				state = *cached.Data
			}

			repo := s.repos.NewPricingPlanRepo()
			plan, apiErr := repo.GetPlanByCode(ctx, planCode)
			if apiErr != nil {
				return tracing.Trace(span, apiErr)
			}

			stripePlan, err := s.stripeClient.GetPricingPlan(ctx, *plan.StripePricingPlanID)
			if err != nil {
				span.RecordError(err)
				return apierror.NewInternalError(err, "failed to fetch pricing plan from Stripe")
			}

			var compConfigs []domain.ComponentConfiguration
			if stripePlan.LicenseFeeComponentID != "" {
				qty := 1
				if plan.SeatMinimum != nil && *plan.SeatMinimum > qty {
					qty = *plan.SeatMinimum
				}
				compConfigs = append(compConfigs, domain.ComponentConfiguration{
					PricingPlanComponentID: stripePlan.LicenseFeeComponentID,
					Quantity:               qty,
				})
			}

			action := domain.BillingIntentAction{
				Type:                    "subscribe",
				PricingPlanID:           *plan.StripePricingPlanID,
				PricingPlanVersion:      stripePlan.LiveVersion,
				ComponentConfigurations: compConfigs,
			}

			stripeIdemKey := fmt.Sprintf("subscribe_%s", stripeCustomerID)
			intentID, err := s.stripeClient.CreateBillingIntent(ctx, state.CadenceID, []domain.BillingIntentAction{action}, stripeIdemKey+"_intent")
			if err != nil {
				span.RecordError(err)
				return apierror.NewInternalError(err, "failed to create billing intent")
			}

			reservation, err := s.stripeClient.ReserveBillingIntent(ctx, intentID)
			if err != nil {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				span.RecordError(err)
				return apierror.NewInternalError(err, "failed to reserve billing intent")
			}

			if reservation.NetAmount <= 0 {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				return apierror.NewInternalError(nil, fmt.Sprintf("expected positive net amount for paid plan %s, got %d", planCode, reservation.NetAmount))
			}

			returnURL := fmt.Sprintf("%s%s", s.frontendURL, constants.DashboardPathBillingPortal)
			piID, piErr := s.stripeClient.CreatePaymentIntent(ctx, reservation.NetAmount, "usd", stripeCustomerID, returnURL)
			if piErr != nil {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				span.RecordError(piErr)
				return apierror.NewInternalError(piErr, "failed to create payment intent")
			}
			paymentIntentID := &piID

			commitResult, err := s.stripeClient.CommitBillingIntent(ctx, intentID, paymentIntentID, state.CadenceID)
			if err != nil {
				_ = s.stripeClient.VoidBillingIntent(ctx, intentID)
				span.RecordError(err)
				return apierror.NewInternalError(err, "failed to commit billing intent")
			}

			if commitResult == nil || len(commitResult.PricingPlanSubscriptionIDs) == 0 {
				return apierror.NewInternalError(nil, "commit succeeded but no subscription ID returned")
			}
			state.PricingPlanSubID = &commitResult.PricingPlanSubscriptionIDs[0]

			stateBody, jsonErr := json.Marshal(state)
			if jsonErr != nil {
				return tracing.Trace(span, apierror.NewInternalError(jsonErr, "Failed to marshal intermediate state."))
			}
			if setErr := s.repos.NewIdempotencyKeyRepo().SetResponse(ctx, iKey.TypeID, 200, stateBody, domain.RecoveryPointIntentCommitted); setErr != nil {
				return setErr
			}
			recoveryPoint = domain.RecoveryPointIntentCommitted
			continue

		case domain.RecoveryPointIntentCommitted:
			if state.AccountID == "" {
				cached, err := idempotency.UnmarshalCachedResponse[subscribeToPricingPlanState](ctx, iKey.ResponseCode, iKey.ResponseBody)
				if err != nil {
					return tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling recovery point response."))
				}
				state = *cached.Data
			}

			servicingStatus := "active"
			collectionStatus := "current"
			persistKey := fmt.Sprintf("subscribe_persist_%s", stripeCustomerID)
			if persistErr := s.coreClient.UpdateAccountSubscription(ctx, persistKey, state.AccountID, nil, "", nil, nil, nil, &state.ProfileID, &state.CadenceID, state.PricingPlanSubID, &servicingStatus, &collectionStatus); persistErr != nil {
				slog.WarnContext(ctx, "failed to persist v2 billing IDs after subscribe",
					"account_id", state.AccountID, "error", persistErr.PublicMessage)
				return persistErr
			}

			if cacheErr := s.idempotencyMed.CacheSuccessResponse(ctx, iKey.TypeID, struct{}{}); cacheErr != nil {
				return cacheErr
			}
			return nil

		default:
			return tracing.Trace(span, apierror.NewInvariantViolationError(fmt.Sprintf("Invalid recovery point: %s", recoveryPoint)))
		}
	}
}

func (s *billingSvcImpl) CreateSetupIntent(ctx context.Context, customerID, idempotencyKey string) (*domain.SetupIntentResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.create_setup_intent")
	defer span.End()

	si, err := s.stripeClient.CreateSetupIntent(ctx, customerID, idempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to create Setup Intent")
	}

	return &domain.SetupIntentResult{
		SetupIntentID: si.ID,
		ClientSecret:  si.ClientSecret,
		Status:        si.Status,
	}, nil
}

func (s *billingSvcImpl) GetSetupIntentStatus(ctx context.Context, setupIntentID string) (*domain.SetupIntentResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_setup_intent_status")
	defer span.End()

	si, err := s.stripeClient.GetSetupIntent(ctx, setupIntentID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get Setup Intent status")
	}

	return &domain.SetupIntentResult{
		SetupIntentID:   si.ID,
		ClientSecret:    si.ClientSecret,
		Status:          si.Status,
		PaymentMethodID: si.PaymentMethodID,
	}, nil
}

func (s *billingSvcImpl) ValidateStripePricingPlan(ctx context.Context, planCode string) *apierror.APIError {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.validate_stripe_pricing_plan")
	defer span.End()

	repo := s.repos.NewPricingPlanRepo()
	plan, apiErr := repo.GetPlanByCode(ctx, planCode)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	if plan.StripePricingPlanID == nil {
		return nil
	}

	_, err := s.stripeClient.GetPricingPlan(ctx, *plan.StripePricingPlanID)
	if err != nil {
		span.RecordError(err)
		return apierror.NewInternalError(err, "Stripe pricing plan is not accessible for plan code: "+planCode)
	}

	return nil
}

func formatAmount(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	dollars := cents / 100
	remainder := cents % 100
	if negative {
		return fmt.Sprintf("-$%d.%02d", dollars, remainder)
	}
	return fmt.Sprintf("$%d.%02d", dollars, remainder)
}
