package constants

// DashboardPath represents all routes to the Augno Dashboard.
type DashboardPath string

const (
	// DashboardPathRegisterVerify is the path where we should send a user to verify
	// their registration token.
	DashboardPathRegisterVerify DashboardPath = "/auth/register/verify"
)

func (m DashboardPath) IsValid() bool {
	switch m {
	case DashboardPathRegisterVerify:
		return true
	default:
		return false
	}
}
