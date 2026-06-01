package resourceregistry

import (
	"context"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSalesOrderLine,
		Load:       stubLoadSalesOrderLines,
		Subs: []resourcekit.SubField{
			{
				Key:         "item",
				Target:      constants.ObjectTypeItem,
				ExtractRefs: extractItemRefsFromSOLine,
			},
		},
	})
}

func stubLoadSalesOrderLines(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
	return nil, nil
}

func extractItemRefsFromSOLine(_ context.Context, parent any) []any {
	l := parent.(*apiresource.SalesOrderLineDetail)
	if l.Item == nil {
		return nil
	}
	return []any{l.Item}
}
