package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleUserID = "us_0151164dcaea4cbded27b50aae"
const SampleUserUsername = "jdoe"
const SampleUserEmail = "jdoe@augno.com"
const SampleUserName = "John Doe"
const SampleUserImageUrl = "https://cdn.augno.com/avatars/us_0151164dcaea4cbded27b50aae.jpg"
const SampleUserPassword = "QgS7Z8Hhj3&1"     // #nosec G101 -- sample data for API docs
const SampleNewUserPassword = "50iR2X0r@bvIH" // #nosec G101 -- sample data for API docs

// User resource.
type User struct {
	// User ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// Email address.
	Email *string `json:"email"`
	// Display name.
	Name *string `json:"name"`
	// Username.
	Username *string `json:"username"`
	// Email verified timestamp, null if unverified.
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	// Profile image URL.
	ImageUrl *string `json:"image_url"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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

// Result of a user photo upload.
type UserPhotoUploadResult struct {
	// Upload success status.
	Success bool `json:"success"`
}

var SampleUserPhotoUploadResult = &UserPhotoUploadResult{
	Success: true,
}

func (*UserPhotoUploadResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoUploadResult)
}

// Presigned URL for a user's profile photo.
type UserPhotoURL struct {
	// Presigned URL for the profile photo, or null if no photo exists.
	URL *string `json:"url"`
}

var SampleUserPhotoURL = &UserPhotoURL{}

func (*UserPhotoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoURL)
}
