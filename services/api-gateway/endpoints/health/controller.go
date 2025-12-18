package healthep

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/contracts"
)

type HealthCtrl interface {
	GetHealth(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *contracts.APIError)
}

type healthCtrlImpl struct {
}

func NewHealthCtrl() HealthCtrl {
	return &healthCtrlImpl{}
}

func (c *healthCtrlImpl) GetHealth(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *contracts.APIError) {
	hc := &apiresource.Healthcheck{
		Status: "healthy",
	}
	return hc, nil
}
