package resourceloaders

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var userLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.user")

func LoadUsers(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, userLoaderTracer, "loader.users.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetUsersByIDsResponse, error) {
			return coreClient.BatchGetUsersByIDs(ctx, &pb.BatchGetUsersByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	out := make(map[string]any, len(resp.Users))
	for _, u := range resp.Users {
		out[u.Id] = userFromProto(u)
	}
	return out, nil
}

func userFromProto(u *pb.UserInfo) *apiresource.User {
	return &apiresource.User{
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
}
