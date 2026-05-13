package paymenttermep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	ownerutil "github.com/augno/api/services/api-gateway/internal/owner"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
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

	return &paymentTermSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *paymentTermSvcImpl) ListPaymentTerms(ctx context.Context, req *ListPaymentTermsRequest) (*apiresource.List[apiresource.PaymentTerm], *apierror.APIError) {
	pbReq := &pb.ListPaymentTermsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPaymentTermsResponse, error) {
			return m.coreClient.ListPaymentTerms(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	var ownerAccount *apiresource.Account
	for _, pt := range resp.PaymentTerms {
		if pt.AccountId != nil {
			ownerAccount = ownerutil.ResolveOwnerAccount(ctx, m.coreClient, pt.AccountId)
			break
		}
	}

	return PaymentTermListPresenter(resp, ownerAccount), nil
}

func (m *paymentTermSvcImpl) GetPaymentTerm(ctx context.Context, req *RetrievePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	pbReq := &pb.GetPaymentTermRequest{
		Id: req.PaymentTermID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPaymentTermResponse, error) {
			return m.coreClient.GetPaymentTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.PaymentTerm.AccountId)
	result := PaymentTermPresenter(resp.PaymentTerm, ownerAccount)
	return &result, nil
}

func (m *paymentTermSvcImpl) CreatePaymentTerm(ctx context.Context, req *CreatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	pbReq := &pb.CreatePaymentTermRequest{
		Name: req.Name,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePaymentTermResponse, error) {
			return m.coreClient.CreatePaymentTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.PaymentTerm.AccountId)
	result := PaymentTermPresenter(resp.PaymentTerm, ownerAccount)
	return &result, nil
}

func (m *paymentTermSvcImpl) UpdatePaymentTerm(ctx context.Context, req *UpdatePaymentTermRequest) (*apiresource.PaymentTerm, *apierror.APIError) {
	pbReq := &pb.UpdatePaymentTermRequest{
		Id:   req.PaymentTermID,
		Name: req.Name,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePaymentTermResponse, error) {
			return m.coreClient.UpdatePaymentTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	ownerAccount := ownerutil.ResolveOwnerAccount(ctx, m.coreClient, resp.PaymentTerm.AccountId)
	result := PaymentTermPresenter(resp.PaymentTerm, ownerAccount)
	return &result, nil
}

func (m *paymentTermSvcImpl) DeletePaymentTerm(ctx context.Context, req *DeletePaymentTermRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeletePaymentTermRequest{
		Id: req.PaymentTermID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, paymentTermSvcTracer, "service.payment_terms.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeletePaymentTerm(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
