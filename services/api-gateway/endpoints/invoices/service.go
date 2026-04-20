package invoiceep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InvoiceSvc interface {
	ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.InvoiceSummary], *apierror.APIError)
	GetInvoice(ctx context.Context, req *GetInvoiceRequest) (*apiresource.Invoice, *apierror.APIError)
	UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.InvoiceSummary, *apierror.APIError)
	ListCustomerInvoices(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError)
}

type InvoiceSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type invoiceSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var invoiceSvcTracer = tracing.GetTracer("api-gateway.endpoints.invoices.service")

func (c *InvoiceSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("invoice endpoint service: core client is required")
	}
	return nil
}

func NewInvoiceSvc(config *InvoiceSvcConfig) InvoiceSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &invoiceSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *invoiceSvcImpl) ListInvoices(ctx context.Context, req *ListInvoicesRequest) (*apiresource.List[apiresource.InvoiceSummary], *apierror.APIError) {
	pbReq := &pb.ListInvoicesRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status,
		ItemIds:          req.ItemIDs,
		CustomerIds:      req.CustomerIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
	}

	if req.StartDate != nil {
		t, err := grpcutil.ParseDateString(*req.StartDate)
		if err == nil {
			pbReq.StartDate = timestamppb.New(t)
		}
	}
	if req.EndDate != nil {
		t, err := grpcutil.ParseDateString(*req.EndDate)
		if err == nil {
			pbReq.EndDate = timestamppb.New(t)
		}
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListInvoicesResponse, error) {
			return m.coreClient.ListInvoices(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return InvoiceListPresenter(resp), nil
}

func (m *invoiceSvcImpl) GetInvoice(ctx context.Context, req *GetInvoiceRequest) (*apiresource.Invoice, *apierror.APIError) {
	pbReq := &pb.GetInvoiceRequest{
		Id:       req.InvoiceID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetInvoiceResponse, error) {
			return m.coreClient.GetInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := InvoicePresenter(resp.Invoice)
	return &result, nil
}

func (m *invoiceSvcImpl) UpdateInvoice(ctx context.Context, req *UpdateInvoiceRequest) (*apiresource.InvoiceSummary, *apierror.APIError) {
	pbReq := &pb.UpdateInvoiceRequest{
		Id:           req.InvoiceID,
		Note:         req.Note,
		HasBeenSent:  req.HasBeenSent,
		IsEdiSent:    req.IsEdiSent,
		IsPaidInFull: req.IsPaidInFull,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateInvoiceResponse, error) {
			return m.coreClient.UpdateInvoice(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := InvoiceSummaryPresenter(resp.Invoice)
	return &result, nil
}

func (m *invoiceSvcImpl) ListCustomerInvoices(ctx context.Context, req *ListCustomerInvoicesRequest) (*apiresource.List[apiresource.InvoiceForPayment], *apierror.APIError) {
	pbReq := &pb.ListCustomerInvoicesRequest{
		CustomerAccountId:    req.CustomerAccountID,
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		IncludeChildAccounts: req.IncludeChildAccounts,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, invoiceSvcTracer, "service.invoices.list_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomerInvoicesResponse, error) {
			return m.coreClient.ListCustomerInvoices(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return CustomerInvoiceListPresenter(resp), nil
}
