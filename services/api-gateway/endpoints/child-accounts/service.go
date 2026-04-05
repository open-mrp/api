package childaccountep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type ChildAccountSvc interface {
	ListChildAccounts(ctx context.Context, req *ListChildAccountsRequest) (*apiresource.List[apiresource.ChildAccount], *apierror.APIError)
	AddChildAccount(ctx context.Context, req *AddChildAccountRequest) (*apiresource.ChildAccount, *apierror.APIError)
	RemoveChildAccount(ctx context.Context, req *RemoveChildAccountRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ChildAccountSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type childAccountSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var childAccountSvcTracer = tracing.GetTracer("api-gateway.endpoints.child-accounts.service")

func (c *ChildAccountSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("child account endpoint service: core client is required")
	}
	return nil
}

func NewChildAccountSvc(config *ChildAccountSvcConfig) ChildAccountSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &childAccountSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *childAccountSvcImpl) ListChildAccounts(ctx context.Context, req *ListChildAccountsRequest) (*apiresource.List[apiresource.ChildAccount], *apierror.APIError) {
	pbReq := &pb.ListChildAccountsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, childAccountSvcTracer, "service.child-accounts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListChildAccountsResponse, error) {
			return m.coreClient.ListChildAccounts(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ChildAccountListPresenter(resp), nil
}

func (m *childAccountSvcImpl) AddChildAccount(ctx context.Context, req *AddChildAccountRequest) (*apiresource.ChildAccount, *apierror.APIError) {
	pbReq := &pb.AddChildAccountRequest{
		ChildAccountId: req.ChildAccountID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, childAccountSvcTracer, "service.child-accounts.add", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AddChildAccountResponse, error) {
			return m.coreClient.AddChildAccount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ChildAccountPresenter(resp.ChildAccount)
	return &result, nil
}

func (m *childAccountSvcImpl) RemoveChildAccount(ctx context.Context, req *RemoveChildAccountRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.RemoveChildAccountRequest{
		ChildAccountId: req.ChildAccountID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, childAccountSvcTracer, "service.child-accounts.remove", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.RemoveChildAccount(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
