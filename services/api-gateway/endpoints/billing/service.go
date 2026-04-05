package billingep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
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
	BillingClient pb.BillingServiceClient
	CoreClient    corepb.CoreServiceClient
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

	result := AccountUsagePresenter(resp)

	// Fetch spending cap from core-service to include in agent spend info.
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if ok && identity != nil && identity.Target != nil && identity.Target.AccountID != "" {
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

	return PricingPlansListPresenter(resp), nil
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

	return PlanChangePreviewPresenter(resp), nil
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

	return SwitchPlanPresenter(resp), nil
}

func (m *billingSvcImpl) GetSpendingCap(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.SpendingCapResponse, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || identity.Target == nil || identity.Target.AccountID == "" {
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
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.set_spending_cap", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*corepb.UpdateAgentSpendingCapResponse, error) {
			return m.coreClient.UpdateAgentSpendingCap(ctx, &corepb.UpdateAgentSpendingCapRequest{
				CapCents: req.CapCents,
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
