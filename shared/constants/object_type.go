package constants

// ObjectType is a string that indicates what type of object a given object is.
type ObjectType string

const (
	// ObjectTypeAccount indicates that the object is an account.
	ObjectTypeAccount ObjectType = "account"
	// ObjectTypeUser indicates that the object is a user.
	ObjectTypeUser ObjectType = "user"
	// ObjectTypeAddress indicates that the object is an address.
	ObjectTypeAddress ObjectType = "address"
	// ObjectTypeAPIKey indicates that the object is an API key.
	ObjectTypeAPIKey ObjectType = "api_key"
	// ObjectTypeList indicates that the object is a list.
	ObjectTypeList ObjectType = "list"
	// ObjectTypeSandbox indicates that the object is a sandbox.
	ObjectTypeSandbox ObjectType = "sandbox"
	// ObjectTypeRegistrationSession indicates that the object is a registration session.
	ObjectTypeRegistrationSession ObjectType = "registration_session"
	// ObjectTypePricingPlan indicates that the object is a pricing plan.
	ObjectTypePricingPlan ObjectType = "pricing_plan"
	// ObjectTypePlanChange indicates that the object is a plan change.
	ObjectTypePlanChange ObjectType = "plan_change"
	// ObjectTypeEnterpriseInquiry indicates that the object is an enterprise inquiry.
	ObjectTypeEnterpriseInquiry ObjectType = "enterprise_inquiry"
	// ObjectTypeRequestLog indicates that the object is a request log.
	ObjectTypeRequestLog ObjectType = "request_log"
	// ObjectTypeRole indicates that the object is a role.
	ObjectTypeRole ObjectType = "role"
	// ObjectTypeUnit indicates that the object is a unit.
	ObjectTypeUnit ObjectType = "unit"
)

func (m ObjectType) IsValid() bool {
	switch m {
	case ObjectTypeAccount, ObjectTypeUser, ObjectTypeAddress, ObjectTypeAPIKey, ObjectTypeList, ObjectTypeSandbox, ObjectTypeRegistrationSession, ObjectTypePricingPlan, ObjectTypePlanChange, ObjectTypeEnterpriseInquiry, ObjectTypeRequestLog, ObjectTypeUnit:
		return true
	default:
		return false
	}
}

func (m ObjectType) EnumValues() []string {
	return []string{string(ObjectTypeAccount), string(ObjectTypeUser), string(ObjectTypeAddress), string(ObjectTypeAPIKey), string(ObjectTypeList), string(ObjectTypeSandbox), string(ObjectTypeRegistrationSession), string(ObjectTypePricingPlan), string(ObjectTypePlanChange), string(ObjectTypeEnterpriseInquiry), string(ObjectTypeRequestLog), string(ObjectTypeUnit)}
}
