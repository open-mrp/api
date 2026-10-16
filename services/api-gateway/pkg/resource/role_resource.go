package apiresource

import (
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

// What a set of people, devices, API keys or agents may do.
//
// A role sets an access level for each area, then adds and removes single permissions on top with `grants` and `revokes`. `permissions` is what it holds in the end. Changes apply to everyone with the role from their next request.
type Role struct {
	// Role ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role"`
	// The role's name, unique within the account.
	Name string `json:"name" validate:"required"`
	// What kind of role this is.
	//
	// - `admin`: everything, including API keys, billing and integrations, which only admins can manage.
	// - `user`: a role your account defines. Roles created through the API have this type.
	// - `scanner`: scanning-station logins.
	// - `sales_rep`: sales reps, whose order analytics cover their own orders.
	// - `agent`: an agent's role.
	TypeCode constants.RoleType `json:"type" validate:"required"`
	// The template the role started from, or `null`. Later changes to the template don't reach the role.
	Template *constants.RoleTemplate `json:"template"`
	// The role's access level in each area.
	Access RoleAccess `json:"access" validate:"required"`
	// Permissions the role holds beyond its access levels.
	Grants []string `json:"grants" validate:"required"`
	// Permissions the role's access levels include but it doesn't hold.
	Revokes []string `json:"revokes" validate:"required"`
	// Every permission the role holds, as `{permission}:{action}` codes such as `customers:read`.
	Permissions *[]string `json:"permissions" expandable:"true"`
	// The notification settings people with this role start from, before their own preferences and after the account's defaults.
	NotificationDefaults NotificationSettings `json:"notification_defaults" validate:"required"`
	// Who owns the role: OpenMRP for the shared system roles, which can't be changed, or your account.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

// A role's access level in each area.
//
// `read` lists and views; `edit` also creates and updates; `full` also deletes and takes the area's sensitive actions; `none` gives nothing beyond what every member holds.
type RoleAccess struct {
	// Items, categories, product lines, properties and units.
	Catalog constants.AccessLevel `json:"catalog" validate:"required"`
	// Customers, contacts, sales orders, pricing and sales reps.
	Sales constants.AccessLevel `json:"sales" validate:"required"`
	// Suppliers, purchase orders and receiving.
	Purchasing constants.AccessLevel `json:"purchasing" validate:"required"`
	// Inventory, locations, picks, shipments and carriers.
	Inventory constants.AccessLevel `json:"inventory" validate:"required"`
	// Production, batches, scanning, work centers and machines.
	Production constants.AccessLevel `json:"production" validate:"required"`
	// Invoices, payments, payment terms and costs.
	Finance constants.AccessLevel `json:"finance" validate:"required"`
	// Your team, roles, API keys, integrations, logs and account settings.
	Settings constants.AccessLevel `json:"settings" validate:"required"`
}

// RoleAccessOf is the response shape of a role's access levels.
func RoleAccessOf(a types.RoleAccess) RoleAccess {
	return RoleAccess{
		Catalog:    a.Level(constants.PermissionAreaCatalog),
		Sales:      a.Level(constants.PermissionAreaSales),
		Purchasing: a.Level(constants.PermissionAreaPurchasing),
		Inventory:  a.Level(constants.PermissionAreaInventory),
		Production: a.Level(constants.PermissionAreaProduction),
		Finance:    a.Level(constants.PermissionAreaFinance),
		Settings:   a.Level(constants.PermissionAreaSettings),
	}
}

const SampleRoleID = "rl_3xknmfqflhvb"
const SampleRoleName = "Warehouse"

var SampleRolePermissions = []string{"inventory:create", "inventory:read", "inventory:update", "items:read", "receiving:create"}

var SampleRole = &Role{
	ID:       SampleRoleID,
	Name:     SampleRoleName,
	Object:   constants.ObjectTypeRole,
	TypeCode: constants.RoleTypeCustom,
	Template: new(constants.RoleTemplateWarehouse),
	Access: RoleAccess{
		Catalog: constants.AccessLevelRead, Sales: constants.AccessLevelRead, Purchasing: constants.AccessLevelRead,
		Inventory: constants.AccessLevelEdit, Production: constants.AccessLevelNone, Finance: constants.AccessLevelNone,
		Settings: constants.AccessLevelNone,
	},
	Grants:               []string{"receiving:create"},
	Revokes:              []string{"inventory:update"},
	NotificationDefaults: SampleNotificationSettings,
	Owner:                SampleOwnerAccount,
	Permissions:          &SampleRolePermissions,
	CreatedAt:            timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:            timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Role) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRole)
}

// One permission in the catalog.
//
// Every permission belongs to one area and level: a role holds it when its access in that area is at least that level, or when it grants it. Permissions at level `none` are held by every member unless revoked.
type Permission struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=permission"`
	// The permission's code, `{permission}:{action}`, as roles list it in `grants`, `revokes` and `permissions`.
	Code string `json:"code" validate:"required"`
	// A short name for the permission.
	Name string `json:"name" validate:"required"`
	// What the permission allows.
	Description string `json:"description" validate:"required"`
	// The area the permission belongs to.
	Area constants.PermissionArea `json:"area" validate:"required"`
	// The lowest access level in its area that includes the permission.
	Level constants.AccessLevel `json:"level" validate:"required"`
}

// PermissionOf is the response shape of a catalog permission.
func PermissionOf(p types.CatalogPermission) Permission {
	return Permission{Object: constants.ObjectTypePermission, Code: p.Code, Name: p.Name, Description: p.Description, Area: p.Area, Level: p.Level}
}

var SamplePermission = &Permission{
	Object: constants.ObjectTypePermission, Code: "sales_orders:read", Name: "Sales orders: read",
	Description: "List and view sales orders.", Area: constants.PermissionAreaSales, Level: constants.AccessLevelRead,
}

func (*Permission) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePermission)
}

// A starting point for a role.
//
// Creating a role with a `template` copies its access and grants; the role doesn't follow later changes to the template.
type RoleTemplate struct {
	// The template's code, which a role's `template` names.
	ID constants.RoleTemplate `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=role_template"`
	// The template's name.
	Name string `json:"name" validate:"required"`
	// Who the template is for.
	Description string `json:"description" validate:"required"`
	// The access levels a role from this template starts with.
	Access RoleAccess `json:"access" validate:"required"`
	// The permissions a role from this template grants beyond its access levels.
	Grants []string `json:"grants" validate:"required"`
}

// RoleTemplateOf is the response shape of a role template.
func RoleTemplateOf(t types.RoleTemplateSpec) RoleTemplate {
	grants := t.Grants
	if grants == nil {
		grants = []string{}
	}
	return RoleTemplate{ID: t.Code, Object: constants.ObjectTypeRoleTemplate, Name: t.Name, Description: t.Description, Access: RoleAccessOf(t.Access), Grants: grants}
}

var SampleRoleTemplate = &RoleTemplate{
	ID: constants.RoleTemplateWarehouse, Object: constants.ObjectTypeRoleTemplate, Name: "Warehouse",
	Description: "Moves stock, picks, packs and ships, and receives deliveries.",
	Access:      SampleRole.Access,
	Grants:      []string{"receiving:create", "receiving:update"},
}

func (*RoleTemplate) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleRoleTemplate)
}
