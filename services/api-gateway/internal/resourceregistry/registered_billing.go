package resourceregistry

import (
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePricingPlan,
		Load:       resourceloaders.LoadPricingPlans,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAccountUsageResponse,
		Load:       resourceloaders.LoadAccountUsageResponses,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeBillingPortalSessionResponse,
		Load:       resourceloaders.LoadBillingPortalSessions,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePlanChangeProration,
		Load:       resourceloaders.LoadPlanChangeProrations,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEnterpriseInquiry,
		Load:       resourceloaders.LoadEnterpriseInquiries,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeEnsureBillingCustomerResponse,
		Load:       resourceloaders.LoadEnsureBillingCustomerResponses,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSwitchPlanResponse,
		Load:       resourceloaders.LoadSwitchPlanResponses,
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSpendingCapResponse,
		Load:       resourceloaders.LoadSpendingCapResponses,
	})
}
