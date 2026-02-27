package constants

// RegistrationLimits defines the per-plan-code caps on how many accounts can
// register. PublicLimit restricts registrations that arrive without an
// invitation token. TotalLimit caps all registrations (public + invited)
// combined. Both limits are checked against non-sandbox accounts only.
type RegistrationLimits struct {
	PublicLimit int64
	TotalLimit  int64
}

// RegistrationLimitsByPlan maps each plan code to its registration caps.
// Every public plan shares the same defaults today; the map makes it easy to
// override per-plan later without changing the enforcement code.
var RegistrationLimitsByPlan = map[PlanCode]RegistrationLimits{
	PlanCodeFree:    {PublicLimit: 10, TotalLimit: 10},
	PlanCodeStarter: {PublicLimit: 20, TotalLimit: 20},
	PlanCodePro:     {PublicLimit: 20, TotalLimit: 20},
}

// GetRegistrationLimits returns the registration limits for a plan code.
// Returns zero limits (effectively closed) for unrecognized plan codes.
func GetRegistrationLimits(planCode PlanCode) RegistrationLimits {
	if limits, ok := RegistrationLimitsByPlan[planCode]; ok {
		return limits
	}
	return RegistrationLimits{}
}
