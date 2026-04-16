package accountuserep

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

type AccountUserSvc interface {
	ListAccountUsers(ctx context.Context, req *ListAccountUsersRequest) (*apiresource.List[apiresource.AccountUser], *apierror.APIError)
	GetAccountUser(ctx context.Context, req *GetAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	CreateAccountUser(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	UpdateAccountUser(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError)
	UpdateAccountUserStatus(ctx context.Context, req *UpdateAccountUserStatusRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AccountUserSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type accountUserSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var accountUserSvcTracer = tracing.GetTracer("api-gateway.endpoints.account_users.service")

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
		Cursor:         req.Cursor,
		Limit:          req.Limit,
		Query:          req.Query,
		RoleType:       req.RoleType.StringPtr(),
		IncludeRemoved: req.IncludeRemoved,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAccountUsersResponse, error) {
			return m.coreClient.ListAccountUsers(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return AccountUserListPresenter(resp), nil
}

func (m *accountUserSvcImpl) GetAccountUser(ctx context.Context, req *GetAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	pbReq := &pb.GetAccountUserRequest{AccountUserId: req.AccountUserID}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAccountUserResponse, error) {
			return m.coreClient.GetAccountUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountUserPresenter(resp.AccountUser)
	return &result, nil
}

func (m *accountUserSvcImpl) CreateAccountUser(ctx context.Context, req *CreateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	pbReq := &pb.CreateAccountUserRequest{
		Name:                    req.Name,
		Email:                   req.Email,
		Username:                req.Username,
		Password:                req.Password,
		RoleId:                  req.RoleID,
		DepartmentId:            req.DepartmentID,
		NotificationPreferences: toProtoNotificationPrefs(req.Preferences),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAccountUserResponse, error) {
			return m.coreClient.CreateAccountUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountUserPresenter(resp.AccountUser)
	return &result, nil
}

func (m *accountUserSvcImpl) UpdateAccountUser(ctx context.Context, req *UpdateAccountUserRequest) (*apiresource.AccountUser, *apierror.APIError) {
	pbReq := &pb.UpdateAccountUserRequest{
		AccountUserId:           req.AccountUserID,
		Name:                    req.Name,
		Email:                   req.Email,
		Username:                req.Username,
		RoleId:                  req.RoleID,
		DepartmentId:            req.DepartmentID,
		NotificationPreferences: toProtoNotificationPrefs(req.Preferences),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, accountUserSvcTracer, "service.account_users.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAccountUserResponse, error) {
			return m.coreClient.UpdateAccountUser(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := AccountUserPresenter(resp.AccountUser)
	return &result, nil
}

func (m *accountUserSvcImpl) UpdateAccountUserStatus(ctx context.Context, req *UpdateAccountUserStatusRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.UpdateAccountUserStatusRequest{
		AccountUserId: req.AccountUserID,
		StatusCode:    string(req.Status),
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
