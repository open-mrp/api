package ediep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type EDISvc interface {
	PullOrders(ctx context.Context, req *PullEDIOrdersRequest) (*apiresource.MessageResource, *apierror.APIError)
	ResubmitInvoice(ctx context.Context, req *ResubmitEDIInvoiceRequest) (*apiresource.MessageResource, *apierror.APIError)
}

type EDISvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type ediSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var ediSvcTracer = tracing.GetTracer("api-gateway.endpoints.edi.service")

func (c *EDISvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("edi endpoint service: core client is required")
	}
	return nil
}

func NewEDISvc(config *EDISvcConfig) EDISvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &ediSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *ediSvcImpl) PullOrders(ctx context.Context, req *PullEDIOrdersRequest) (*apiresource.MessageResource, *apierror.APIError) {
	pbReq := &pb.PullEDIOrdersRequest{}

	resp, apiErr := grpcutil.CallRPC(ctx, ediSvcTracer, "service.edi.pull_orders", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PullEDIOrdersResponse, error) {
			return m.coreClient.PullEDIOrders(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.MessageResource{
		Object:  constants.ObjectTypeMessage,
		Message: resp.Message,
	}, nil
}

func (m *ediSvcImpl) ResubmitInvoice(ctx context.Context, req *ResubmitEDIInvoiceRequest) (*apiresource.MessageResource, *apierror.APIError) {
	pbReq := &pb.ResubmitEDIInvoiceRequest{
		InvoiceId: req.InvoiceID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, ediSvcTracer, "service.edi.resubmit_invoice", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ResubmitEDIInvoiceResponse, error) {
			return m.coreClient.ResubmitEDIInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.MessageResource{
		Object:  constants.ObjectTypeMessage,
		Message: resp.Message,
	}, nil
}
