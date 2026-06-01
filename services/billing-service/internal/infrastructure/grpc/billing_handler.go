package grpc

import (
	"context"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/billing"
	"github.com/augno/api/shared/safeconv"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type billingHandler struct {
	pb.UnimplementedBillingServiceServer

	billingSvc           domain.BillingSvc
	stripeWebhookSvc     domain.StripeWebhookSvc
	stripePublishableKey string
}

func NewBillingHandler(server *grpc.Server, billingSvc domain.BillingSvc, stripeWebhookSvc domain.StripeWebhookSvc, stripePublishableKey ...string) *billingHandler {
	var pubKey string
	if len(stripePublishableKey) > 0 {
		pubKey = stripePublishableKey[0]
	}
	handler := &billingHandler{
		billingSvc:           billingSvc,
		stripeWebhookSvc:     stripeWebhookSvc,
		stripePublishableKey: pubKey,
	}

	pb.RegisterBillingServiceServer(server, handler)
	return handler
}

func (h *billingHandler) ProcessWebhookEvent(ctx context.Context, req *pb.ProcessWebhookEventRequest) (*pb.ProcessWebhookEventResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.stripeWebhookSvc.ProcessWebhookEvent(ctx, domain.ProcessWebhookEventInput{
		RawPayload:      req.RawPayload,
		StripeSignature: req.StripeSignature,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ProcessWebhookEventResponse{
		Success: result.Success,
	}, nil
}

func (h *billingHandler) GetPlanByCode(ctx context.Context, req *pb.GetPlanByCodeRequest) (*pb.GetPlanByCodeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	plan, apiErr := h.billingSvc.GetPlanByCode(ctx, req.PlanCode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPlan := &pb.PricingPlan{
		Id:            plan.TypeID,
		Name:          plan.Name,
		PlanTypeCode:  plan.PlanTypeCode,
		PricePerSeat:  plan.PricePerSeat,
		DisplayOrder:  safeconv.IntToInt32(plan.DisplayOrder),
		IsHighlighted: plan.IsHighlighted,
		ButtonText:    plan.ButtonText,
	}

	if plan.PricePerMonth != nil {
		pbPlan.PricePerMonth = plan.PricePerMonth
	}
	if plan.SeatMinimum != nil {
		v := safeconv.IntToInt32(*plan.SeatMinimum)
		pbPlan.SeatMinimum = &v
	}
	if plan.IncludesPreviousPlan != nil {
		pbPlan.IncludesPreviousPlan = plan.IncludesPreviousPlan
	}

	pbLimits := make([]*pb.PlanLimit, len(plan.Limits))
	for i, limit := range plan.Limits {
		pbLimit := &pb.PlanLimit{Key: limit.Key}
		if limit.Value != nil {
			v := safeconv.IntToInt32(*limit.Value)
			pbLimit.Value = &v
		}
		pbLimits[i] = pbLimit
	}
	pbPlan.Limits = pbLimits

	return &pb.GetPlanByCodeResponse{Plan: pbPlan}, nil
}

func (h *billingHandler) ListPricingPlans(ctx context.Context, req *pb.ListPricingPlansRequest) (*pb.ListPricingPlansResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.billingSvc.ListPricingPlans(ctx, domain.ListPricingPlansInput{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbPlans := make([]*pb.PricingPlan, len(result.Plans))
	for i, plan := range result.Plans {
		pbLimits := make([]*pb.PlanLimit, len(plan.Limits))
		for j, limit := range plan.Limits {
			pbLimit := &pb.PlanLimit{
				Key: limit.Key,
			}
			if limit.Value != nil {
				v := safeconv.IntToInt32(*limit.Value)
				pbLimit.Value = &v
			}
			pbLimits[j] = pbLimit
		}

		pbPlan := &pb.PricingPlan{
			Id:              plan.TypeID,
			Name:            plan.Name,
			PlanTypeCode:    plan.PlanTypeCode,
			PricePerSeat:    plan.PricePerSeat,
			Limits:          pbLimits,
			DisplayFeatures: plan.DisplayFeatures,
			DisplayOrder:    safeconv.IntToInt32(plan.DisplayOrder),
			IsHighlighted:   plan.IsHighlighted,
			ButtonText:      plan.ButtonText,
		}

		if plan.PricePerMonth != nil {
			pbPlan.PricePerMonth = plan.PricePerMonth
		}
		if plan.SeatMinimum != nil {
			v := safeconv.IntToInt32(*plan.SeatMinimum)
			pbPlan.SeatMinimum = &v
		}
		if plan.IncludesPreviousPlan != nil {
			pbPlan.IncludesPreviousPlan = plan.IncludesPreviousPlan
		}

		pbPlans[i] = pbPlan
	}

	return &pb.ListPricingPlansResponse{
		Plans: pbPlans,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *billingHandler) CreateCustomer(ctx context.Context, req *pb.CreateCustomerRequest) (*pb.CreateCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	accountID := req.Metadata["account_id"]

	var result *domain.EnsureBillingCustomerResult
	var apiErr *apierror.APIError

	if accountID != "" {
		result, apiErr = h.billingSvc.EnsureBillingCustomer(ctx, accountID)
	} else {
		result, apiErr = h.billingSvc.CreateRegistrationCustomer(ctx, req.Email, req.Name, req.IdempotencyKey, req.Metadata)
	}
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomerResponse{
		CustomerId: result.StripeCustomerID,
	}, nil
}

func (h *billingHandler) GetAccountUsage(ctx context.Context, req *pb.GetAccountUsageRequest) (*pb.GetAccountUsageResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	result, apiErr := h.billingSvc.GetAccountUsage(ctx, identity.Target.AccountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.GetAccountUsageResponse{
		Seats:                    &pb.UsageItem{Current: safeconv.IntToInt32(result.Seats.Current)},
		Invoices:                 &pb.UsageItem{Current: safeconv.IntToInt32(result.Invoices.Current)},
		Batches:                  &pb.UsageItem{Current: safeconv.IntToInt32(result.Batches.Current)},
		Sandboxes:                &pb.UsageItem{Current: safeconv.IntToInt32(result.Sandboxes.Current)},
		EstimatedAgentSpendCents: result.EstimatedAgentSpendCents,
	}

	if result.Seats.Limit != nil {
		v := safeconv.IntToInt32(*result.Seats.Limit)
		resp.Seats.Limit = &v
	}
	if result.Invoices.Limit != nil {
		v := safeconv.IntToInt32(*result.Invoices.Limit)
		resp.Invoices.Limit = &v
	}
	if result.Batches.Limit != nil {
		v := safeconv.IntToInt32(*result.Batches.Limit)
		resp.Batches.Limit = &v
	}
	if result.Sandboxes.Limit != nil {
		v := safeconv.IntToInt32(*result.Sandboxes.Limit)
		resp.Sandboxes.Limit = &v
	}

	if result.Subscription != nil {
		resp.Subscription = &pb.SubscriptionInfo{
			ServicingStatus:  result.Subscription.ServicingStatus,
			CollectionStatus: result.Subscription.CollectionStatus,
		}
	}

	if result.AgentTokenDetail != nil {
		d := result.AgentTokenDetail
		resp.AgentTokenDetail = &pb.AgentTokenUsageDetail{
			IncludedTokens:              d.IncludedTokens,
			UsedTokens:                  d.UsedTokens,
			InputTokens:                 d.InputTokens,
			OutputTokens:                d.OutputTokens,
			AdditionalTokensPurchased:   d.AdditionalTokensPurchased,
			TotalAvailable:              d.TotalAvailable,
			CurrentPeriodCost:           d.CurrentPeriodCost,
			BillingPeriodEnd:            timestamppb.New(d.BillingPeriodEnd),
			OverageCostPerMillionTokens: d.OverageCostPerMillionTokens,
		}
	}

	return resp, nil
}

func (h *billingHandler) CreateBillingPortalSession(ctx context.Context, req *pb.CreateBillingPortalSessionRequest) (*pb.CreateBillingPortalSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	url, apiErr := h.billingSvc.CreateBillingPortalSession(ctx, identity.Target.AccountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateBillingPortalSessionResponse{Url: url}, nil
}

func (h *billingHandler) RequestEnterpriseUpgrade(ctx context.Context, req *pb.RequestEnterpriseUpgradeRequest) (*pb.RequestEnterpriseUpgradeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	var actorID string
	var actorName string
	if identity.Actor != nil {
		actorID = identity.Actor.ID
		if identity.Actor.Name != nil {
			actorName = *identity.Actor.Name
		}
	}

	result, apiErr := h.billingSvc.RequestEnterpriseUpgrade(ctx, domain.RequestEnterpriseUpgradeInput{
		AccountID: identity.Target.AccountID,
		ActorID:   actorID,
		ActorName: actorName,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.RequestEnterpriseUpgradeResponse{Success: result.Success}, nil
}

func (h *billingHandler) EnsureBillingCustomer(ctx context.Context, req *pb.EnsureBillingCustomerRequest) (*pb.EnsureBillingCustomerResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	var accountID string
	if req.AccountId != nil && *req.AccountId != "" {
		accountID = *req.AccountId
	} else {
		identity, ok := appctx.GetIdentityFromContext(ctx)
		if !ok || identity == nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
		}
		if !identity.IsInternalUser() || !identity.IsAdmin() {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
		}
		if identity.Target == nil {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
		}
		accountID = identity.Target.AccountID
	}

	result, apiErr := h.billingSvc.EnsureBillingCustomer(ctx, accountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.EnsureBillingCustomerResponse{
		StripeCustomerId: result.StripeCustomerID,
		Created:          result.Created,
	}
	if result.BillingProfileID != nil {
		resp.BillingProfileId = result.BillingProfileID
	}

	return resp, nil
}

func (h *billingHandler) SwitchPlan(ctx context.Context, req *pb.SwitchPlanRequest) (*pb.SwitchPlanResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	result, apiErr := h.billingSvc.SwitchPlan(ctx, identity.Target.AccountID, req.PlanId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.SwitchPlanResponse{
		Success: result.Success,
	}
	if result.IntentID != nil {
		resp.IntentId = result.IntentID
	}

	return resp, nil
}

func (h *billingHandler) PreviewPlanChange(ctx context.Context, req *pb.PreviewPlanChangeRequest) (*pb.PreviewPlanChangeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() || !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	result, apiErr := h.billingSvc.PreviewPlanChange(ctx, identity.Target.AccountID, req.PlanId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLineItems := make([]*pb.PlanChangePreviewLineItem, len(result.LineItems))
	for i, li := range result.LineItems {
		pbLineItems[i] = &pb.PlanChangePreviewLineItem{
			Description: li.Description,
			Amount:      li.Amount,
		}
	}

	return &pb.PreviewPlanChangeResponse{
		Preview: &pb.PlanChangePreview{
			NetAmount:                  result.NetAmount,
			FormattedNetAmount:         result.FormattedNetAmount,
			MonthlyBillAmount:          result.MonthlyBillAmount,
			FormattedMonthlyBillAmount: result.FormattedMonthlyBillAmount,
			LineItems:                  pbLineItems,
			IsEstimate:                 result.IsEstimate,
		},
	}, nil
}

func (h *billingHandler) SetupBillingProfile(ctx context.Context, req *pb.SetupBillingProfileRequest) (*pb.SetupBillingProfileResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	accountID := req.GetAccountId()
	if accountID == "" {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.billingSvc.SetupBillingProfile(ctx, accountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SetupBillingProfileResponse{
		BillingProfileId: result.ProfileID,
		BillingCadenceId: result.CadenceID,
	}, nil
}

func (h *billingHandler) SubscribeToPricingPlan(ctx context.Context, req *pb.SubscribeToPricingPlanRequest) (*pb.SubscribeToPricingPlanResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	if req.StripeCustomerId == "" {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationError("stripe_customer_id is required"))
	}
	if req.PlanCode == "" {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationError("plan_code is required"))
	}

	apiErr := h.billingSvc.SubscribeToPricingPlan(ctx, req.StripeCustomerId, req.PlanCode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SubscribeToPricingPlanResponse{}, nil
}

func (h *billingHandler) CreateSetupIntent(ctx context.Context, req *pb.CreateSetupIntentRequest) (*pb.CreateSetupIntentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.billingSvc.CreateSetupIntent(ctx, req.CustomerId, req.IdempotencyKey)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateSetupIntentResponse{
		SetupIntentId:  result.SetupIntentID,
		ClientSecret:   result.ClientSecret,
		PublishableKey: h.stripePublishableKey,
	}, nil
}

func (h *billingHandler) GetSetupIntentStatus(ctx context.Context, req *pb.GetSetupIntentStatusRequest) (*pb.GetSetupIntentStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.billingSvc.GetSetupIntentStatus(ctx, req.SetupIntentId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetSetupIntentStatusResponse{
		Status:          result.Status,
		PaymentMethodId: result.PaymentMethodID,
	}, nil
}

func (h *billingHandler) ValidateStripePricingPlan(ctx context.Context, req *pb.ValidateStripePricingPlanRequest) (*pb.ValidateStripePricingPlanResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if req.PlanCode == "" {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationError("plan_code is required"))
	}

	apiErr := h.billingSvc.ValidateStripePricingPlan(ctx, req.PlanCode)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ValidateStripePricingPlanResponse{Valid: true}, nil
}
