package accountuserep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountUserSvc interface {
	ListAccountUsers(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError)
	GetAccountUser(ctx context.Context, req *RetrieveAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	CreateAccountUser(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	UpdateAccountUser(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	ActivateAccountUser(ctx context.Context, req *ActivateAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError)
	DisableAccountUser(ctx context.Context, req *DisableAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError)
	RemoveAccountUser(ctx context.Context, req *RemoveAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AccountUserSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type accountUserSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountUserSvcTracer = tracing.GetTracer("api-gateway.endpoints.account_users.service")

func removedScopeIncludesRemoved(scope *constants.RemovedResourceScope) bool {
	return scope != nil && *scope == constants.RemovedResourceScopeIncluded
}

func (c *AccountUserSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("account user endpoint service: core client is required")
	}
	return nil
}

func NewAccountUserSvc(config *AccountUserSvcConfig) AccountUserSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &accountUserSvcImpl{coreClient: config.CoreClient}
}

func (m *accountUserSvcImpl) ListAccountUsers(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError) {
	pbReq := &pb.ListAccountUsersRequest{
		Cursor:               req.Cursor,
		Limit:                req.Limit,
		Query:                req.Query,
		RoleType:             req.RoleType.StringPtr(),
		IsCommissionEligible: req.IsCommissionEligible,
		IncludeRemoved:       removedScopeIncludesRemoved(req.RemovedScope),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountUsersResponse, error) {
			return m.coreClient.ListAccountUsers(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.AccountUsers))
	for i, au := range resp.AccountUsers {
		ids[i] = au.Id
	}
	loaded, apiErr := resourceloaders.LoadAccountUsers(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	users := make([]apiresource.AccountUser, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			users = append(users, *(v.(*apiresource.AccountUser)))
		}
	}
	return apiresource.NewList(users, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *accountUserSvcImpl) GetAccountUser(ctx context.Context, req *RetrieveAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	return loadAccountUserByID(ctx, req.AccountUserID)
}

func (m *accountUserSvcImpl) CreateAccountUser(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	pbReq := &pb.CreateAccountUserRequest{
		Name:                    req.Name.Ptr(),
		Email:                   req.Email.Ptr(),
		Username:                req.Username.Ptr(),
		Password:                req.Password.Ptr(),
		RoleId:                  req.RoleID.Ptr(),
		DepartmentId:            req.DepartmentID.Ptr(),
		IsCommissionEligible:    req.IsCommissionEligible.Ptr(),
		NotificationPreferences: toProtoNotificationPrefs(req.Preferences),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountUserResponse, error) {
			return m.coreClient.CreateAccountUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadAccountUserByID(ctx, resp.AccountUser.Id)
}

func (m *accountUserSvcImpl) UpdateAccountUser(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	pbReq := &pb.UpdateAccountUserRequest{
		AccountUserId:           req.AccountUserID,
		Name:                    req.Name.Ptr(),
		Email:                   req.Email.Ptr(),
		Username:                req.Username.Ptr(),
		RoleId:                  field.StringClearableToProto(req.RoleID),
		DepartmentId:            field.StringClearableToProto(req.DepartmentID),
		IsCommissionEligible:    req.IsCommissionEligible.Ptr(),
		NotificationPreferences: toProtoNotificationPrefs(req.Preferences),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountUserResponse, error) {
			return m.coreClient.UpdateAccountUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadAccountUserByID(ctx, resp.AccountUser.Id)
}

func (m *accountUserSvcImpl) ActivateAccountUser(ctx context.Context, req *ActivateAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	return m.transitionAccountUserStatus(ctx, req.AccountUserID, constants.AccountUserStatusActive)
}

func (m *accountUserSvcImpl) DisableAccountUser(ctx context.Context, req *DisableAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	return m.transitionAccountUserStatus(ctx, req.AccountUserID, constants.AccountUserStatusDisabled)
}

func (m *accountUserSvcImpl) RemoveAccountUser(ctx context.Context, req *RemoveAccountUserRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	return m.transitionAccountUserStatus(ctx, req.AccountUserID, constants.AccountUserStatusRemoved)
}

func (m *accountUserSvcImpl) transitionAccountUserStatus(ctx context.Context, accountUserID string, status constants.AccountUserStatus) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.UpdateAccountUserStatusRequest{
		AccountUserId: accountUserID,
		StatusCode:    string(status),
	}

	_, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.update_status", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.UpdateAccountUserStatus(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func loadAccountUserByID(ctx context.Context, id string) (*apiresource.AccountUser, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAccountUsers(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Account user not found.")
	}
	return v.(*apiresource.AccountUser), nil
}

func toProtoNotificationPrefs(in []NotificationPreferenceItem) []*pb.NotificationPreferenceItem {
	if len(in) == 0 {
		return nil
	}
	out := make([]*pb.NotificationPreferenceItem, len(in))
	for i, p := range in {
		out[i] = &pb.NotificationPreferenceItem{
			NotificationTypeCode: string(p.NotificationTypeCode),
			Enabled:              p.Enabled,
		}
	}
	return out
}
