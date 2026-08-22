package checkoutsessionep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type CheckoutSessionSvc interface {
	CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSessionResponse, *apierror.APIError)
}

type CheckoutSessionSvcConfig struct {
	// CoreSalesClient (required) is the core-service sales gRPC client.
	CoreSalesClient pb.CoreSalesServiceClient
}

type checkoutSessionSvcImpl struct {
	coreSalesClient pb.CoreSalesServiceClient
}

var checkoutSessionSvcTracer = tracing.GetTracer("api-gateway.endpoints.checkout-sessions.service")

func (c *CheckoutSessionSvcConfig) validate() error {
	if c.CoreSalesClient == nil {
		return fmt.Errorf("checkout session endpoint service: core sales client is required")
	}
	return nil
}

func NewCheckoutSessionSvc(config *CheckoutSessionSvcConfig) CheckoutSessionSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &checkoutSessionSvcImpl{
		coreSalesClient: config.CoreSalesClient,
	}
}

func (m *checkoutSessionSvcImpl) CreateCheckoutSession(ctx context.Context, req *CreateCheckoutSessionRequest) (*CheckoutSessionResponse, *apierror.APIError) {
	pbReq := &pb.CreateCustomerCheckoutSessionRequest{
		OrderId:         req.OrderID,
		OrderNumber:     req.OrderNumber,
		OrderTotalCents: req.OrderTotalCents,
	}

	pbReq.CustomerPo = req.CustomerPO.Ptr()

	resp, apiErr := grpcutil.CallRPC(ctx, checkoutSessionSvcTracer, "service.checkout_sessions.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateCustomerCheckoutSessionResponse, error) {
			return m.coreSalesClient.CreateCustomerCheckoutSession(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &CheckoutSessionResponse{
		Object:                      constants.ObjectTypeCheckoutSession,
		CheckoutSessionClientSecret: resp.ClientSecret,
	}, nil
}
