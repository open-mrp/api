package receivableep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ReceivableSvc interface {
	ListReceivables(ctx context.Context, req *ListReceivablesRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError)
	ListReceivablesByCustomer(ctx context.Context, req *ListReceivablesByCustomerRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError)
	ExportReceivablesByCustomer(ctx context.Context, req *ExportReceivablesByCustomerRequest) (*httptransport.FileDownload, *apierror.APIError)
	EmailReceivablesForCustomer(ctx context.Context, req *EmailReceivablesForCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ReceivableSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type receivableSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var receivableSvcTracer = tracing.GetTracer("api-gateway.endpoints.receivables.service")

func (c *ReceivableSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("receivable endpoint service: core client is required")
	}
	return nil
}

func NewReceivableSvc(config *ReceivableSvcConfig) ReceivableSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &receivableSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *receivableSvcImpl) ListReceivables(ctx context.Context, req *ListReceivablesRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
	pbReq := &pb.ListReceivablesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	if req.CutoffDate != nil {
		pbReq.CutoffDate = timestamppb.New(*req.CutoffDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivableSvcTracer, "service.receivables.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListReceivablesResponse, error) {
			return m.coreClient.ListReceivables(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ReceivableEntryListPresenter(resp), nil
}

func (m *receivableSvcImpl) ListReceivablesByCustomer(ctx context.Context, req *ListReceivablesByCustomerRequest) (*apiresource.List[apiresource.ReceivableEntry], *apierror.APIError) {
	pbReq := &pb.ListReceivablesByCustomerRequest{
		CustomerAccountId: req.AccountID,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
	}

	if req.CutoffDate != nil {
		pbReq.CutoffDate = timestamppb.New(*req.CutoffDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivableSvcTracer, "service.receivables.list_by_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListReceivablesByCustomerResponse, error) {
			return m.coreClient.ListReceivablesByCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ReceivableEntryListPresenter(resp), nil
}

func (m *receivableSvcImpl) ExportReceivablesByCustomer(ctx context.Context, req *ExportReceivablesByCustomerRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportReceivablesByCustomerRequest{
		CustomerAccountId: req.AccountID,
	}

	if req.CutoffDate != nil {
		pbReq.CutoffDate = timestamppb.New(*req.CutoffDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, receivableSvcTracer, "service.receivables.export_by_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportReceivablesByCustomerResponse, error) {
			return m.coreClient.ExportReceivablesByCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	body, err := export.ReceivablesToCSV(resp.Receivables)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	filename := "receivables-report.csv"
	if req.CutoffDate != nil {
		filename = fmt.Sprintf("receivables-report-%s.csv", req.CutoffDate.Format("1/2/2006"))
	}

	return &httptransport.FileDownload{
		ContentType: export.CSVContentType,
		Filename:    filename,
		Body:        body,
	}, nil
}

func (m *receivableSvcImpl) EmailReceivablesForCustomer(ctx context.Context, req *EmailReceivablesForCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.EmailReceivablesForCustomerRequest{
		CustomerAccountId: req.AccountID,
		RecipientEmails:   req.RecipientEmails,
	}

	_, apiErr := grpcutil.CallRPC(ctx, receivableSvcTracer, "service.receivables.email_for_customer", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EmailReceivablesForCustomerResponse, error) {
			return m.coreClient.EmailReceivablesForCustomer(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
