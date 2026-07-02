package constants

// ReviewRequirement is whether a human must review/approve something before it executes. It is an enum (not a boolean) so additional requirements (e.g. conditional, first-use-only) can be added without a breaking change. It applies both to an individual agent action and to an agent's tool configuration.
type ReviewRequirement string

const (
	// ReviewRequirementNotRequired means no human review is required before execution (the default).
	ReviewRequirementNotRequired ReviewRequirement = "not_required"
	// ReviewRequirementRequired means a human must review/approve before execution.
	ReviewRequirementRequired ReviewRequirement = "required"
)

func (s ReviewRequirement) IsValid() bool {
	switch s {
	case ReviewRequirementNotRequired, ReviewRequirementRequired:
		return true
	default:
		return false
	}
}

func (s ReviewRequirement) EnumValues() []string {
	return []string{string(ReviewRequirementNotRequired), string(ReviewRequirementRequired)}
}

func (s *ReviewRequirement) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// ReviewRequirementFromBool maps the persisted boolean to its review-requirement enum.
func ReviewRequirementFromBool(required bool) ReviewRequirement {
	if required {
		return ReviewRequirementRequired
	}
	return ReviewRequirementNotRequired
}
