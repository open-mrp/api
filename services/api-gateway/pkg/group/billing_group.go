package httpgroup

import (
	"fmt"

	billingep "github.com/augno/api/services/api-gateway/endpoints/billing"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type BillingEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type BillingEndpointGroupConfig struct {
	BillingClient *grpcclient.BillingServiceClient
	CoreClient    *grpcclient.CoreServiceClient
}

func (c *BillingEndpointGroupConfig) validate() error {
	if c.BillingClient == nil {
		return fmt.Errorf("billing endpoint group: billing client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("billing endpoint group: core client is required")
	}
	return nil
}

func (*BillingEndpointGroup) Materialize(config *BillingEndpointGroupConfig) *BillingEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	billingSvc := billingep.NewBillingSvc(&billingep.BillingSvcConfig{
		BillingClient: config.BillingClient.Client,
		CoreClient:    config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Billing",
		Description:  "Billing and pricing plan operations.",
		ResourceType: &apiresource.PricingPlan{},
	}

	getPricingPlansEndpoint := (&billingep.GetPricingPlansEndpoint{}).Materialize().WithService(inner, billingSvc)
	getAccountUsageEndpoint := (&billingep.GetAccountUsageEndpoint{}).Materialize().WithService(inner, billingSvc)
	createBillingPortalSessionEndpoint := (&billingep.CreateBillingPortalSessionEndpoint{}).Materialize().WithService(inner, billingSvc)
	getPlanChangePreviewEndpoint := (&billingep.GetPlanChangePreviewEndpoint{}).Materialize().WithService(inner, billingSvc)
	createEnterpriseInquiryEndpoint := (&billingep.CreateEnterpriseInquiryEndpoint{}).Materialize().WithService(inner, billingSvc)
	ensureBillingCustomerEndpoint := (&billingep.EnsureBillingCustomerEndpoint{}).Materialize().WithService(inner, billingSvc)
	switchPlanEndpoint := (&billingep.SwitchPlanEndpoint{}).Materialize().WithService(inner, billingSvc)
	getSpendingCapEndpoint := (&billingep.GetSpendingCapEndpoint{}).Materialize().WithService(inner, billingSvc)
	setSpendingCapEndpoint := (&billingep.SetSpendingCapEndpoint{}).Materialize().WithService(inner, billingSvc)
	inner.Endpoints = []apiendpoint.APIEndpointer{
		getPricingPlansEndpoint,
		getAccountUsageEndpoint,
		createBillingPortalSessionEndpoint,
		getPlanChangePreviewEndpoint,
		createEnterpriseInquiryEndpoint,
		ensureBillingCustomerEndpoint,
		switchPlanEndpoint,
		getSpendingCapEndpoint,
		setSpendingCapEndpoint,
	}

	return &BillingEndpointGroup{inner}
}
