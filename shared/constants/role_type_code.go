package constants

// RoleType represents the type of role a user has.
type RoleType string

const (
	// RoleTypeAdmin indicates that the user is an admin.
	RoleTypeAdmin RoleType = "admin"
	// RoleTypeCustom indicates that the role was custom made to fit a particular need.
	RoleTypeCustom RoleType = "user" // ! NOTE: Should update to "custom" in DB and app code
	// RoleTypeScanner indicates that the user is a scanner.
	RoleTypeScanner RoleType = "scanner"
	// RoleTypeSalesRep indicates that the user is a sales rep.
	RoleTypeSalesRep RoleType = "sales_rep"
	// RoleTypeAgent indicates that the role is assigned to an agent.
	RoleTypeAgent RoleType = "agent"
)

func (m RoleType) IsValid() bool {
	switch m {
	case RoleTypeAdmin, RoleTypeCustom, RoleTypeScanner, RoleTypeSalesRep, RoleTypeAgent:
		return true
	default:
		return false
	}
}

func (m RoleType) EnumValues() []string {
	return []string{string(RoleTypeAdmin), string(RoleTypeCustom), string(RoleTypeScanner), string(RoleTypeSalesRep), string(RoleTypeAgent)}
}
