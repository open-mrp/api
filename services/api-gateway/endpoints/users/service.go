package userep

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserSvc interface {
	GetUser(ctx context.Context, req *RetrieveUserRequest) (*apiresource.User, *apierror.APIError)
	UpdateUser(ctx context.Context, req *UpdateUserRequest) (*apiresource.User, *apierror.APIError)
	UploadUserPhoto(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError)
	GetUserPhotoURL(ctx context.Context, req *GetUserPhotoURLRequest) (*apiresource.UserPhotoURL, *apierror.APIError)
}

type UserSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type userSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var userSvcTracer = tracing.GetTracer("api-gateway.endpoints.users.service")

func (c *UserSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("user endpoint service: core client is required")
	}
	return nil
}

func NewUserSvc(config *UserSvcConfig) UserSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &userSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *userSvcImpl) GetUser(ctx context.Context, req *RetrieveUserRequest) (*apiresource.User, *apierror.APIError) {
	pbReq := &pb.GetUserRequest{
		Id: req.UserID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, userSvcTracer, "service.users.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
			return m.coreClient.GetUser(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := userFromProto(resp.User)
	return &result, nil
}

func (m *userSvcImpl) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*apiresource.User, *apierror.APIError) {
	pbReq := &pb.UpdateUserRequest{
		Id:       req.UserID,
		Name:     req.Name.Ptr(),
		ImageUrl: req.ImageUrl.Ptr(),
	}
	if v, ok := req.EmailVerified.Value(); ok {
		pbReq.EmailVerifiedAt = timestamppb.New(v)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, userSvcTracer, "service.users.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateUserResponse, error) {
			return m.coreClient.UpdateUser(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := userFromProto(resp.User)
	return &result, nil
}

func (m *userSvcImpl) UploadUserPhoto(ctx context.Context, req *UploadUserPhotoRequest) (*apiresource.UserPhotoUploadResult, *apierror.APIError) {
	pbReq := &pb.UploadUserPhotoRequest{
		Id:          req.UserID,
		File:        req.RawBody,
		ContentType: req.ContentType,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, userSvcTracer, "service.users.upload_photo", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UploadUserPhotoResponse, error) {
			return m.coreClient.UploadUserPhoto(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.UserPhotoUploadResult{
		Object:  constants.ObjectTypeUserPhotoUploadResult,
		Success: resp.Success,
	}, nil
}

func (m *userSvcImpl) GetUserPhotoURL(ctx context.Context, req *GetUserPhotoURLRequest) (*apiresource.UserPhotoURL, *apierror.APIError) {
	pbReq := &pb.GetUserPhotoURLRequest{
		Id: req.UserID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, userSvcTracer, "service.users.get_photo_url", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUserPhotoURLResponse, error) {
			return m.coreClient.GetUserPhotoURL(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.UserPhotoURL{
		Object: constants.ObjectTypeUserPhotoURL,
		URL:    resp.Url,
	}, nil
}

func userFromProto(u *pb.UserInfo) apiresource.User {
	if u == nil {
		return apiresource.User{}
	}

	user := apiresource.User{
		ID:              u.Id,
		Object:          constants.ObjectTypeUser,
		Email:           u.Email,
		Name:            u.Name,
		Username:        u.Username,
		ImageUrl:        u.ImageUrl,
		EmailVerifiedAt: grpcutil.TimestampToTimePtr(u.EmailVerifiedAt),
		CreatedAt:       grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(u.UpdatedAt),
	}

	return user
}
