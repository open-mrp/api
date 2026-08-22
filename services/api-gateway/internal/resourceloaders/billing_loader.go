package resourceloaders

import (
	"context"

	apierror "github.com/open-mrp/api/shared/errors"
)

func LoadPricingPlans(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPricingPlans should not be called — pricing plans are not used as expandable sub-resources",
	)
}

func LoadAccountUsageResponses(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadAccountUsageResponses should not be called — account usage responses are not used as expandable sub-resources",
	)
}

func LoadBillingPortalSessions(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadBillingPortalSessions should not be called — billing portal sessions are not used as expandable sub-resources",
	)
}

func LoadPlanChangeProrations(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadPlanChangeProrations should not be called — plan change prorations are not used as expandable sub-resources",
	)
}

func LoadEnterpriseInquiries(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadEnterpriseInquiries should not be called — enterprise inquiries are not used as expandable sub-resources",
	)
}

func LoadEnsureBillingCustomerResponses(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadEnsureBillingCustomerResponses should not be called — ensure billing customer responses are not used as expandable sub-resources",
	)
}

func LoadSwitchPlanResponses(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSwitchPlanResponses should not be called — switch plan responses are not used as expandable sub-resources",
	)
}

func LoadSpendingCapResponses(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, apierror.NewInvariantViolationError(
		"LoadSpendingCapResponses should not be called — spending cap responses are not used as expandable sub-resources",
	)
}
