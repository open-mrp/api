package accountstatusep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

type AccountStatusSvc interface {
	ListAccountStatuses(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError)
	GetAccountStatus(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError)
}

type AccountStatusSvcConfig struct {
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

	return &accountStatusSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *accountStatusSvcImpl) ListAccountStatuses(ctx context.Context, req *ListAccountStatusesRequest) (*apiresource.List[apiresource.AccountStatus], *apierror.APIError) {
	pbReq := &pb.ListAccountStatusesRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountStatusSvcTracer, "service.account_statuses.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountStatusesResponse, error) {
			return m.coreClient.ListAccountStatuses(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AccountStatusListPresenter(resp), nil
}

func (m *accountStatusSvcImpl) GetAccountStatus(ctx context.Context, req *RetrieveAccountStatusRequest) (*apiresource.AccountStatus, *apierror.APIError) {
	pbReq := &pb.GetAccountStatusRequest{
		Identifier: req.AccountStatusID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountStatusSvcTracer, "service.account_statuses.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountStatusResponse, error) {
			return m.coreClient.GetAccountStatus(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountStatusPresenter(resp.AccountStatus)
	return &result, nil
}
