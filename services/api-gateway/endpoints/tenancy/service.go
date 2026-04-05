package tenancyep

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
)

type TenancySvc interface {
	GetTenancy(ctx context.Context, req *GetTenancyRequest) (*apiresource.Tenancy, *apierror.APIError)
	SwitchAccount(ctx context.Context, req *SwitchAccountRequest) (*apiresource.Tenancy, *apierror.APIError)
	GetCurrentUser(ctx context.Context, req *GetCurrentUserRequest) (*apiresource.User, *apierror.APIError)
	ListCustomerAccounts(ctx context.Context, req *ListCustomerAccountsRequest) (*apiresource.List[apiresource.CustomerAccountSummary], *apierror.APIError)
}

type TenancySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type tenancySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var tenancySvcTracer = tracing.GetTracer("api-gateway.endpoints.tenancy.service")

func (c *TenancySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("tenancy endpoint service: core client is required")
	}
	return nil
}

func NewTenancySvc(config *TenancySvcConfig) TenancySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &tenancySvcImpl{coreClient: config.CoreClient}
}

func (m *tenancySvcImpl) GetTenancy(ctx context.Context, req *GetTenancyRequest) (*apiresource.Tenancy, *apierror.APIError) {
	identity, _ := appctx.GetIdentityFromContext(ctx)

	pbReq := &pb.GetTenancyRequest{}
	if identity != nil && identity.Actor != nil {
		pbReq.UserId = identity.Actor.ID
	}
	if identity != nil && identity.Target != nil && identity.Target.AccountID != "" {
		pbReq.TargetAccountId = &identity.Target.AccountID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, tenancySvcTracer, "service.tenancy.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTenancyResponse, error) {
			return m.coreClient.GetTenancy(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return TenancyPresenter(resp), nil
}

func (m *tenancySvcImpl) SwitchAccount(ctx context.Context, req *SwitchAccountRequest) (*apiresource.Tenancy, *apierror.APIError) {
	identity, _ := appctx.GetIdentityFromContext(ctx)

	pbReq := &pb.SwitchTenancyAccountRequest{
		AccountId: req.AccountID,
	}
	if identity != nil && identity.Actor != nil {
		pbReq.UserId = identity.Actor.ID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, tenancySvcTracer, "service.tenancy.switch_account", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTenancyResponse, error) {
			return m.coreClient.SwitchAccount(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return TenancyPresenter(resp), nil
}

func (m *tenancySvcImpl) GetCurrentUser(ctx context.Context, req *GetCurrentUserRequest) (*apiresource.User, *apierror.APIError) {
	identity, _ := appctx.GetIdentityFromContext(ctx)

	if apiErr := identity.CheckIsUser(); apiErr != nil {
		return nil, apiErr
	}

	pbReq := &pb.GetCurrentUserRequest{
		UserId: identity.Actor.ID,
	}
	if identity.Target != nil && identity.Target.AccountID != "" {
		pbReq.TargetAccountId = &identity.Target.AccountID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, tenancySvcTracer, "service.tenancy.get_current_user", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetCurrentUserResponse, error) {
			return m.coreClient.GetCurrentUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return CurrentUserPresenter(resp), nil
}

func (m *tenancySvcImpl) ListCustomerAccounts(ctx context.Context, req *ListCustomerAccountsRequest) (*apiresource.List[apiresource.CustomerAccountSummary], *apierror.APIError) {
	identity, _ := appctx.GetIdentityFromContext(ctx)

	pbReq := &pb.ListCustomerAccountsForUserRequest{
		VendorAccountId: req.VendorAccountID,
	}
	if identity != nil && identity.Actor != nil {
		pbReq.UserId = identity.Actor.ID
	}

	resp, apiErr := grpcutil.CallRPC(ctx, tenancySvcTracer, "service.tenancy.list_customer_accounts", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListCustomerAccountsForUserResponse, error) {
			return m.coreClient.ListCustomerAccountsForUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return CustomerAccountListPresenter(resp), nil
}
