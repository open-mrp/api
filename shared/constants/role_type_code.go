package constants

// RoleTypeCode represents the type of role a user has.
type RoleTypeCode string

const (
	// RoleTypeCodeAdmin indicates that the user is an admin.
	RoleTypeCodeAdmin RoleTypeCode = "admin"
	// RoleTypeCodeCustom indicates that the role was custom made to fit a particular need.
	RoleTypeCodeCustom RoleTypeCode = "user" // ! NOTE: Should update to "custom" in DB and app code
	// RoleTypeCodeScanner indicates that the user is a scanner.
	RoleTypeCodeScanner RoleTypeCode = "scanner"
	// RoleTypeCodeSalesRep indicates that the user is a sales rep.
	RoleTypeCodeSalesRep RoleTypeCode = "sales_rep"
	// RoleTypeCodeAgent indicates that the role is assigned to an agent.
	RoleTypeCodeAgent RoleTypeCode = "agent"
)

func (m RoleTypeCode) IsValid() bool {
	switch m {
	case RoleTypeCodeAdmin, RoleTypeCodeCustom, RoleTypeCodeScanner, RoleTypeCodeSalesRep, RoleTypeCodeAgent:
		return true
	default:
		return false
	}
}

func (m RoleTypeCode) EnumValues() []string {
	return []string{string(RoleTypeCodeAdmin), string(RoleTypeCodeCustom), string(RoleTypeCodeScanner), string(RoleTypeCodeSalesRep), string(RoleTypeCodeAgent)}
}
