package billingep

import (
	"context"
	"fmt"
	"time"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	pb "github.com/augno/api/shared/proto/billing"
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
	ConfirmPlanSwitch(ctx context.Context, req *ConfirmPlanSwitchRequest) (*apiresource.ConfirmPlanSwitchResponse, *apierror.APIError)
}

type BillingSvcConfig struct {
	BillingClient pb.BillingServiceClient
}

type billingSvcImpl struct {
	billingClient pb.BillingServiceClient
}

var billingSvcTracer = tracing.GetTracer("api-gateway.endpoints.billing.service")

func (c *BillingSvcConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("billing endpoint service: billing client is required")
	}
	return nil
}

func NewBillingSvc(config *BillingSvcConfig) BillingSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &billingSvcImpl{
		billingClient: config.BillingClient,
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

	return &apiresource.BillingPortalSessionResponse{URL: resp.Url}, nil
}

func (m *billingSvcImpl) GetAccountUsage(ctx context.Context, _ *apiresource.EmptyResource) (*apiresource.AccountUsageResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.get_account_usage", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountUsageResponse, error) {
			return m.billingClient.GetAccountUsage(ctx, &pb.GetAccountUsageRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AccountUsagePresenter(resp), nil
}

func (m *billingSvcImpl) GetPricingPlans(ctx context.Context, req *apiresource.PaginationRequest) (*apiresource.List[apiresource.PricingPlan], *apierror.APIError) {
	pbReq := &pb.ListPricingPlansRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
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
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.get_proration_preview", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetProrationPreviewResponse, error) {
			return m.billingClient.GetProrationPreview(ctx, &pb.GetProrationPreviewRequest{
				PlanId: req.PlanID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ProrationPreviewPresenter(resp), nil
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
		StripeCustomerID: resp.StripeCustomerId,
		Created:          resp.Created,
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

func (m *billingSvcImpl) ConfirmPlanSwitch(ctx context.Context, req *ConfirmPlanSwitchRequest) (*apiresource.ConfirmPlanSwitchResponse, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, billingSvcTracer, "service.billing.confirm_plan_switch", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ConfirmPlanSwitchResponse, error) {
			return m.billingClient.ConfirmPlanSwitch(ctx, &pb.ConfirmPlanSwitchRequest{
				CheckoutSessionId: req.SessionID,
				PlanId:            req.PlanID,
			}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ConfirmPlanSwitchPresenter(resp), nil
}
