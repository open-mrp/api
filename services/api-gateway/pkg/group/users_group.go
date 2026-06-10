package httpgroup

import (
	"fmt"

	userep "github.com/augno/api/services/api-gateway/endpoints/users"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type UsersEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type UsersEndpointGroupConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient *grpcclient.CoreServiceClient
}

func (c *UsersEndpointGroupConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("users endpoint group: core client is required")
	}
	return nil
}

func (*UsersEndpointGroup) Materialize(config *UsersEndpointGroupConfig) *UsersEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	svc := userep.NewUserSvc(&userep.UserSvcConfig{
		CoreClient: config.CoreClient.Client,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "Users",
		Description:  "Retrieve and manage user profiles.",
		ResourceType: &apiresource.User{},
	}

	retrieveEndpoint := apiendpoint.From(&userep.RetrieveUserEndpoint{}).WithService(inner, svc)
	updateEndpoint := apiendpoint.From(&userep.UpdateUserEndpoint{}).WithService(inner, svc)
	uploadPhotoEndpoint := apiendpoint.From(&userep.UploadUserPhotoEndpoint{}).WithService(inner, svc)
	getPhotoEndpoint := apiendpoint.From(&userep.GetUserPhotoURLEndpoint{}).WithService(inner, svc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		retrieveEndpoint,
		updateEndpoint,
		uploadPhotoEndpoint,
		getPhotoEndpoint,
	}

	return &UsersEndpointGroup{inner}
}
