package accountgroupproductlineaccessep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountGroupProductLineAccessSvc interface {
	ListAccountGroupProductLineAccess(ctx context.Context, req *ListAccountGroupProductLineAccessRequest) (*apiresource.List[apiresource.AccountGroupProductLineAccess], *apierror.APIError)
	GetAccountGroupProductLineAccess(ctx context.Context, req *RetrieveAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError)
	CreateAccountGroupProductLineAccess(ctx context.Context, req *CreateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError)
	UpdateAccountGroupProductLineAccess(ctx context.Context, req *UpdateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError)
	DeleteAccountGroupProductLineAccess(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AccountGroupProductLineAccessSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type accountGroupProductLineAccessSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountGroupProductLineAccessSvcTracer = tracing.GetTracer("api-gateway.endpoints.account-group-product-line-access.service")

func (c *AccountGroupProductLineAccessSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account group product line access endpoint service: core client is required")
	}
	return nil
}

func NewAccountGroupProductLineAccessSvc(config *AccountGroupProductLineAccessSvcConfig) AccountGroupProductLineAccessSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &accountGroupProductLineAccessSvcImpl{coreClient: config.CoreClient}
}

func (m *accountGroupProductLineAccessSvcImpl) ListAccountGroupProductLineAccess(ctx context.Context, req *ListAccountGroupProductLineAccessRequest) (*apiresource.List[apiresource.AccountGroupProductLineAccess], *apierror.APIError) {
	pbReq := &pb.ListAccountGroupProductLineAccessRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupProductLineAccessSvcTracer, "service.account_group_product_line_access.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountGroupProductLineAccessResponse, error) {
			return m.coreClient.ListAccountGroupProductLineAccess(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	ids := make([]string, len(resp.Items))
	for i, item := range resp.Items {
		ids[i] = item.AccountGroupId
	}
	loaded, apiErr := resourceloaders.LoadAccountGroupProductLineAccess(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.AccountGroupProductLineAccess, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.AccountGroupProductLineAccess)))
		}
	}
	return apiresource.NewList(items, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *accountGroupProductLineAccessSvcImpl) GetAccountGroupProductLineAccess(ctx context.Context, req *RetrieveAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
	return loadAccountGroupProductLineAccessByID(ctx, req.AccountGroupID)
}

func (m *accountGroupProductLineAccessSvcImpl) CreateAccountGroupProductLineAccess(ctx context.Context, req *CreateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
	pbReq := &pb.CreateAccountGroupProductLineAccessRequest{
		AccountGroupId: req.AccountGroupID,
		ProductLineIds: req.ProductLineIDs,
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupProductLineAccessSvcTracer, "service.account_group_product_line_access.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountGroupProductLineAccessResponse, error) {
			return m.coreClient.CreateAccountGroupProductLineAccess(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadAccountGroupProductLineAccessByID(ctx, resp.Item.AccountGroupId)
}

func (m *accountGroupProductLineAccessSvcImpl) UpdateAccountGroupProductLineAccess(ctx context.Context, req *UpdateAccountGroupProductLineAccessRequest) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
	pbReq := &pb.UpdateAccountGroupProductLineAccessRequest{
		AccountGroupId: req.AccountGroupID,
	}
	if v, ok := req.ProductLineIDs.Value(); ok {
		pbReq.ProductLineIds = v
	}
	resp, apiErr := grpcutil.CallRPC(ctx, accountGroupProductLineAccessSvcTracer, "service.account_group_product_line_access.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountGroupProductLineAccessResponse, error) {
			return m.coreClient.UpdateAccountGroupProductLineAccess(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return loadAccountGroupProductLineAccessByID(ctx, resp.Item.AccountGroupId)
}

func (m *accountGroupProductLineAccessSvcImpl) DeleteAccountGroupProductLineAccess(ctx context.Context, req *DeleteAccountGroupProductLineAccessRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAccountGroupProductLineAccessRequest{AccountGroupId: req.AccountGroupID}
	_, apiErr := grpcutil.CallRPC(ctx, accountGroupProductLineAccessSvcTracer, "service.account_group_product_line_access.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteAccountGroupProductLineAccess(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadAccountGroupProductLineAccessByID wraps the single-ID load pattern used
// after mutations and for the retrieve endpoint. Keyed by account_group_id.
func loadAccountGroupProductLineAccessByID(ctx context.Context, accountGroupID string) (*apiresource.AccountGroupProductLineAccess, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAccountGroupProductLineAccess(ctx, []string{accountGroupID})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[accountGroupID]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Account group product line access not found.")
	}
	return v.(*apiresource.AccountGroupProductLineAccess), nil
}
