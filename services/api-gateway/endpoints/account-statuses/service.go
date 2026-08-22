package accountstatusep

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
)

type AccountStatusSvc interface {
	ListAccountStatuses(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError)
	GetAccountStatus(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError)
}

type AccountStatusSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type accountStatusSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountStatusSvcTracer = tracing.GetTracer("api-gateway.endpoints.account-statuses.service")

func (c *AccountStatusSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account status endpoint service: core client is required")
	}
	return nil
}

func NewAccountStatusSvc(config *AccountStatusSvcConfig) AccountStatusSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &accountStatusSvcImpl{coreClient: config.CoreClient}
}

func (m *accountStatusSvcImpl) ListAccountStatuses(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError) {
	pbReq := &pb.ListAccountStatusesRequest{Cursor: req.Cursor, Limit: req.Limit, Query: req.Query}
	resp, apiErr := grpcutil.CallRPC(ctx, accountStatusSvcTracer, "service.account_statuses.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountStatusesResponse, error) {
			return m.coreClient.ListAccountStatuses(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.AccountStatuses))
	for i, as := range resp.AccountStatuses {
		ids[i] = as.Id
	}
	loaded, apiErr := resourceloaders.LoadAccountStatuses(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.AccountStatus, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.AccountStatus)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

// GetAccountStatus resolves by id-or-code via the legacy GetAccountStatus gRPC,
// then loads the resolved record through the resourcekit loader.
func (m *accountStatusSvcImpl) GetAccountStatus(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, accountStatusSvcTracer, "service.account_statuses.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountStatusResponse, error) {
			return m.coreClient.GetAccountStatus(ctx, &pb.GetAccountStatusRequest{Identifier: req.AccountStatusID}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	loaded, apiErr := resourceloaders.LoadAccountStatuses(ctx, []string{resp.AccountStatus.Id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[resp.AccountStatus.Id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Account status not found.")
	}
	return v.(*apiresource.AccountStatus), nil
}
