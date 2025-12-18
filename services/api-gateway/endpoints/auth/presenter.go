package authep

import (
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/ptrutil"
)

func UserPresenter(user *pb.User) apiresource.User {
	if user == nil {
		return apiresource.User{}
	}

	var emailVerified *time.Time
	if user.EmailVerified != nil {
		emailVerified = ptrutil.Time(user.EmailVerified.AsTime())
	}

	createdAt := time.Time{}
	if user.CreatedAt != nil {
		createdAt = user.CreatedAt.AsTime()
	}

	updatedAt := time.Time{}
	if user.UpdatedAt != nil {
		updatedAt = user.UpdatedAt.AsTime()
	}

	return apiresource.User{
		ID:            user.Id,
		Object:        "user",
		Email:         user.Email,
		Name:          user.Name,
		Username:      user.Username,
		EmailVerified: emailVerified,
		ImageUrl:      user.ImageUrl,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
