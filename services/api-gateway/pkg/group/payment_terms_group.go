package httpgroup

import (
	"fmt"

	paymenttermep "github.com/augno/api/services/api-gateway/endpoints/payment-terms"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type PaymentTermsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type PaymentTermsEndpointGroupConfig struct {
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
		Title:        "Payment Terms Management",
		Description:  "List and manage payment terms.",
		ResourceType: &apiresource.PaymentTerm{},
	}

	listPaymentTermsEndpoint := (&paymenttermep.ListPaymentTermsEndpoint{}).Materialize().WithService(inner, paymentTermSvc)
	getPaymentTermEndpoint := (&paymenttermep.RetrievePaymentTermEndpoint{}).Materialize().WithService(inner, paymentTermSvc)
	createPaymentTermEndpoint := (&paymenttermep.CreatePaymentTermEndpoint{}).Materialize().WithService(inner, paymentTermSvc)
	updatePaymentTermEndpoint := (&paymenttermep.UpdatePaymentTermEndpoint{}).Materialize().WithService(inner, paymentTermSvc)
	deletePaymentTermEndpoint := (&paymenttermep.DeletePaymentTermEndpoint{}).Materialize().WithService(inner, paymentTermSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listPaymentTermsEndpoint,
		getPaymentTermEndpoint,
		createPaymentTermEndpoint,
		updatePaymentTermEndpoint,
		deletePaymentTermEndpoint,
	}

	return &PaymentTermsEndpointGroup{inner}
}
