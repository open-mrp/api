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
		Title:        "Location Management",
		Description:  "List and manage locations.",
		ResourceType: &apiresource.Location{},
	}

	listLocationsEndpoint := (&locationep.ListLocationsEndpoint{}).Materialize().WithService(inner, locationSvc)
	getLocationEndpoint := (&locationep.GetLocationEndpoint{}).Materialize().WithService(inner, locationSvc)
	createLocationEndpoint := (&locationep.CreateLocationEndpoint{}).Materialize().WithService(inner, locationSvc)
	updateLocationEndpoint := (&locationep.UpdateLocationEndpoint{}).Materialize().WithService(inner, locationSvc)
	deleteLocationEndpoint := (&locationep.DeleteLocationEndpoint{}).Materialize().WithService(inner, locationSvc)
	listLocationTypesEndpoint := (&locationep.ListLocationTypesEndpoint{}).Materialize().WithService(inner, locationSvc)
	getLocationTypeEndpoint := (&locationep.GetLocationTypeEndpoint{}).Materialize().WithService(inner, locationSvc)

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
