package domain

import (
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

// AccountUserDetail is an enriched account user model with joined user, role,
// and department data. Used by the account user management endpoints.
type AccountUserDetail struct {
	ID                  string
	UserID              string
	Name                *string `audit:"name"`
	Email               *string `audit:"email"`
	Username            *string `audit:"username"`
	ImageURL            *string `audit:"image_url"`
	EmailVerified       bool    `audit:"email_verified"`
	RoleID              *string `audit:"role_id"`
	RoleName            *string `audit:"role_name"`
	RoleType            *string `audit:"role_type_code"`
	DepartmentID        *string `audit:"department_id"`
	DepartmentName      *string `audit:"department_name"`
	DepartmentCreatedAt *time.Time
	DepartmentUpdatedAt *time.Time
	StatusCode          constants.AccountUserStatus `audit:"status_code"`
	LastUsedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ListAccountUsersParams are the parameters for listing account users.
type ListAccountUsersParams struct {
	AccountID      string
	Query          *string
	Cursor         *string
	Limit          int32
	RoleType       *string
	IncludeRemoved bool
	Includes       []string
}

// ListAccountUsersResult is the result of listing account users.
type ListAccountUsersResult struct {
	Items      []*AccountUserDetail
	PageInfo   pagination.PageInfo
	TotalCount int64
}

// CreateAccountUserParams are the parameters for creating an account user.
type CreateAccountUserParams struct {
	AccountID               string
	Name                    *string
	Email                   *string
	Username                *string
	Password                *string // #nosec G117 -- domain model field, not a hardcoded credential
	RoleID                  *string
	DepartmentID            *string
	NotificationPreferences []NotificationPreferenceItem
}

// NotificationPreference represents a stored notification preference.
type NotificationPreference struct {
	ID                   string
	NotificationTypeCode string
}

// NotificationPreferenceItem represents a single preference toggle.
type NotificationPreferenceItem struct {
	NotificationTypeCode string
	Enabled              bool
}

// UpdateAccountUserParams are the parameters for updating an account user.
// NotificationPreferences: nil means "do not touch"; a non-nil (possibly empty)
// slice applies the provided toggles.
type UpdateAccountUserParams struct {
	AccountID               string
	AccountUserID           string
	Name                    *string
	Email                   *string
	Username                *string
	RoleID                  *string
	ClearRoleID             bool
	DepartmentID            *string
	ClearDepartmentID       bool
	NotificationPreferences []NotificationPreferenceItem
}
