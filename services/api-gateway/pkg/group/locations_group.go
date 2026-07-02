package httpgroup

import (
	"fmt"

	locationep "github.com/augno/api/services/api-gateway/endpoints/locations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type LocationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type LocationsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *LocationsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("locations endpoint group: core client is required")
	}
	return nil
}

func (*LocationsEndpointGroup) Materialize(config *LocationsEndpointGroupConfig) *LocationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	locationSvc := locationep.NewLocationSvc(&locationep.LocationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Location",
		Description:  "List and manage locations.",
		ResourceType: &apiresource.Location{},
	}

	listLocationsEndpoint := apiendpoint.From(&locationep.ListLocationsEndpoint{}).WithService(inner, locationSvc)
	getLocationEndpoint := apiendpoint.From(&locationep.RetrieveLocationEndpoint{}).WithService(inner, locationSvc)
	createLocationEndpoint := apiendpoint.From(&locationep.CreateLocationEndpoint{}).WithService(inner, locationSvc)
	updateLocationEndpoint := apiendpoint.From(&locationep.UpdateLocationEndpoint{}).WithService(inner, locationSvc)
	deleteLocationEndpoint := apiendpoint.From(&locationep.DeleteLocationEndpoint{}).WithService(inner, locationSvc)
	listLocationTypesEndpoint := apiendpoint.From(&locationep.ListLocationTypesEndpoint{}).WithService(inner, locationSvc)
	getLocationTypeEndpoint := apiendpoint.From(&locationep.RetrieveLocationTypeEndpoint{}).WithService(inner, locationSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listLocationsEndpoint,
		getLocationEndpoint,
		createLocationEndpoint,
		updateLocationEndpoint,
		deleteLocationEndpoint,
		listLocationTypesEndpoint,
		getLocationTypeEndpoint,
	}

	return &LocationsEndpointGroup{inner}
}
