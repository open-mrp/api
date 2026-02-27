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
}

func (c *APIKeysEndpointGroupConfig) validate() error {
	if c.AuthClient == nil {
		return fmt.Errorf("api keys endpoint group: auth client is required")
	}
	return nil
}

func (*APIKeysEndpointGroup) Materialize(config *APIKeysEndpointGroupConfig) *APIKeysEndpointGroup {
	if err := config.validate(); err != nil {
		panic(err)
	}

	apiKeySvc := apikeyep.NewAPIKeySvc(&apikeyep.APIKeySvcConfig{
		AuthClient: config.AuthClient.Client,
	})

	authMw := middleware.AuthMiddleware(&middleware.AuthMiddlewareConfig{
		AuthClient: config.AuthClient,
	})

	inner := &apiendpoint.APIEndpointGroup{
		Title:        "API Key Management",
		Description:  "Handles creating and managing API keys for programmatic access.",
		ResourceType: &apiresource.APIKey{},
	}

	getAPIKeyEndpoint := (&apikeyep.GetAPIKeyEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)
	listAPIKeysEndpoint := (&apikeyep.ListAPIKeysEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)
	createAPIKeyEndpoint := (&apikeyep.CreateAPIKeyEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)
	rotateAPIKeyEndpoint := (&apikeyep.RotateAPIKeyEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)
	revokeAPIKeyEndpoint := (&apikeyep.RevokeAPIKeyEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)
	getDocAPIKeyEndpoint := (&apikeyep.GetDocAPIKeyEndpoint{}).Materialize().WithMiddleware(authMw).WithService(inner, apiKeySvc)

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
