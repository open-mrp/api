package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SamplePermissionID = "perm_gum1nfdm75ro"
const SamplePermissionGroupID = "pg_584hkihly2mh"

// One area of the product that access can be granted for, such as customers, invoices, or production runs.
//
// A role never grants a permission outright; it grants specific actions on it, written as `{code}:{action}` — for example `customers:read`.
type Permission struct {
	// Permission ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission"`
	// Stable code identifying the area this permission controls, such as `customers` or `sales_orders`.
	//
	// Pair the code with an action (`create`, `read`, `update`, or `delete`) to form the permission strings used when creating or updating a role.
	Code string `json:"code" validate:"required"`
	// Human-readable name for the permission.
	Name string `json:"name" validate:"required"`
	// Human-readable description of what this permission controls.
	Description *string `json:"description"`
	// Code of the permission group this permission is listed under, such as `inventory`.
	PermissionGroupCode string `json:"group" validate:"required"`
	// When the permission was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the permission was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePermission = &Permission{
	ID:                  SamplePermissionID,
	Object:              constants.ObjectTypePermission,
	Code:                "customers:read",
	Name:                "Read Customers",
	Description:         nil,
	PermissionGroupCode: "customers",
	CreatedAt:           timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:           timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Permission) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePermission)
}

// A category of the permission catalog that collects related permissions, such as inventory or invoices.
//
// Groups exist to organize the catalog for display; access is always granted by the individual permissions inside a group, never by the group itself.
type PermissionGroup struct {
	// Permission group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission_group"`
	// Unique code identifying the permission group, such as `customers`.
	Code string `json:"code" validate:"required"`
	// Human-readable name for the permission group.
	Name string `json:"name" validate:"required"`
	// Free-form description of the permission group.
	Description *string `json:"description"`
	// The individual permissions collected under this group.
	Permissions *List[Permission] `json:"permissions"`
	// Provenance of this permission group.
	//
	// Permission groups form the platform-defined permission catalog and are system-owned; they are the same across every account.
	Owner *Owner `json:"owner" expandable:"true"`
	// When the permission group was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the permission group was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePermissionGroup = &PermissionGroup{
	ID:          SamplePermissionGroupID,
	Object:      constants.ObjectTypePermissionGroup,
	Code:        "customers",
	Name:        "Customers",
	Description: nil,
	Permissions: NewList([]Permission{*SamplePermission}, PageInfo{}),
	Owner:       SampleOwnerSystem,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PermissionGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePermissionGroup)
}
