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
	HashedPassword *string `json:"-"`
	EmailVerified  *time.Time
	ImageUrl       *string
	StatusCode     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusDeleted  UserStatus = "deleted"
)

func (s UserStatus) String() string {
	return string(s)
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
