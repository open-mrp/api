package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/apikit/include"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypePricingPlan,
		Load:       resourceloaders.LoadPricingPlans,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAccountUsageResponse,
		Load:       resourceloaders.LoadAccountUsageResponses,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeBillingPortalSessionResponse,
		Load:       resourceloaders.LoadBillingPortalSessions,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypePlanChangeProration,
		Load:       resourceloaders.LoadPlanChangeProrations,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEnterpriseInquiry,
		Load:       resourceloaders.LoadEnterpriseInquiries,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeEnsureBillingCustomerResponse,
		Load:       resourceloaders.LoadEnsureBillingCustomerResponses,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSwitchPlanResponse,
		Load:       resourceloaders.LoadSwitchPlanResponses,
	})
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeSpendingCapResponse,
		Load:       resourceloaders.LoadSpendingCapResponses,
	})
}
