package healthep

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

type HealthSvc interface {
	GetHealth(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *apierror.APIError)
}

type healthSvcImpl struct {
}

func NewHealthSvc() HealthSvc {
	return &healthSvcImpl{}
}

func (c *healthSvcImpl) GetHealth(ctx context.Context, req *apiresource.EmptyResource) (*apiresource.Healthcheck, *apierror.APIError) {
	hc := &apiresource.Healthcheck{
		Status: "healthy",
	}
	return hc, nil
}
