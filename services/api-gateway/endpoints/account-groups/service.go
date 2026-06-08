package accountgroupep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
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
	return &accountGroupSvcImpl{coreClient: config.CoreClient}
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
	ids := make([]string, len(resp.AccountGroups))
	for i, ag := range resp.AccountGroups {
		ids[i] = ag.Id
	}
	loaded, apiErr := resourceloaders.LoadAccountGroups(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.AccountGroup, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.AccountGroup)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *accountGroupSvcImpl) GetAccountGroup(ctx context.Context, req *RetrieveAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	return loadAccountGroupByID(ctx, req.AccountGroupID)
}

func (m *accountGroupSvcImpl) CreateAccountGroup(ctx context.Context, req *CreateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	commissionPolicy := constants.CommissionPolicyExempt
	if v, ok := req.CommissionPolicy.Value(); ok {
		commissionPolicy = v
	}
	freightPolicy := constants.FreightPolicyBilled
	if v, ok := req.FreightPolicy.Value(); ok {
		freightPolicy = v
	}
	pbReq := &pb.CreateAccountGroupRequest{
		Name:             req.Name,
		Type:             string(req.Type),
		CommissionPolicy: string(commissionPolicy),
		FreightPolicy:    string(freightPolicy),
		Description:      req.Description.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountGroupResponse, error) {
			return m.coreClient.CreateAccountGroup(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadAccountGroupByID(ctx, resp.AccountGroup.Id)
}

func (m *accountGroupSvcImpl) UpdateAccountGroup(ctx context.Context, req *UpdateAccountGroupRequest) (*apiresource.AccountGroup, *apierror.APIError) {
	pbReq := &pb.UpdateAccountGroupRequest{
		Id:               req.AccountGroupID,
		Name:             req.Name.Ptr(),
		Description:      field.StringClearableToProto(req.Description),
		CommissionPolicy: req.CommissionPolicy.Ptr().StringPtr(),
		FreightPolicy:    req.FreightPolicy.Ptr().StringPtr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountGroupResponse, error) {
			return m.coreClient.UpdateAccountGroup(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadAccountGroupByID(ctx, resp.AccountGroup.Id)
}

func (m *accountGroupSvcImpl) DeleteAccountGroup(ctx context.Context, req *DeleteAccountGroupRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAccountGroupRequest{Id: req.AccountGroupID}
	_, apiErr := grpcutil.CallRPC(ctx, accountGroupSvcTracer, "service.account_groups.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAccountGroup(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadAccountGroupByID wraps the single-ID load pattern used after every
// mutation and for the retrieve endpoint.
func loadAccountGroupByID(ctx context.Context, id string) (*apiresource.AccountGroup, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAccountGroups(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Account group not found.")
	}
	return v.(*apiresource.AccountGroup), nil
}
