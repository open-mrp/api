package httpgroup

import (
	"fmt"

	documentsettingep "github.com/open-mrp/api/services/api-gateway/endpoints/document-settings"
	grpcclient "github.com/open-mrp/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
)

type DocumentSettingsEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type DocumentSettingsEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *DocumentSettingsEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("document settings endpoint group: core client is required")
	}
	return nil
}

func (*DocumentSettingsEndpointGroup) Materialize(config *DocumentSettingsEndpointGroupConfig) *DocumentSettingsEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := documentsettingep.NewDocumentSettingSvc(&documentsettingep.DocumentSettingSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Document Settings",
		Description:  "Your account's customizations to the documents it generates, such as the document-control block printed on batch travelers.",
		ResourceType: &apiresource.DocumentSetting{},
	}

	inner.Endpoints = []apiendpoint.APIEndpointer{
		apiendpoint.From(&documentsettingep.ListDocumentSettingsEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&documentsettingep.RetrieveDocumentSettingEndpoint{}).WithService(inner, svc),
		apiendpoint.From(&documentsettingep.UpdateDocumentSettingEndpoint{}).WithService(inner, svc),
	}

	return &DocumentSettingsEndpointGroup{inner}
}
