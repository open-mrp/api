package constants

// The global "Customer" role assigned to customer-portal users.
//
// It is a single global role (role.account_id IS NULL) of role_type "user",
// resolved by its fixed ID rather than by type (multiple roles share the "user"
// type). Its permissions define what a customer may do on the portal. The
// customer registration flow references these constants; the row itself is
// mirrored in shared/db/seed/0004_auth.sql for local/test.
const (
	// GlobalCustomerRoleID is the fixed ID of the global Customer role.
	GlobalCustomerRoleID = "rl_7vafmsquekgt"
	// GlobalCustomerRoleName is the display name of the global Customer role.
	GlobalCustomerRoleName = "Customer"
)
