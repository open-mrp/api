package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleUserID = "us_01gf7a8200e9pvbd6bgyq395ae"
const SampleUserUsername = "jdoe"
const SampleUserEmail = "jdoe@augno.com"
const SampleUserName = "John Doe"
const SampleUserImageUrl = "https://cdn.augno.com/avatars/us_01gf7a8200e9pvbd6bgyq395ae.jpg"
const SampleUserPassword = "QgS7Z8Hhj3&1"     // #nosec G101 -- sample data for API docs
const SampleNewUserPassword = "50iR2X0r@bvIH" // #nosec G101 -- sample data for API docs

// A user in the Augno system.
type User struct {
	// The unique identifier for this user.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// The user's email address.
	Email *string `json:"email"`
	// The user's display name.
	Name *string `json:"name"`
	// The user's unique username.
	Username *string `json:"username"`
	// When the user's email was verified, null if unverified.
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	// URL to the user's profile image.
	ImageUrl *string `json:"image_url"`
	// When this user was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this user was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleUser = &User{
	ID:              SampleUserID,
	Object:          constants.ObjectTypeUser,
	Username:        new(SampleUserUsername),
	Email:           new(SampleUserEmail),
	Name:            new(SampleUserName),
	ImageUrl:        new(SampleUserImageUrl),
	EmailVerifiedAt: timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*User) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUser)
}

// UserPhotoUploadResult is the response for a user photo upload.
type UserPhotoUploadResult struct {
	// Whether the upload was successful.
	Success bool `json:"success"`
}

var SampleUserPhotoUploadResult = &UserPhotoUploadResult{
	Success: true,
}

func (*UserPhotoUploadResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoUploadResult)
}

// UserPhotoURL holds a presigned URL for a user's profile photo.
type UserPhotoURL struct {
	// The presigned URL for the profile photo, or null if no photo exists.
	URL *string `json:"url"`
}

var SampleUserPhotoURL = &UserPhotoURL{}

func (*UserPhotoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoURL)
}
