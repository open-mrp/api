package httpgroup

import (
	"fmt"

	scanningstationep "github.com/augno/api/services/api-gateway/endpoints/scanning-stations"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type ScanningStationsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type ScanningStationsEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *ScanningStationsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("scanning stations endpoint group: core client is required")
	}
	return nil
}

func (*ScanningStationsEndpointGroup) Materialize(config *ScanningStationsEndpointGroupConfig) *ScanningStationsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	scanningStationSvc := scanningstationep.NewScanningStationSvc(&scanningstationep.ScanningStationSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Scanning Stations Management",
		Description:  "List and manage scanning stations.",
		ResourceType: &apiresource.ScanningStation{},
	}

	listEndpoint := (&scanningstationep.ListScanningStationsEndpoint{}).Materialize().WithService(inner, scanningStationSvc)
	retrieveEndpoint := (&scanningstationep.RetrieveScanningStationEndpoint{}).Materialize().WithService(inner, scanningStationSvc)
	createEndpoint := (&scanningstationep.CreateScanningStationEndpoint{}).Materialize().WithService(inner, scanningStationSvc)
	updateEndpoint := (&scanningstationep.UpdateScanningStationEndpoint{}).Materialize().WithService(inner, scanningStationSvc)
	deleteEndpoint := (&scanningstationep.DeleteScanningStationEndpoint{}).Materialize().WithService(inner, scanningStationSvc)
	connectStepsEndpoint := (&scanningstationep.ConnectProductionStepsEndpoint{}).Materialize().WithService(inner, scanningStationSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		connectStepsEndpoint,
	}

	return &ScanningStationsEndpointGroup{inner}
}
