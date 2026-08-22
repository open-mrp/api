package httpgroup

import (
	"fmt"

	operatingcalendarep "github.com/open-mrp/api/services/api-gateway/endpoints/operating-calendars"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type OperatingCalendarsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type OperatingCalendarsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *OperatingCalendarsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("operating calendars endpoint group: core client is required")
	}
	return nil
}

func (*OperatingCalendarsEndpointGroup) Materialize(config *OperatingCalendarsEndpointGroupConfig) *OperatingCalendarsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := operatingcalendarep.NewOperatingCalendarSvc(&operatingcalendarep.OperatingCalendarSvcConfig{
		CoreClient: config.CoreClient.ProductionSchedule,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Operating Calendars",
		Description:  "The days a plant tenders freight and a customer's dock accepts it, less the holidays and shutdowns either side is closed for. Every ship-by date is resolved against them, so an order is never committed to a day nobody can act on.",
		ResourceType: &apiresource.OperatingCalendar{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&operatingcalendarep.ListOperatingCalendarsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.RetrieveOperatingCalendarEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.CreateOperatingCalendarEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.UpdateOperatingCalendarEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.DeleteOperatingCalendarEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.ListOperatingCalendarClosuresEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.CreateOperatingCalendarClosureEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&operatingcalendarep.DeleteOperatingCalendarClosureEndpoint{}).WithService(inner, svc),
	}

	return &OperatingCalendarsEndpointGroup{inner}
}
