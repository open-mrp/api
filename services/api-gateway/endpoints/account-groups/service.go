package accountgroupep

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
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountGroupSvc interface {
	ListAccountGroups(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError)
	GetAccountGroup(ctx context.Context, req *RetrieveAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError)
	CreateAccountGroup(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError)
	UpdateAccountGroup(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError)
	DeleteAccountGroup(ctx context.Context, req *DeleteAccountGroupRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AccountGroupSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type accountGroupSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountGroupSvcTracer = tracing.GetTracer("api-gateway.endpoints.account-groups.service")

func (c *AccountGroupSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account group endpoint service: core client is required")
	}
	return nil
}

func NewAccountGroupSvc(config *AccountGroupSvcConfig) AccountGroupSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &accountGroupSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *accountGroupSvcImpl) ListAccountGroups(ctx context.Context, req *ListAccountGroupsRequest) (*apiresource.List[apiresource.AccountGroup], *apierror.APIError) {
	pbReq := &pb.ListAccountGroupsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
		Type:   req.Type.StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountGroupsResponse, error) {
			return m.coreClient.ListAccountGroups(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return AccountGroupListPresenter(ctx, resp), nil
}

func (m *accountGroupSvcImpl) GetAccountGroup(ctx context.Context, req *RetrieveAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	pbReq := &pb.GetAccountGroupRequest{
		Id: req.AccountGroupID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountGroupResponse, error) {
			return m.coreClient.GetAccountGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountGroupPresenter(resp.AccountGroup)
	return &result, nil
}

func (m *accountGroupSvcImpl) CreateAccountGroup(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	commissionPolicy := constants.CommissionPolicyExempt
	if req.CommissionPolicy != nil {
		commissionPolicy = *req.CommissionPolicy
	}
	freightPolicy := constants.FreightPolicyBilled
	if req.FreightPolicy != nil {
		freightPolicy = *req.FreightPolicy
	}

	pbReq := &pb.CreateAccountGroupRequest{
		Name:             req.Name,
		Type:             string(req.Type),
		CommissionPolicy: string(commissionPolicy),
		FreightPolicy:    string(freightPolicy),
		Description:      req.Description,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountGroupResponse, error) {
			return m.coreClient.CreateAccountGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountGroupPresenter(resp.AccountGroup)
	return &result, nil
}

func (m *accountGroupSvcImpl) UpdateAccountGroup(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	pbReq := &pb.UpdateAccountGroupRequest{
		Id:               req.AccountGroupID,
		Name:             req.Name,
		Description:      req.Description,
		CommissionPolicy: req.CommissionPolicy.StringPtr(),
		FreightPolicy:    req.FreightPolicy.StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountGroupResponse, error) {
			return m.coreClient.UpdateAccountGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountGroupPresenter(resp.AccountGroup)
	return &result, nil
}

func (m *accountGroupSvcImpl) DeleteAccountGroup(ctx context.Context, req *DeleteAccountGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAccountGroupRequest{
		Id: req.AccountGroupID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAccountGroup(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
