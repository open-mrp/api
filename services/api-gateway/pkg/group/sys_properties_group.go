package httpgroup

import (
	"fmt"

	syspropertyep "github.com/augno/api/services/api-gateway/endpoints/sys-properties"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type SysPropertiesEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type SysPropertiesEndpointGroupConfig struct {
	CoreClient *grpcclient.CoreServiceClient
}

func (c *SysPropertiesEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("sys properties endpoint group: core client is required")
	}
	return nil
}

func (*SysPropertiesEndpointGroup) Materialize(config *SysPropertiesEndpointGroupConfig) *SysPropertiesEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := syspropertyep.NewSysPropertySvc(&syspropertyep.SysPropertySvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "System Properties Management",
		Description:  "List and manage system properties (auto-incrementing counters).",
		ResourceType: &apiresource.SysProperty{},
	}

	listEndpoint := (&syspropertyep.ListSysPropertiesEndpoint{}).Materialize().WithService(inner, svc)
	retrieveEndpoint := (&syspropertyep.RetrieveSysPropertyEndpoint{}).Materialize().WithService(inner, svc)
	updateEndpoint := (&syspropertyep.UpdateSysPropertyEndpoint{}).Materialize().WithService(inner, svc)
	getLatestValueEndpoint := (&syspropertyep.GetLatestSysPropertyValueEndpoint{}).Materialize().WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		listEndpoint,
		retrieveEndpoint,
		updateEndpoint,
		getLatestValueEndpoint,
	}

	return &SysPropertiesEndpointGroup{inner}
}
