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
	// BillingClient (required) is the billing-service gRPC client.
	BillingClient *grpcclient.BillingServiceClient

	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
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

	getPricingPlansEndpoint := apiendpoint.From(&billingep.GetPricingPlansEndpoint{}).WithService(inner, billingSvc)
	getAccountUsageEndpoint := apiendpoint.From(&billingep.GetAccountUsageEndpoint{}).WithService(inner, billingSvc)
	createBillingPortalSessionEndpoint := apiendpoint.From(&billingep.CreateBillingPortalSessionEndpoint{}).WithService(inner, billingSvc)
	getPlanChangePreviewEndpoint := apiendpoint.From(&billingep.GetPlanChangePreviewEndpoint{}).WithService(inner, billingSvc)
	createEnterpriseInquiryEndpoint := apiendpoint.From(&billingep.CreateEnterpriseInquiryEndpoint{}).WithService(inner, billingSvc)
	ensureBillingCustomerEndpoint := apiendpoint.From(&billingep.EnsureBillingCustomerEndpoint{}).WithService(inner, billingSvc)
	switchPlanEndpoint := apiendpoint.From(&billingep.SwitchPlanEndpoint{}).WithService(inner, billingSvc)
	getSpendingCapEndpoint := apiendpoint.From(&billingep.GetSpendingCapEndpoint{}).WithService(inner, billingSvc)
	setSpendingCapEndpoint := apiendpoint.From(&billingep.SetSpendingCapEndpoint{}).WithService(inner, billingSvc)
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
