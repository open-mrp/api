package authep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUserPresenter(t *testing.T) {
	t.Parallel()
	email := "test@example.com"
	name := "Test User"
	username := "testuser"
	imageURL := "https://example.com/avatar.jpg"

	user := &pb.User{
		Id:            "us_01abc",
		Email:         &email,
		Name:          &name,
		Username:      &username,
		ImageUrl:      &imageURL,
		EmailVerified: timestamppb.Now(),
		CreatedAt:     timestamppb.Now(),
		UpdatedAt:     timestamppb.Now(),
	}

	result := UserPresenter(user)
	resourcetest.ValidateResourceStruct(t, "User", result)
}
