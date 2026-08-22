package httpgroup

import (
	"fmt"

	paymenttermep "github.com/open-mrp/api/services/api-gateway/endpoints/payment-terms"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type PaymentTermsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PaymentTermsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *PaymentTermsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("payment terms endpoint group: core client is required")
	}
	return nil
}

func (*PaymentTermsEndpointGroup) Materialize(config *PaymentTermsEndpointGroupConfig) *PaymentTermsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	paymentTermSvc := paymenttermep.NewPaymentTermSvc(&paymenttermep.PaymentTermSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Payment Terms",
		Description:  "List and manage payment terms.",
		ResourceType: &apiresource.PaymentTerm{},
	}

	listPaymentTermsEndpoint := apiendpoint.From(&paymenttermep.ListPaymentTermsEndpoint{}).WithService(inner, paymentTermSvc)
	getPaymentTermEndpoint := apiendpoint.From(&paymenttermep.RetrievePaymentTermEndpoint{}).WithService(inner, paymentTermSvc)
	createPaymentTermEndpoint := apiendpoint.From(&paymenttermep.CreatePaymentTermEndpoint{}).WithService(inner, paymentTermSvc)
	updatePaymentTermEndpoint := apiendpoint.From(&paymenttermep.UpdatePaymentTermEndpoint{}).WithService(inner, paymentTermSvc)
	deletePaymentTermEndpoint := apiendpoint.From(&paymenttermep.DeletePaymentTermEndpoint{}).WithService(inner, paymentTermSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPaymentTermsEndpoint,
		getPaymentTermEndpoint,
		createPaymentTermEndpoint,
		updatePaymentTermEndpoint,
		deletePaymentTermEndpoint,
	}

	return &PaymentTermsEndpointGroup{inner}
}
