package constants

// DashboardPath represents all routes to the Augno Dashboard.
type DashboardPath string

const (
	// DashboardPathRegisterVerify is the path where we should send a user to verify
	// their registration token.
	DashboardPathRegisterVerify DashboardPath = "/auth/register/verify"
	// DashboardPathResetPassword is the path where we should send a user to reset their password.
	DashboardPathResetPassword DashboardPath = "/auth/reset-password" // #nosec G101 - URL path, not a credential
	// DashboardPathLogin is the path where we should send a user to login.
	DashboardPathLogin DashboardPath = "/auth/login"
	// DashboardPathRegisterCheckoutReturn is the path Stripe redirects to after checkout.
	// Use with fmt.Sprintf to inject the session ID: fmt.Sprintf(path, sessionTypeID)
	DashboardPathRegisterCheckoutReturn DashboardPath = "/auth/register/%s?checkout_session_id={CHECKOUT_SESSION_ID}"
	// DashboardPathBillingPortal is the path users return to after the Stripe billing portal.
	DashboardPathBillingPortal DashboardPath = "/dashboard/account?tab=billing"
)

func (m DashboardPath) IsValid() bool {
	switch m {
	case DashboardPathRegisterVerify:
		return true
	default:
		return false
	}
}
