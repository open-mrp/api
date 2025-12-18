package types

import (
	"time"

	pb "github.com/augno/api/shared/proto/auth"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type User struct {
	ID             string
	Email          *string
	Name           *string
	Username       *string
	HashedPassword *string
	EmailVerified  *time.Time
	ImageUrl       *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (u *User) ToProto() *pb.User {
	if u == nil {
		return nil
	}

	var emailVerified *timestamppb.Timestamp
	if u.EmailVerified != nil {
		emailVerified = timestamppb.New(*u.EmailVerified)
	}

	return &pb.User{
		Id:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Username:      u.Username,
		ImageUrl:      u.ImageUrl,
		EmailVerified: emailVerified,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		UpdatedAt:     timestamppb.New(u.UpdatedAt),
	}
}
