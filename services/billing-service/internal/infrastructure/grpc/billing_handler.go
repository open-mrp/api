package grpc

import (
	"context"
	"math"

	"github.com/augno/api/services/billing-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/billing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func safeIntToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

type billingHandler struct {
	pb.UnimplementedBillingServiceServer

	billingSvc       domain.BillingSvc
	stripeWebhookSvc domain.StripeWebhookSvc
	checkoutSvc      domain.CheckoutSvc
}

func NewBillingHandler(server *grpc.Server, billingSvc domain.BillingSvc, stripeWebhookSvc domain.StripeWebhookSvc, checkoutSvc domain.CheckoutSvc) *billingHandler {
	handler := &billingHandler{
		billingSvc:       billingSvc,
		stripeWebhookSvc: stripeWebhookSvc,
		checkoutSvc:      checkoutSvc,
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
		DisplayOrder:  safeIntToInt32(plan.DisplayOrder),
		IsHighlighted: plan.IsHighlighted,
		ButtonText:    plan.ButtonText,
	}

	if plan.PricePerMonth != nil {
		pbPlan.PricePerMonth = plan.PricePerMonth
	}
	if plan.SeatMinimum != nil {
		v := safeIntToInt32(*plan.SeatMinimum)
		pbPlan.SeatMinimum = &v
	}
	if plan.IncludesPreviousPlan != nil {
		pbPlan.IncludesPreviousPlan = plan.IncludesPreviousPlan
	}

	pbLimits := make([]*pb.PlanLimit, len(plan.Limits))
	for i, limit := range plan.Limits {
		pbLimit := &pb.PlanLimit{Key: limit.Key}
		if limit.Value != nil {
			v := safeIntToInt32(*limit.Value)
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
				v := safeIntToInt32(*limit.Value)
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
			DisplayOrder:    safeIntToInt32(plan.DisplayOrder),
			IsHighlighted:   plan.IsHighlighted,
			ButtonText:      plan.ButtonText,
		}

		if plan.PricePerMonth != nil {
			pbPlan.PricePerMonth = plan.PricePerMonth
		}
		if plan.SeatMinimum != nil {
			v := safeIntToInt32(*plan.SeatMinimum)
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

	cust, apiErr := h.checkoutSvc.CreateCustomer(ctx, req.Email, req.Name, req.IdempotencyKey, req.Metadata)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCustomerResponse{
		CustomerId: cust.ID,
	}, nil
}

func (h *billingHandler) GetCheckoutSessionStatus(ctx context.Context, req *pb.GetCheckoutSessionStatusRequest) (*pb.GetCheckoutSessionStatusResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.checkoutSvc.GetCheckoutSessionStatus(ctx, domain.GetCheckoutSessionStatusInput{
		CheckoutSessionID: req.CheckoutSessionId,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.GetCheckoutSessionStatusResponse{
		Status: result.Status,
	}
	if result.SubscriptionID != "" {
		resp.SubscriptionId = &result.SubscriptionID
	}
	if result.CustomerID != "" {
		resp.CustomerId = &result.CustomerID
	}

	return resp, nil
}

func (h *billingHandler) CreateCheckoutSession(ctx context.Context, req *pb.CreateCheckoutSessionRequest) (*pb.CreateCheckoutSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	result, apiErr := h.checkoutSvc.CreateCheckoutSession(ctx, domain.CreateCheckoutSessionInput{
		CustomerID:     req.CustomerId,
		PlanCode:       req.PlanCode,
		ReturnURL:      req.ReturnUrl,
		IdempotencyKey: req.IdempotencyKey,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateCheckoutSessionResponse{
		SessionId:      result.SessionID,
		ClientSecret:   result.ClientSecret,
		PublishableKey: result.PublishableKey,
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
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	result, apiErr := h.billingSvc.GetAccountUsage(ctx, *identity.TargetAccountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	resp := &pb.GetAccountUsageResponse{
		Seats:     &pb.UsageItem{Current: safeIntToInt32(result.Seats.Current)},
		Invoices:  &pb.UsageItem{Current: safeIntToInt32(result.Invoices.Current)},
		Batches:   &pb.UsageItem{Current: safeIntToInt32(result.Batches.Current)},
		Sandboxes: &pb.UsageItem{Current: safeIntToInt32(result.Sandboxes.Current)},
	}

	if result.Seats.Limit != nil {
		v := safeIntToInt32(*result.Seats.Limit)
		resp.Seats.Limit = &v
	}
	if result.Invoices.Limit != nil {
		v := safeIntToInt32(*result.Invoices.Limit)
		resp.Invoices.Limit = &v
	}
	if result.Batches.Limit != nil {
		v := safeIntToInt32(*result.Batches.Limit)
		resp.Batches.Limit = &v
	}
	if result.Sandboxes.Limit != nil {
		v := safeIntToInt32(*result.Sandboxes.Limit)
		resp.Sandboxes.Limit = &v
	}

	if result.Subscription != nil {
		sub := &pb.SubscriptionInfo{
			Status:            result.Subscription.Status,
			CancelAtPeriodEnd: result.Subscription.CancelAtPeriodEnd,
		}
		if result.Subscription.CurrentPeriodEnd != nil {
			sub.CurrentPeriodEnd = timestamppb.New(*result.Subscription.CurrentPeriodEnd)
		}
		if result.Subscription.TrialEnd != nil {
			sub.TrialEnd = timestamppb.New(*result.Subscription.TrialEnd)
		}
		if result.Subscription.CancelAt != nil {
			sub.CancelAt = timestamppb.New(*result.Subscription.CancelAt)
		}
		resp.Subscription = sub
	}

	return resp, nil
}

func (h *billingHandler) CreateBillingPortalSession(ctx context.Context, req *pb.CreateBillingPortalSessionRequest) (*pb.CreateBillingPortalSessionResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	url, apiErr := h.billingSvc.CreateBillingPortalSession(ctx, *identity.TargetAccountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateBillingPortalSessionResponse{Url: url}, nil
}

func (h *billingHandler) RequestEnterpriseUpgrade(ctx context.Context, req *pb.RequestEnterpriseUpgradeRequest) (*pb.RequestEnterpriseUpgradeResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
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
		AccountID: *identity.TargetAccountID,
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

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	result, apiErr := h.billingSvc.EnsureBillingCustomer(ctx, *identity.TargetAccountID)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EnsureBillingCustomerResponse{
		StripeCustomerId: result.StripeCustomerID,
		Created:          result.Created,
	}, nil
}

func (h *billingHandler) SwitchPlan(ctx context.Context, req *pb.SwitchPlanRequest) (*pb.SwitchPlanResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	result, apiErr := h.billingSvc.SwitchPlan(ctx, *identity.TargetAccountID, req.PlanId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.SwitchPlanResponse{
		Success:         result.Success,
		RequiresPayment: result.RequiresPayment,
		CheckoutUrl:     result.CheckoutURL,
	}, nil
}

func (h *billingHandler) ConfirmPlanSwitch(ctx context.Context, req *pb.ConfirmPlanSwitchRequest) (*pb.ConfirmPlanSwitchResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	result, apiErr := h.billingSvc.ConfirmPlanSwitch(ctx, *identity.TargetAccountID, req.CheckoutSessionId, req.PlanId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ConfirmPlanSwitchResponse{
		Success: result.Success,
	}, nil
}

func (h *billingHandler) GetProrationPreview(ctx context.Context, req *pb.GetProrationPreviewRequest) (*pb.GetProrationPreviewResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if !identity.IsInternalUser() && !identity.IsAdmin() {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthorizationError("You are not authorized to perform this action."))
	}
	if identity.TargetAccountID == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Target account ID is required."))
	}

	result, apiErr := h.billingSvc.GetProrationPreview(ctx, *identity.TargetAccountID, req.PlanId)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLineItems := make([]*pb.ProrationLineItem, len(result.LineItems))
	for i, li := range result.LineItems {
		pbLineItems[i] = &pb.ProrationLineItem{
			Description: li.Description,
			Amount:      li.Amount,
			IsProration: li.IsProration,
		}
	}

	return &pb.GetProrationPreviewResponse{
		Preview: &pb.ProrationPreview{
			CreditAmount:                result.CreditAmount,
			ChargeAmount:                result.ChargeAmount,
			NetAmount:                   result.NetAmount,
			FormattedNetAmount:          result.FormattedNetAmount,
			IsCredit:                    result.IsCredit,
			TotalInvoiceAmount:          result.TotalInvoiceAmount,
			FormattedTotalInvoiceAmount: result.FormattedTotalInvoiceAmount,
			MonthlyBillAmount:           result.MonthlyBillAmount,
			FormattedMonthlyBillAmount:  result.FormattedMonthlyBillAmount,
			LineItems:                   pbLineItems,
		},
	}, nil
}
