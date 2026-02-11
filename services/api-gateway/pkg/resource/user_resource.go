package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/timeutil"
)

const SampleUserID = "us_01gf7a8200e9pvbd6bgyq395ae"
const SampleUserUsername = "jdoe"
const SampleUserEmail = "jdoe@augno.com"
const SampleUserName = "John Doe"
const SampleUserImageUrl = "https://example.com/avatar.jpg"
const SampleUserPassword = "super-secret-password"
const SampleNewUserPassword = "new-super-secret-password"

var SampleUser = &User{
	ID:            SampleUserID,
	Object:        constants.ObjectTypeUser,
	Username:      ptrutil.Ptr(SampleUserUsername),
	Email:         ptrutil.Ptr(SampleUserEmail),
	Name:          ptrutil.Ptr(SampleUserName),
	ImageUrl:      ptrutil.Ptr(SampleUserImageUrl),
	EmailVerified: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// A user in the Augno system
type User struct {
	// The unique identifier for this user
	ID string `json:"id" validate:"required"`
	// The resource type identifier
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// The user's email address
	Email *string `json:"email"`
	// The user's display name
	Name *string `json:"name"`
	// The user's unique username
	Username *string `json:"username"`
	// When the user's email was verified, null if unverified
	EmailVerified *time.Time `json:"email_verified"`
	// URL to the user's profile image
	ImageUrl *string `json:"image_url"`
	// When this user was created
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this user was last updated
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func (*User) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUser)
}
