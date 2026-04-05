package authep

import (
	"github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/auth"
)

// UserPresenter presents a user resource from a proto user.
func UserPresenter(user *pb.User) apiresource.User {
	if user == nil {
		return apiresource.User{}
	}

	return apiresource.User{
		ID:              user.Id,
		Object:          constants.ObjectTypeUser,
		Email:           user.Email,
		Name:            user.Name,
		Username:        user.Username,
		EmailVerifiedAt: grpc.TimestampToTimePtr(user.EmailVerified),
		ImageUrl:        user.ImageUrl,
		CreatedAt:       grpc.TimestampToTime(user.CreatedAt),
		UpdatedAt:       grpc.TimestampToTime(user.UpdatedAt),
	}
}
