package constants

// RoleTypeCode represents the type of role a user has.
type RoleTypeCode string

const (
	// RoleTypeCodeAdmin indicates that the user is an admin.
	RoleTypeCodeAdmin RoleTypeCode = "admin"
	// RoleTypeCodeUser indicates that the user is a user.
	RoleTypeCodeUser RoleTypeCode = "user"
	// RoleTypeCodeScanner indicates that the user is a scanner.
	RoleTypeCodeScanner RoleTypeCode = "scanner"
	// RoleTypeCodeSalesRep indicates that the user is a sales rep.
	RoleTypeCodeSalesRep RoleTypeCode = "sales_rep"
	// RoleTypeCodeAPIKey indicates that the user is an API key.
	RoleTypeCodeAPIKey RoleTypeCode = "api_key"
	// RoleTypeCodeCustomer indicates that the user is a customer.
	RoleTypeCodeCustomer RoleTypeCode = "customer"
)

func (m RoleTypeCode) IsValid() bool {
	switch m {
	case RoleTypeCodeAdmin, RoleTypeCodeUser, RoleTypeCodeScanner, RoleTypeCodeSalesRep, RoleTypeCodeAPIKey, RoleTypeCodeCustomer:
		return true
	default:
		return false
	}
}

func (m RoleTypeCode) EnumValues() []string {
	return []string{string(RoleTypeCodeAdmin), string(RoleTypeCodeUser), string(RoleTypeCodeScanner), string(RoleTypeCodeSalesRep), string(RoleTypeCodeAPIKey), string(RoleTypeCodeCustomer)}
}
