package billingep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/billing"
	corepb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type BillingSvc interface {
	GetPricingPlans(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError)
	GetAccountUsage(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError)
	CreateBillingPortalSession(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.BillingPortalSessionResponse, *apierror.APIError)
	GetPlanChangePreview(ctx context.Context, req *GetPlanProrationRequest) (*apiresource.PlanChangeProration, *apierror.APIError)
	CreateEnterpriseInquiry(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnterpriseInquiry, *apierror.APIError)
	EnsureBillingCustomer(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.EnsureBillingCustomerResponse, *apierror.APIError)
	SwitchPlan(ctx context.Context, req *SwitchPlanRequest) (*apiresource.SwitchPlanResponse, *apierror.APIError)
	GetSpendingCap(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError)
	SetSpendingCap(ctx context.Context, req *SetSpendingCapRequest) (*apiresource.SpendingCapResponse, *apierror.APIError)
}

type BillingSvcConfig struct {
	// BillingClient (required) is the billing-service gRPC client.
	BillingClient pb.BillingServiceClient

	// CoreClient (required) is the core-service gRPC client.
	CoreClient corepb.CoreServiceClient
}

type billingSvcImpl struct {
	billingClient pb.BillingServiceClient
	coreClient    corepb.CoreServiceClient
}

var billingSvcTracer = tracing.GetTracer("api-gateway.endpoints.billing.service")

func (c *BillingSvcConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("billing endpoint service: billing client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("billing endpoint service: core client is required")
	}
	return nil
}

func NewBillingSvc(config *BillingSvcConfig) BillingSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &billingSvcImpl{
		billingClient: config.BillingClient,
		coreClient:    config.CoreClient,
	}
}

func (m *billingSvcImpl) CreateBillingPortalSession(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.BillingPortalSessionResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.create_billing_portal_session", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateBillingPortalSessionResponse, error) {
			return m.billingClient.CreateBillingPortalSession(ctx, &pb.CreateBillingPortalSessionRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.BillingPortalSessionResponse{Object: constants.ObjectTypeBillingPortalSessionResponse, URL: resp.Url}, nil
}

func (m *billingSvcImpl) GetAccountUsage(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.get_account_usage", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountUsageResponse, error) {
			return m.billingClient.GetAccountUsage(ctx, &pb.GetAccountUsageRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := accountUsageFromProto(resp)

	// Fetch spending cap from core-service to include in agent spend info. The
	// downstream GetAccountContext RPC is unguarded (auth bootstrap plumbing),
	// so require an authenticated identity here before calling it.
	identity, idErr := httptransport.GetIdentity(ctx)
	if idErr == nil && identity.IsAuthenticated() && identity.Target != nil && identity.Target.AccountID != "" {
		acctResp, capErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.get_spending_cap_for_usage", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetAccountContextResponse, error) {
				return m.coreClient.GetAccountContext(ctx, &corepb.GetAccountContextRequest{
					AccountId: identity.Target.AccountID,
				}, opts...)
			})
		if capErr == nil && result.AgentSpend != nil {
			result.AgentSpend.CapCents = acctResp.AgentMonthlySpendingCapCents
		}
	}

	return result, nil
}

func (m *billingSvcImpl) GetPricingPlans(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError) {
	pbReq := &pb.ListPricingPlansRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.list_pricing_plans", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPricingPlansResponse, error) {
			return m.billingClient.ListPricingPlans(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return pricingPlansListFromProto(ctx, resp), nil
}

func (m *billingSvcImpl) GetPlanChangePreview(ctx context.Context, req *GetPlanProrationRequest) (*apiresource.PlanChangeProration, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.preview_plan_change", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PreviewPlanChangeResponse, error) {
			return m.billingClient.PreviewPlanChange(ctx, &pb.PreviewPlanChangeRequest{
				PlanId: req.PlanID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return planChangePreviewFromProto(resp), nil
}

func (m *billingSvcImpl) EnsureBillingCustomer(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.EnsureBillingCustomerResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.ensure_billing_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EnsureBillingCustomerResponse, error) {
			return m.billingClient.EnsureBillingCustomer(ctx, &pb.EnsureBillingCustomerRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EnsureBillingCustomerResponse{
		Object:           constants.ObjectTypeEnsureBillingCustomerResponse,
		StripeCustomerID: resp.StripeCustomerId,
		Created:          resp.Created,
		BillingProfileID: resp.BillingProfileId,
	}, nil
}

func (m *billingSvcImpl) CreateEnterpriseInquiry(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.EnterpriseInquiry, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.request_enterprise_upgrade", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RequestEnterpriseUpgradeResponse, error) {
			return m.billingClient.RequestEnterpriseUpgrade(ctx, &pb.RequestEnterpriseUpgradeRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	inquiryID, idErr := id.GenID(id.EnterpriseInquiryIDPrefix, nil)
	if idErr != nil {
		return nil, idErr
	}

	return &apiresource.EnterpriseInquiry{
		ID:        inquiryID,
		Object:    constants.ObjectTypeEnterpriseInquiry,
		CreatedAt: time.Now(),
	}, nil
}

func (m *billingSvcImpl) SwitchPlan(ctx context.Context, req *SwitchPlanRequest) (*apiresource.SwitchPlanResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.switch_plan", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SwitchPlanResponse, error) {
			return m.billingClient.SwitchPlan(ctx, &pb.SwitchPlanRequest{
				PlanId: req.PlanID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return switchPlanFromProto(resp), nil
}

func (m *billingSvcImpl) GetSpendingCap(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError) {
	identity, idErr := httptransport.GetIdentity(ctx)
	if idErr != nil {
		return nil, idErr
	}
	// The downstream GetAccountContext RPC is unguarded (auth bootstrap
	// plumbing), so authentication must be enforced here at the gateway.
	if apiErr := identity.CheckIsAuthenticated(); apiErr != nil {
		return nil, apiErr
	}
	if identity.Target == nil || identity.Target.AccountID == "" {
		return nil, apierror.NewAuthenticationError("Missing account context")
	}

	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.get_spending_cap", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.GetAccountContextResponse, error) {
			return m.coreClient.GetAccountContext(ctx, &corepb.GetAccountContextRequest{
				AccountId: identity.Target.AccountID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.SpendingCapResponse{
		Object:   constants.ObjectTypeSpendingCapResponse,
		CapCents: resp.AgentMonthlySpendingCapCents,
	}, nil
}

func (m *billingSvcImpl) SetSpendingCap(ctx context.Context, req *SetSpendingCapRequest) (*apiresource.SpendingCapResponse, *apierror.APIError) {
	// Clearable contract: omitting cap_cents leaves the existing cap
	// unchanged (only an explicit null clears it), so skip the update and
	// return the current cap.
	if req.CapCents.IsUnset() {
		return m.GetSpendingCap(ctx, &apiresource.EmptyResource{})
	}

	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.set_spending_cap", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.UpdateAgentSpendingCapResponse, error) {
			return m.coreClient.UpdateAgentSpendingCap(ctx, &corepb.UpdateAgentSpendingCapRequest{
				CapCents: req.CapCents.ValuePtr(),
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.SpendingCapResponse{
		Object:   constants.ObjectTypeSpendingCapResponse,
		CapCents: resp.CapCents,
	}, nil
}

func pricingPlanFromProto(p *pb.PricingPlan) apiresource.PricingPlan {
	if p == nil {
		return apiresource.PricingPlan{}
	}

	limitSlice := make([]apiresource.PlanLimit, len(p.Limits))
	for i, l := range p.Limits {
		var value *int
		if l.Value != nil {
			v := int(*l.Value)
			value = &v
		}
		limitSlice[i] = apiresource.PlanLimit{
			Object: constants.ObjectTypePlanLimit,
			Key:    l.Key,
			Value:  value,
		}
	}

	var pricePerMonth *float64
	if p.PricePerMonth != nil {
		pricePerMonth = p.PricePerMonth
	}

	var seatMinimum *int
	if p.SeatMinimum != nil {
		v := int(*p.SeatMinimum)
		seatMinimum = &v
	}

	var includesPreviousPlan *string
	if p.IncludesPreviousPlan != nil {
		includesPreviousPlan = p.IncludesPreviousPlan
	}

	return apiresource.PricingPlan{
		ID:                   p.Id,
		Object:               constants.ObjectTypePricingPlan,
		Name:                 p.Name,
		PlanTypeCode:         constants.PublicPlanCode(p.PlanTypeCode),
		PricePerSeat:         p.PricePerSeat,
		PricePerMonth:        pricePerMonth,
		SeatMinimum:          seatMinimum,
		Limits:               apiresource.NewList(limitSlice, apiresource.PageInfo{}),
		DisplayFeatures:      p.DisplayFeatures,
		DisplayOrder:         int(p.DisplayOrder),
		IsHighlighted:        p.IsHighlighted,
		ButtonText:           p.ButtonText,
		IncludesPreviousPlan: includesPreviousPlan,
	}
}

func pricingPlansListFromProto(ctx context.Context, resp *pb.ListPricingPlansResponse) *apiresource.List[apiresource.PricingPlan] {
	if resp == nil {
		return apiresource.NewList([]apiresource.PricingPlan{}, apiresource.PageInfo{})
	}

	plans := make([]apiresource.PricingPlan, len(resp.Plans))
	for i, p := range resp.Plans {
		plans[i] = pricingPlanFromProto(p)
	}

	return apiresource.NewList(plans, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func usageItemFromProto(item *pb.UsageItem) apiresource.UsageItem {
	if item == nil {
		return apiresource.UsageItem{}
	}
	ui := apiresource.UsageItem{
		Object:  constants.ObjectTypeUsageItem,
		Current: int(item.Current),
	}
	if item.Limit != nil {
		v := int(*item.Limit)
		ui.Limit = &v
	}
	return ui
}

func planChangePreviewFromProto(resp *pb.PreviewPlanChangeResponse) *apiresource.PlanChangeProration {
	if resp == nil || resp.Preview == nil {
		return &apiresource.PlanChangeProration{}
	}

	p := resp.Preview
	lineItems := make([]apiresource.PlanChangeLineItem, len(p.LineItems))
	for i, li := range p.LineItems {
		lineItems[i] = apiresource.PlanChangeLineItem{
			Object:      constants.ObjectTypePlanChangeLineItem,
			Description: li.Description,
			Amount:      li.Amount,
		}
	}

	return &apiresource.PlanChangeProration{
		Object:                     constants.ObjectTypePlanChangeProration,
		NetAmount:                  p.NetAmount,
		FormattedNetAmount:         p.FormattedNetAmount,
		MonthlyBillAmount:          p.MonthlyBillAmount,
		FormattedMonthlyBillAmount: p.FormattedMonthlyBillAmount,
		LineItems:                  apiresource.NewList(lineItems, apiresource.PageInfo{}),
		IsEstimate:                 p.IsEstimate,
	}
}

func switchPlanFromProto(resp *pb.SwitchPlanResponse) *apiresource.SwitchPlanResponse {
	if resp == nil {
		return &apiresource.SwitchPlanResponse{}
	}

	return &apiresource.SwitchPlanResponse{
		Object:   constants.ObjectTypeSwitchPlanResponse,
		Success:  resp.Success,
		IntentID: resp.IntentId,
	}
}

func accountUsageFromProto(resp *pb.GetAccountUsageResponse) *apiresource.AccountUsageResponse {
	if resp == nil {
		return &apiresource.AccountUsageResponse{}
	}

	result := &apiresource.AccountUsageResponse{
		Object:    constants.ObjectTypeAccountUsageResponse,
		Seats:     usageItemFromProto(resp.Seats),
		Invoices:  usageItemFromProto(resp.Invoices),
		Batches:   usageItemFromProto(resp.Batches),
		Sandboxes: usageItemFromProto(resp.Sandboxes),
	}

	if resp.Subscription != nil {
		result.Subscription = &apiresource.SubscriptionInfo{
			Object:           constants.ObjectTypeSubscriptionInfo,
			ServicingStatus:  resp.Subscription.ServicingStatus,
			CollectionStatus: resp.Subscription.CollectionStatus,
		}
	}

	result.AgentSpend = &apiresource.AgentSpendInfo{
		Object:              constants.ObjectTypeAgentSpendInfo,
		EstimatedSpendCents: resp.EstimatedAgentSpendCents,
	}

	if resp.AgentTokenDetail != nil {
		d := resp.AgentTokenDetail
		billingPeriodEnd := ""
		if d.BillingPeriodEnd != nil {
			billingPeriodEnd = d.BillingPeriodEnd.AsTime().UTC().Format("2006-01-02T15:04:05Z")
		}
		result.AgentTokenDetail = &apiresource.AgentTokenDetail{
			Object:                      constants.ObjectTypeAgentTokenDetail,
			IncludedTokens:              d.IncludedTokens,
			UsedTokens:                  d.UsedTokens,
			InputTokens:                 d.InputTokens,
			OutputTokens:                d.OutputTokens,
			AdditionalTokensPurchased:   d.AdditionalTokensPurchased,
			TotalAvailable:              d.TotalAvailable,
			CurrentPeriodCost:           d.CurrentPeriodCost,
			BillingPeriodEnd:            billingPeriodEnd,
			OverageCostPerMillionTokens: d.OverageCostPerMillionTokens,
		}
	}

	return result
}
