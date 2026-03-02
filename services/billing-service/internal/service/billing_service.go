package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
	"github.com/stripe/stripe-go/v84"
)

var billingSvcTracer = tracing.GetTracer("billing-service.service")

type NotificationClient interface {
	SendEnterpriseRequest(ctx context.Context, accountID, accountName, currentPlanName, requesterName, requesterEmail string) *apierror.APIError
}

type BillingSvcConfig struct {
	Repos              domain.RepoFactory
	StripeClient       domain.StripeClient
	CoreClient         domain.CoreClient
	FrontendURL        string
	NotificationClient NotificationClient
}

type billingSvcImpl struct {
	repos              domain.RepoFactory
	stripeClient       domain.StripeClient
	coreClient         domain.CoreClient
	frontendURL        string
	notificationClient NotificationClient
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

	plans, pageInfo, apiErr := repo.ListPricingPlans(ctx, input.Cursor, input.Limit)
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

	// Fetch subscription info to determine the billing period start for
	// per-period usage counts (invoices & batches).
	subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	usage := &domain.AccountUsage{
		Seats:     domain.UsageItem{Current: userCount, Limit: limitMap["seats_maximum"]},
		Sandboxes: domain.UsageItem{Current: sandboxCount, Limit: limitMap["sandboxes_maximum"]},
	}

	var periodStart time.Time
	if subInfo.StripeSubscriptionID != nil {
		stripeSub, err := s.stripeClient.GetSubscription(ctx, *subInfo.StripeSubscriptionID)
		if err != nil {
			span.RecordError(err)
			return nil, apierror.NewInternalError(err, "failed to retrieve Stripe subscription")
		}

		periodStart = stripeSub.CurrentPeriodStart

		usage.Subscription = &domain.SubscriptionInfoResult{
			Status:            stripeSub.Status,
			CurrentPeriodEnd:  &stripeSub.CurrentPeriodEnd,
			TrialEnd:          stripeSub.TrialEnd,
			CancelAtPeriodEnd: stripeSub.CancelAtPeriodEnd,
			CancelAt:          stripeSub.CancelAt,
		}
	} else if subInfo.SubscriptionStatus != nil {
		usage.Subscription = &domain.SubscriptionInfoResult{
			Status:           *subInfo.SubscriptionStatus,
			CurrentPeriodEnd: subInfo.SubscriptionCurrentPeriodEnd,
		}
		// Enterprise / DB-only subscription: estimate period start from period end.
		if subInfo.SubscriptionCurrentPeriodEnd != nil {
			periodStart = subInfo.SubscriptionCurrentPeriodEnd.AddDate(0, -1, 0)
		} else {
			now := time.Now().UTC()
			periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		}
	} else {
		// Free plan with no subscription: use start of the current calendar month.
		now := time.Now().UTC()
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

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

	return usage, nil
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

func (s *billingSvcImpl) GetProrationPreview(ctx context.Context, accountID string, planID string) (*domain.ProrationPreview, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.get_proration_preview")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	// Get Stripe customer ID
	customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if customerID == nil {
		return nil, apierror.NewValidationError("Account has no Stripe customer.")
	}

	// Get the target plan by type ID
	planRepo := s.repos.NewPricingPlanRepo()
	plan, apiErr := planRepo.GetPlanByTypeID(ctx, planID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limits, apiErr := planRepo.GetPlanLimitsByTypeID(ctx, plan.TypeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	plan.Limits = limits

	// Get current user count for seat quantity
	userCount, apiErr := repo.CountUsersByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Calculate unit amount in cents
	var unitAmount int64
	if plan.PricePerMonth != nil && *plan.PricePerMonth > 0 {
		unitAmount = int64(math.Round(*plan.PricePerMonth * 100))
	} else {
		unitAmount = int64(math.Round(plan.PricePerSeat * 100))
	}

	planCode := plan.PlanTypeCode

	// Get or create Stripe product and price
	productIdempotencyKey := fmt.Sprintf("reg_product_%s", planCode)
	prod, err := s.stripeClient.GetOrCreateProduct(ctx, planCode, plan.Name, productIdempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get or create Stripe product")
	}

	priceIdempotencyKey := fmt.Sprintf("reg_price_%s", planCode)
	stripePrice, err := s.stripeClient.GetOrCreatePrice(ctx, prod.ID, unitAmount, planCode, priceIdempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get or create Stripe price")
	}

	// Determine seat quantity
	quantity := int64(userCount)
	if plan.SeatMinimum != nil && int64(*plan.SeatMinimum) > quantity {
		quantity = int64(*plan.SeatMinimum)
	}
	if quantity < 1 {
		quantity = 1
	}

	monthlyBillAmount := unitAmount * quantity

	// Try to find an active subscription
	subs, err := s.stripeClient.ListSubscriptions(ctx, *customerID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to list subscriptions")
	}

	// No active subscription — return simple estimate
	if len(subs) == 0 {
		return &domain.ProrationPreview{
			CreditAmount:                0,
			ChargeAmount:                monthlyBillAmount,
			NetAmount:                   monthlyBillAmount,
			FormattedNetAmount:          formatAmount(monthlyBillAmount),
			IsCredit:                    false,
			TotalInvoiceAmount:          monthlyBillAmount,
			FormattedTotalInvoiceAmount: formatAmount(monthlyBillAmount),
			MonthlyBillAmount:           monthlyBillAmount,
			FormattedMonthlyBillAmount:  formatAmount(monthlyBillAmount),
			LineItems: []domain.ProrationLineItem{
				{
					Description: fmt.Sprintf("%s plan — %d seat(s) x %s/mo", plan.Name, quantity, formatAmount(unitAmount)),
					Amount:      monthlyBillAmount,
					IsProration: false,
				},
			},
		}, nil
	}

	// Has active subscription — use Stripe invoice preview
	activeSub := subs[0]
	var existingItemID string
	if len(activeSub.Items.Data) > 0 {
		existingItemID = activeSub.Items.Data[0].ID
	}

	items := []*stripe.InvoiceCreatePreviewSubscriptionDetailsItemParams{
		{
			ID:       stripe.String(existingItemID),
			Price:    stripe.String(stripePrice.ID),
			Quantity: stripe.Int64(quantity),
		},
	}

	inv, err := s.stripeClient.CreateInvoicePreview(ctx, *customerID, activeSub.ID, items)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to create invoice preview")
	}

	// Process invoice lines
	var creditAmount, chargeAmount int64
	var lineItems []domain.ProrationLineItem
	for _, line := range inv.Lines.Data {
		lineItems = append(lineItems, domain.ProrationLineItem{
			Description: line.Description,
			Amount:      line.Amount,
			IsProration: isLineItemProration(line),
		})

		if line.Amount < 0 {
			creditAmount += -line.Amount
		} else {
			chargeAmount += line.Amount
		}
	}

	netAmount := chargeAmount - creditAmount
	isCredit := netAmount < 0

	totalInvoiceAmount := inv.Total
	if totalInvoiceAmount < 0 {
		totalInvoiceAmount = 0
	}

	return &domain.ProrationPreview{
		CreditAmount:                creditAmount,
		ChargeAmount:                chargeAmount,
		NetAmount:                   netAmount,
		FormattedNetAmount:          formatAmount(netAmount),
		IsCredit:                    isCredit,
		TotalInvoiceAmount:          totalInvoiceAmount,
		FormattedTotalInvoiceAmount: formatAmount(totalInvoiceAmount),
		MonthlyBillAmount:           monthlyBillAmount,
		FormattedMonthlyBillAmount:  formatAmount(monthlyBillAmount),
		LineItems:                   lineItems,
	}, nil
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

	repo := s.repos.NewAccountUsageRepo()

	customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if customerID != nil {
		return &domain.EnsureBillingCustomerResult{
			StripeCustomerID: *customerID,
			Created:          false,
		}, nil
	}

	accountName, _, apiErr := repo.GetAccountNameAndPlanCode(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	adminEmail, apiErr := repo.GetAdminEmailByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if adminEmail == "" {
		return nil, apierror.NewValidationError("No admin user found for account.")
	}

	idempotencyKey := fmt.Sprintf("ensure_customer_%s", accountID)
	metadata := map[string]string{
		"accountID":   accountID,
		"accountName": accountName,
	}

	customer, err := s.stripeClient.CreateCustomer(ctx, adminEmail, accountName, idempotencyKey, metadata)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to create Stripe customer")
	}

	if apiErr := repo.UpdateStripeCustomerIDByAccountID(ctx, customer.ID, accountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.EnsureBillingCustomerResult{
		StripeCustomerID: customer.ID,
		Created:          true,
	}, nil
}

func (s *billingSvcImpl) SwitchPlan(ctx context.Context, accountID string, planID string) (*domain.SwitchPlanResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.switch_plan")
	defer span.End()

	repo := s.repos.NewAccountUsageRepo()

	// Get Stripe customer ID for the account
	customerID, apiErr := repo.GetStripeCustomerIDByAccountID(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Get target plan by type ID + limits
	planRepo := s.repos.NewPricingPlanRepo()
	plan, apiErr := planRepo.GetPlanByTypeID(ctx, planID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limits, apiErr := planRepo.GetPlanLimitsByTypeID(ctx, plan.TypeID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	plan.Limits = limits

	// We need a Stripe customer for any plan switch
	if customerID == nil {
		return nil, apierror.NewValidationError("Account has no Stripe customer. Call EnsureBillingCustomer first.")
	}

	// Calculate unit amount and quantity for target plan
	planCode := plan.PlanTypeCode
	var unitAmount int64
	var quantity int64

	if plan.PlanTypeCode == string(constants.PlanCodeFree) {
		// Free plan: $0, 1 seat
		unitAmount = 0
		quantity = 1
	} else {
		if plan.PricePerMonth != nil && *plan.PricePerMonth > 0 {
			unitAmount = int64(math.Round(*plan.PricePerMonth * 100))
		} else {
			unitAmount = int64(math.Round(plan.PricePerSeat * 100))
		}

		userCount, countErr := repo.CountUsersByAccountID(ctx, accountID)
		if countErr != nil {
			return nil, tracing.Trace(span, countErr)
		}

		quantity = int64(userCount)
		if plan.SeatMinimum != nil && int64(*plan.SeatMinimum) > quantity {
			quantity = int64(*plan.SeatMinimum)
		}
		if quantity < 1 {
			quantity = 1
		}
	}

	// Get or create Stripe product and price for the target plan
	productIdempotencyKey := fmt.Sprintf("reg_product_%s", planCode)
	prod, err := s.stripeClient.GetOrCreateProduct(ctx, planCode, plan.Name, productIdempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get or create Stripe product")
	}

	priceIdempotencyKey := fmt.Sprintf("reg_price_%s", planCode)
	stripePrice, err := s.stripeClient.GetOrCreatePrice(ctx, prod.ID, unitAmount, planCode, priceIdempotencyKey)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get or create Stripe price")
	}

	// Find the existing subscription — check our DB first, then Stripe
	subInfo, apiErr := repo.GetAccountSubscriptionInfo(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	subs, err := s.stripeClient.ListSubscriptions(ctx, *customerID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to list subscriptions")
	}

	// Prefer the subscription stored in our DB; fall back to the first one from Stripe
	var activeSub *stripe.Subscription
	if subInfo.StripeSubscriptionID != nil {
		for _, sub := range subs {
			if sub.ID == *subInfo.StripeSubscriptionID {
				activeSub = sub
				break
			}
		}
	}
	if activeSub == nil && len(subs) > 0 {
		activeSub = subs[0]
	}

	// If there's an existing subscription, update it to the new plan's price
	if activeSub != nil {
		var existingItemID string
		if len(activeSub.Items.Data) > 0 {
			existingItemID = activeSub.Items.Data[0].ID
		}

		items := []*stripe.SubscriptionItemsParams{
			{
				ID:       stripe.String(existingItemID),
				Price:    stripe.String(stripePrice.ID),
				Quantity: stripe.Int64(quantity),
			},
		}

		updatedSub, err := s.stripeClient.UpdateSubscription(ctx, activeSub.ID, items)
		if err != nil {
			span.RecordError(err)
			return nil, apierror.NewInternalError(err, "failed to update subscription")
		}

		// Propagate the plan change to core-service immediately so the seat
		// adjustment runs with the caller's identity (the webhook path has no
		// identity and cannot safely deactivate users).
		status := string(updatedSub.Status)
		subID := updatedSub.ID
		var periodEnd *time.Time
		if len(updatedSub.Items.Data) > 0 && updatedSub.Items.Data[0].CurrentPeriodEnd > 0 {
			t := time.Unix(updatedSub.Items.Data[0].CurrentPeriodEnd, 0)
			periodEnd = &t
		}
		idempotencyKey := fmt.Sprintf("switch_plan_%s_%s", accountID, planID)
		if switchErr := s.coreClient.UpdateAccountSubscription(ctx, idempotencyKey, accountID, &status, planCode, &subID, periodEnd, nil); switchErr != nil {
			return nil, switchErr
		}

		return &domain.SwitchPlanResult{
			Success:         true,
			RequiresPayment: false,
		}, nil
	}

	// No existing subscription — check if customer has a payment method on file
	paymentMethodIDs, err := s.stripeClient.ListPaymentMethods(ctx, *customerID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to list payment methods")
	}

	if len(paymentMethodIDs) > 0 {
		// Has payment method — create subscription directly
		newSub, createErr := s.stripeClient.CreateSubscription(ctx, *customerID, stripePrice.ID, quantity, paymentMethodIDs[0])
		if createErr != nil {
			span.RecordError(createErr)
			return nil, apierror.NewInternalError(createErr, "failed to create subscription")
		}

		status := newSub.Status
		periodEnd := newSub.CurrentPeriodEnd
		subID := newSub.ID
		idempotencyKey := fmt.Sprintf("switch_plan_%s_%s", accountID, planID)
		if switchErr := s.coreClient.UpdateAccountSubscription(ctx, idempotencyKey, accountID, &status, planCode, &subID, &periodEnd, nil); switchErr != nil {
			return nil, switchErr
		}

		return &domain.SwitchPlanResult{
			Success:         true,
			RequiresPayment: false,
		}, nil
	}

	// No payment methods at all — fall back to hosted checkout
	successURL := fmt.Sprintf("%s%s?plan_upgrade=complete&session_id={CHECKOUT_SESSION_ID}", s.frontendURL, constants.DashboardPathBillingPortal)
	cancelURL := fmt.Sprintf("%s%s", s.frontendURL, constants.DashboardPathBillingPortal)

	checkoutSession, err := s.stripeClient.CreateHostedCheckoutSession(ctx, domain.StripeHostedCheckoutInput{
		CustomerID: *customerID,
		PriceID:    stripePrice.ID,
		Quantity:   quantity,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
	})
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to create checkout session")
	}

	checkoutURL := checkoutSession.URL
	return &domain.SwitchPlanResult{
		Success:         true,
		RequiresPayment: true,
		CheckoutURL:     &checkoutURL,
	}, nil
}

func (s *billingSvcImpl) ConfirmPlanSwitch(ctx context.Context, accountID string, checkoutSessionID string, planID string) (*domain.ConfirmPlanSwitchResult, *apierror.APIError) {
	ctx, span := tracing.StartSpan(ctx, billingSvcTracer, "service.billing.confirm_plan_switch")
	defer span.End()

	// Get checkout session status from Stripe
	sessStatus, err := s.stripeClient.GetCheckoutSession(ctx, checkoutSessionID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get checkout session")
	}

	if sessStatus.Status != "complete" {
		return nil, apierror.NewValidationError("Checkout session is not complete. Current status: " + sessStatus.Status)
	}

	// Get subscription details from Stripe
	if sessStatus.SubscriptionID == "" {
		return nil, apierror.NewInternalError(nil, "checkout session has no subscription")
	}

	stripeSub, err := s.stripeClient.GetSubscription(ctx, sessStatus.SubscriptionID)
	if err != nil {
		span.RecordError(err)
		return nil, apierror.NewInternalError(err, "failed to get subscription from Stripe")
	}

	// Look up plan by type ID for plan code
	planRepo := s.repos.NewPricingPlanRepo()
	plan, apiErr := planRepo.GetPlanByTypeID(ctx, planID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Update account subscription via core client
	status := stripeSub.Status
	periodEnd := stripeSub.CurrentPeriodEnd
	subID := stripeSub.ID
	idempotencyKey := fmt.Sprintf("confirm_switch_%s_%s", accountID, checkoutSessionID)

	apiErr = s.coreClient.UpdateAccountSubscription(ctx, idempotencyKey, accountID, &status, plan.PlanTypeCode, &subID, &periodEnd, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	return &domain.ConfirmPlanSwitchResult{Success: true}, nil
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

func isLineItemProration(line *stripe.InvoiceLineItem) bool {
	if line.Parent != nil {
		return line.Parent.Type == stripe.InvoiceLineItemParentTypeInvoiceItemDetails
	}
	return false
}
