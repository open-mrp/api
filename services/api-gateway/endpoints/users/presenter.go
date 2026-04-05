package userep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func UserPresenter(u *pb.UserInfo) apiresource.User {
	if u == nil {
		return apiresource.User{}
	}

	user := apiresource.User{
		ID:        u.Id,
		Object:    constants.ObjectTypeUser,
		CreatedAt: grpcutil.TimestampToTime(u.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(u.UpdatedAt),
	}

	if u.Email != nil {
		user.Email = u.Email
	}
	if u.Name != nil {
		user.Name = u.Name
	}
	if u.Username != nil {
		user.Username = u.Username
	}
	if u.ImageUrl != nil {
		user.ImageUrl = u.ImageUrl
	}
	user.EmailVerifiedAt = grpcutil.TimestampToTimePtr(u.EmailVerifiedAt)

	return user
}
