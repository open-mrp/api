package httpgroup

import (
	"fmt"

	apikeyep "github.com/augno/api/services/api-gateway/endpoints/api-keys"
	grpcclient "github.com/augno/api/services/api-gateway/grpc-client"
	"github.com/augno/api/services/api-gateway/internal/middleware"
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
)

type APIKeysEndpointGroup struct {
	*apiendpoint.APIEndpointGroup
}

type APIKeysEndpointGroupConfig struct {
	AuthClient *grpcclient.AuthServiceClient
	CoreClient *grpcclient.CoreServiceClient
}

func (c *APIKeysEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("api keys endpoint group: auth client is required")
	}
	if c.CoreClient == nil {
		return fmt.Errorf("api keys endpoint group: core client is required")
	}
	return nil
}

func (*APIKeysEndpointGroup) Materialize(config *APIKeysEndpointGroupConfig) *APIKeysEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	apiKeySvc := apikeyep.NewAPIKeySvc(&apikeyep.APIKeySvcConfig{
		AuthClient: config.AuthClient.Client,
		CoreClient: config.CoreClient.Client,
	})

	authMw := middleware.AuthMiddleware(&middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "API Key Management",
		Description:  "Create and manage API keys for programmatic access.",
		ResourceType: &apiresource.APIKey{},
	}

	getAPIKeyEndpoint := apiendpoint.From(&apikeyep.RetrieveAPIKeyEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)
	listAPIKeysEndpoint := apiendpoint.From(&apikeyep.ListAPIKeysEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)
	createAPIKeyEndpoint := apiendpoint.From(&apikeyep.CreateAPIKeyEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)
	rotateAPIKeyEndpoint := apiendpoint.From(&apikeyep.RotateAPIKeyEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)
	revokeAPIKeyEndpoint := apiendpoint.From(&apikeyep.RevokeAPIKeyEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)
	getDocAPIKeyEndpoint := apiendpoint.From(&apikeyep.GetDocAPIKeyEndpoint{}).WithMiddleware(authMw).WithService(inner, apiKeySvc)

	inner.Endpoints = []apiendpoint.APIEndpointer{
		getAPIKeyEndpoint,
		listAPIKeysEndpoint,
		createAPIKeyEndpoint,
		rotateAPIKeyEndpoint,
		revokeAPIKeyEndpoint,
		getDocAPIKeyEndpoint,
	}

	return &APIKeysEndpointGroup{inner}
}
