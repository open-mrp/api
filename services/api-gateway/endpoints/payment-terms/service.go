package paymenttermep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PaymentTermSvc interface {
	ListPaymentTerms(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError)
	GetPaymentTerm(ctx context.Context, req *RetrievePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError)
	CreatePaymentTerm(ctx context.Context, req *CreatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError)
	UpdatePaymentTerm(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError)
	DeletePaymentTerm(ctx context.Context, req *DeletePaymentTermRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type PaymentTermSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type paymentTermSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var paymentTermSvcTracer = tracing.GetTracer("api-gateway.endpoints.payment-terms.service")

func (c *PaymentTermSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("payment term endpoint service: core client is required")
	}
	return nil
}

func NewPaymentTermSvc(config *PaymentTermSvcConfig) PaymentTermSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &paymentTermSvcImpl{coreClient: config.CoreClient}
}

func (m *paymentTermSvcImpl) ListPaymentTerms(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError) {
	pbReq := &pb.ListPaymentTermsRequest{Cursor: req.Cursor, Limit: req.Limit, Query: req.Query}
	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPaymentTermsResponse, error) {
			return m.coreClient.ListPaymentTerms(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.PaymentTerms))
	for i, pt := range resp.PaymentTerms {
		ids[i] = pt.Id
	}
	loaded, apiErr := resourceloaders.LoadPaymentTerms(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.PaymentTerm, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.PaymentTerm)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *paymentTermSvcImpl) GetPaymentTerm(ctx context.Context, req *RetrievePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	return loadPaymentTermByID(ctx, req.PaymentTermID)
}

func (m *paymentTermSvcImpl) CreatePaymentTerm(ctx context.Context, req *CreatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePaymentTermResponse, error) {
			return m.coreClient.CreatePaymentTerm(ctx, &pb.CreatePaymentTermRequest{Name: req.Name}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadPaymentTermByID(ctx, resp.PaymentTerm.Id)
}

func (m *paymentTermSvcImpl) UpdatePaymentTerm(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePaymentTermResponse, error) {
			return m.coreClient.UpdatePaymentTerm(ctx, &pb.UpdatePaymentTermRequest{Id: req.PaymentTermID, Name: req.Name.Ptr()}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadPaymentTermByID(ctx, resp.PaymentTerm.Id)
}

func (m *paymentTermSvcImpl) DeletePaymentTerm(ctx context.Context, req *DeletePaymentTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	_, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeletePaymentTerm(ctx, &pb.DeletePaymentTermRequest{Id: req.PaymentTermID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadPaymentTermByID wraps the single-ID load pattern used after every
// mutation and for the retrieve endpoint. Same approach as carriers.
func loadPaymentTermByID(ctx context.Context, id string) (*apiresource.PaymentTerm, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadPaymentTerms(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Payment term not found.")
	}
	return v.(*apiresource.PaymentTerm), nil
}
