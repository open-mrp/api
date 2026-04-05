package constants

// PlanCode represents the category of a plan.
type PlanCode string

const (
	// PlanCodeFree indicates that the plan is a free plan.
	PlanCodeFree PlanCode = "free"
	// PlanCodeStarter indicates that the plan is a starter plan.
	PlanCodeStarter PlanCode = "starter"
	// PlanCodePro indicates that the plan is a pro plan.
	PlanCodePro PlanCode = "pro"
	// PlanCodeEnterprise indicates that the plan is an enterprise plan.
	PlanCodeEnterprise PlanCode = "enterprise"
	// PlanCodeEnterpriseTemplate indicates that the plan is an enterprise template plan.
	// In practice, we only use the template plan for display purposes.
	PlanCodeEnterpriseTemplate PlanCode = "enterprise_template"
)

func (p PlanCode) IsValid() bool {
	switch p {
	case PlanCodeFree, PlanCodeStarter, PlanCodePro, PlanCodeEnterprise, PlanCodeEnterpriseTemplate:
		return true
	default:
		return false
	}
}

func (p PlanCode) EnumValues() []string {
	return []string{string(PlanCodeFree), string(PlanCodeStarter), string(PlanCodePro), string(PlanCodeEnterprise), string(PlanCodeEnterpriseTemplate)}
}

// PublicPlanCode is the public facing code for a plan.
type PublicPlanCode string

const (
	// PublicPlanCodeFree indicates that the plan is a free plan.
	PublicPlanCodeFree PublicPlanCode = "free"
	// PublicPlanCodeStarter indicates that the plan is a starter plan.
	PublicPlanCodeStarter PublicPlanCode = "starter"
	// PublicPlanCodePro indicates that the plan is a pro plan.
	PublicPlanCodePro PublicPlanCode = "pro"
)

func (p PublicPlanCode) IsValid() bool {
	switch p {
	case PublicPlanCodeFree, PublicPlanCodeStarter, PublicPlanCodePro:
		return true
	default:
		return false
	}
}

func (p PublicPlanCode) EnumValues() []string {
	return []string{string(PublicPlanCodeFree), string(PublicPlanCodeStarter), string(PublicPlanCodePro)}
}
