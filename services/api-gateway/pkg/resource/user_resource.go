package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/ptrutil"
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
	Object:        "user",
	Username:      ptrutil.String(SampleUserUsername),
	Email:         ptrutil.String(SampleUserEmail),
	Name:          ptrutil.String(SampleUserName),
	ImageUrl:      ptrutil.String(SampleUserImageUrl),
	EmailVerified: ptrutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:     ptrutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     ptrutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

// A user in the Augno system
type User struct {
	// The ID of the user
	ID string `json:"id" validate:"required"`
	// The object type, always "user"
	Object string `json:"object" validate:"required"`
	// The email of the user
	Email *string `json:"email"`
	// The name of the user
	Name *string `json:"name"`
	// The username of the user
	Username *string `json:"username"`
	// The email verified status of the user
	EmailVerified *time.Time `json:"email_verified"`
	// The image URL of the user
	ImageUrl *string `json:"image_url"`
	// The created at timestamp of the user
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The updated at timestamp of the user
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

func (*User) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUser)
}
