package grpc

import (
	"context"

	"github.com/open-mrp/api/services/billing-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/billing"
	"github.com/open-mrp/api/shared/safeconv"

	"google.golang.org/grpc"
)

type billingHandler struct {
	pb.UnimplementedBillingServiceServer

	billingSvc       domain.BillingSvc
	stripeWebhookSvc domain.StripeWebhookSvc
}

func NewBillingHandler(server *grpc.Server, billingSvc domain.BillingSvc, stripeWebhookSvc domain.StripeWebhookSvc) *billingHandler {
	handler := &billingHandler{
		billingSvc:       billingSvc,
		stripeWebhookSvc: stripeWebhookSvc,
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

func (h *billingHandler) GetAccountUsage(ctx context.Context, req *pb.GetAccountUsageRequest) (*pb.GetAccountUsageResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	// The usage is the seller's own business: its seats, invoice and batch volume, subscription standing and agent spend.
	if identity.IsRelationActor() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("Customer and supplier portals cannot read the account's usage."))
	}
	if !identity.IsTargetAccountSet() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
		PlanName:                 result.PlanName,
		BaseFeeCents:             result.BaseFeeCents,
		BaseFeeInterval:          result.BaseFeeInterval,
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
	if !identity.IsTargetAccountSet() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
	if !identity.IsTargetAccountSet() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
		if !identity.IsTargetAccountSet() {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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

func (h *billingHandler) GetAgentSpend(ctx context.Context, req *pb.GetAgentSpendRequest) (*pb.GetAgentSpendResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

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
		if !identity.IsTargetAccountSet() {
			return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
		}
		accountID = identity.Target.AccountID
	}

	cents, apiErr := h.billingSvc.GetAgentSpendCents(ctx, accountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	rates, apiErr := h.billingSvc.GetAgentTokenRates(ctx, accountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbRates := make([]*pb.TokenRate, 0, len(rates))
	for _, r := range rates {
		pbRates = append(pbRates, &pb.TokenRate{
			Model:           r.Model,
			TokenType:       r.TokenType,
			UnitAmountCents: r.UnitAmountCents,
		})
	}

	return &pb.GetAgentSpendResponse{EstimatedSpendCents: cents, TokenRates: pbRates}, nil
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
	if !identity.IsTargetAccountSet() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
	if !identity.IsTargetAccountSet() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The OpenMRP-Account header is required."))
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
