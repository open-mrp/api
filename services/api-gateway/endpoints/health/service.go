package healthep

import (
	"context"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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
		Object: constants.ObjectTypeHealthcheck,
		Status: "healthy",
	}
	return hc, nil
}
