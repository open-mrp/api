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

// A user's global profile, shared across every account they belong to.
//
// Account-specific settings (status, role, department) live on the account user resource that links the user to each account.
type User struct {
	// User ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user"`
	// Email address.
	Email *string `json:"email"`
	// User's full display name.
	Name *string `json:"name"`
	// Username.
	Username *string `json:"username"`
	// When the user verified their email address.
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	// URL of the user's profile image.
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
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user_photo_upload_result"`
	// Whether the photo was uploaded successfully.
	Success bool `json:"success"`
}

var SampleUserPhotoUploadResult = &UserPhotoUploadResult{
	Object:  constants.ObjectTypeUserPhotoUploadResult,
	Success: true,
}

func (*UserPhotoUploadResult) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoUploadResult)
}

// Presigned URL for a user's profile photo.
type UserPhotoURL struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=user_photo_url"`
	// Presigned URL for the profile photo.
	//
	// The URL is valid for one hour after it is issued.
	URL *string `json:"url"`
}

var SampleUserPhotoURL = &UserPhotoURL{
	Object: constants.ObjectTypeUserPhotoURL,
	URL:    new("https://cdn.augno.com/avatars/us_0151164dcaea4cbded27b50aae.jpg?X-Amz-Expires=3600&X-Amz-Signature=example"),
}

func (*UserPhotoURL) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleUserPhotoURL)
}
