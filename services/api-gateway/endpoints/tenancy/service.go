package tenancyep

import (
	"context"
	"fmt"

	"maps"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
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

	return tenancyFromProto(resp), nil
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

	return tenancyFromProto(resp), nil
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

	return currentUserFromProto(resp), nil
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

	return customerAccountListFromProto(resp), nil
}

func tenancyAccountPlanFromProto(p *pb.TenancyAccountPlanProto) *apiresource.TenancyAccountPlan {
	if p == nil {
		return nil
	}

	limits := make(map[string]*int32, len(p.Limits))
	for _, entry := range p.Limits {
		if entry == nil {
			continue
		}
		if entry.Value != nil {
			v := *entry.Value
			limits[entry.Key] = &v
		} else {
			limits[entry.Key] = nil
		}
	}

	features := make(map[string]bool, len(p.Features))
	maps.Copy(features, p.Features)

	return &apiresource.TenancyAccountPlan{
		TypeID:        p.TypeId,
		Object:        constants.ObjectTypeAccountPlan,
		Name:          p.Name,
		PlanTypeCode:  p.PlanTypeCode,
		Version:       p.Version,
		PricePerSeat:  p.PricePerSeat,
		PricePerMonth: p.PricePerMonth,
		SeatMinimum:   p.SeatMinimum,
		Limits:        limits,
		Features:      features,
	}
}

func tenancyPendingRegistrationFromProto(pr *pb.TenancyPendingRegistrationProto) *apiresource.TenancyPendingRegistration {
	if pr == nil {
		return nil
	}
	return &apiresource.TenancyPendingRegistration{
		SessionID: pr.SessionId,
		PlanCode:  pr.PlanCode,
		Step:      pr.Step,
		CreatedAt: grpcutil.TimestampToTime(pr.CreatedAt),
	}
}

func tenancyFromProto(resp *pb.GetTenancyResponse) *apiresource.Tenancy {
	if resp == nil {
		return &apiresource.Tenancy{
			Object:        constants.ObjectTypeTenancy,
			Sandboxes:     apiresource.NewList([]apiresource.TenancySandboxAccount{}, apiresource.PageInfo{}),
			OtherAccounts: apiresource.NewList([]apiresource.TenancyOtherAccount{}, apiresource.PageInfo{}),
		}
	}

	result := &apiresource.Tenancy{
		Object:              constants.ObjectTypeTenancy,
		PendingRegistration: tenancyPendingRegistrationFromProto(resp.PendingRegistration),
	}

	if resp.CurrentAccount != nil {
		ca := &apiresource.TenancyCurrentAccount{
			ID:                       resp.CurrentAccount.Id,
			Object:                   constants.ObjectTypeAccount,
			Name:                     resp.CurrentAccount.Name,
			Type:                     resp.CurrentAccount.Type,
			OnboardingStatus:         resp.CurrentAccount.OnboardingStatus,
			Plan:                     resp.CurrentAccount.PlanCode,
			Slug:                     resp.CurrentAccount.Slug,
			InternalStripeCustomerID: resp.CurrentAccount.InternalStripeCustomerId,
			AccountPlan:              tenancyAccountPlanFromProto(resp.CurrentAccount.AccountPlan),
		}

		if resp.CurrentAccount.Role != nil {
			accountID := resp.CurrentAccount.Id
			permissions := resp.CurrentAccount.Role.Permissions
			ca.Role = &apiresource.Role{
				ID:          resp.CurrentAccount.Role.Id,
				Object:      constants.ObjectTypeRole,
				Name:        resp.CurrentAccount.Role.Name,
				TypeCode:    constants.RoleType(resp.CurrentAccount.Role.RoleTypeCode),
				Owner:       apiresource.NewOwner(&accountID),
				Permissions: &permissions,
				CreatedAt:   grpcutil.TimestampToTime(resp.CurrentAccount.Role.CreatedAt),
				UpdatedAt:   grpcutil.TimestampToTime(resp.CurrentAccount.Role.UpdatedAt),
			}
		}

		result.CurrentAccount = ca
	}

	sandboxes := make([]apiresource.TenancySandboxAccount, 0, len(resp.Sandboxes))
	for _, s := range resp.Sandboxes {
		sandboxes = append(sandboxes, apiresource.TenancySandboxAccount{
			ID:     s.Id,
			Object: constants.ObjectTypeAccount,
			Name:   s.Name,
		})
	}
	result.Sandboxes = apiresource.NewList(sandboxes, apiresource.PageInfo{})

	if resp.OwnerAccount != nil {
		result.OwnerAccount = &apiresource.TenancyOwnerAccount{
			ID:     resp.OwnerAccount.Id,
			Object: constants.ObjectTypeAccount,
			Name:   resp.OwnerAccount.Name,
		}
	}

	otherAccounts := make([]apiresource.TenancyOtherAccount, 0, len(resp.OtherAccounts))
	for _, o := range resp.OtherAccounts {
		otherAccounts = append(otherAccounts, apiresource.TenancyOtherAccount{
			ID:     o.Id,
			Object: constants.ObjectTypeAccount,
			Name:   o.Name,
			Type:   o.Type,
		})
	}
	result.OtherAccounts = apiresource.NewList(otherAccounts, apiresource.PageInfo{})

	return result
}

func currentUserFromProto(resp *pb.GetCurrentUserResponse) *apiresource.User {
	if resp == nil {
		return &apiresource.User{}
	}

	return &apiresource.User{
		ID:              resp.Id,
		Object:          constants.ObjectTypeUser,
		Email:           resp.Email,
		Name:            resp.Name,
		Username:        resp.Username,
		EmailVerifiedAt: grpcutil.TimestampToTimePtr(resp.EmailVerifiedAt),
		ImageUrl:        resp.ImageUrl,
		CreatedAt:       grpcutil.TimestampToTime(resp.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(resp.UpdatedAt),
	}
}

func customerAccountListFromProto(resp *pb.ListCustomerAccountsForUserResponse) *apiresource.List[apiresource.CustomerAccountSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.CustomerAccountSummary](nil, apiresource.PageInfo{})
	}

	accounts := make([]apiresource.CustomerAccountSummary, len(resp.Accounts))
	for i, a := range resp.Accounts {
		accounts[i] = apiresource.CustomerAccountSummary{
			ID:     a.Id,
			Object: constants.ObjectTypeAccount,
			Name:   a.Name,
		}
	}

	return apiresource.NewList(accounts, apiresource.PageInfo{})
}
