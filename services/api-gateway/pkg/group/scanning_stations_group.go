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
	// CoreClient (required) is the core-service gRPC client.
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
		Title:        "Scanning Stations",
		Description:  "List and manage scanning stations.",
		ResourceType: &apiresource.ScanningStation{},
	}

	listEndpoint := apiendpoint.From(&scanningstationep.ListScanningStationsEndpoint{}).WithService(inner, scanningStationSvc)
	retrieveEndpoint := apiendpoint.From(&scanningstationep.RetrieveScanningStationEndpoint{}).WithService(inner, scanningStationSvc)
	createEndpoint := apiendpoint.From(&scanningstationep.CreateScanningStationEndpoint{}).WithService(inner, scanningStationSvc)
	updateEndpoint := apiendpoint.From(&scanningstationep.UpdateScanningStationEndpoint{}).WithService(inner, scanningStationSvc)
	deleteEndpoint := apiendpoint.From(&scanningstationep.DeleteScanningStationEndpoint{}).WithService(inner, scanningStationSvc)
	connectStepsEndpoint := apiendpoint.From(&scanningstationep.ConnectProductionStepsEndpoint{}).WithService(inner, scanningStationSvc)
	bulkUpsertEndpoint := apiendpoint.From(&scanningstationep.BulkUpsertScanningStationsEndpoint{}).WithService(inner, scanningStationSvc)
	exportEndpoint := apiendpoint.From(&scanningstationep.ExportScanningStationsEndpoint{}).WithService(inner, scanningStationSvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		createEndpoint,
		updateEndpoint,
		deleteEndpoint,
		connectStepsEndpoint,
		bulkUpsertEndpoint,
		exportEndpoint,
	}

	return &ScanningStationsEndpointGroup{inner}
}
