package stub

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// OpenRouterClient reports no spend, for use in test mode.
type OpenRouterClient struct{}

func (c *OpenRouterClient) GetKeyUsage(_ context.Context, _ string) (*domain.OpenRouterKeyUsage, *apierror.APIError) {
	return &domain.OpenRouterKeyUsage{SpentToday: "0", SpentThisWeek: "0", SpentThisMonth: "0", SpentTotal: "0"}, nil
}
