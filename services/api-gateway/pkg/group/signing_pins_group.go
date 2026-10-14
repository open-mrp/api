package httpgroup

import (
	"fmt"

	signingpinep "github.com/open-mrp/api/services/api-gateway/endpoints/signing-pins"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
)

type SigningPinsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SigningPinsEndpointGroupConfig struct {
	// AuthClient (required) is the auth-service gRPC client.
	AuthClient *grpcclient.AuthServiceClient
}

func (c *SigningPinsEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("signing pins endpoint group: auth client is required")
	}
	return nil
}

func (*SigningPinsEndpointGroup) Materialize(config *SigningPinsEndpointGroupConfig) *SigningPinsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := signingpinep.NewSigningPinSvc(&signingpinep.SigningPinSvcConfig{
		AuthClient: config.AuthClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:       "Signing PINs",
		Description: "Set, change and clear the PINs people use to sign batch releases and check in at scanning stations.",
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&signingpinep.SetSigningPinEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&signingpinep.SetTemporarySigningPinEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&signingpinep.ClearSigningPinEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&signingpinep.ChangeSigningPinEndpoint{}).WithService(inner, svc),
	}

	return &SigningPinsEndpointGroup{inner}
}
