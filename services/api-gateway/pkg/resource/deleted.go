package apiresource

import (
	"context"

	"github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/appctx"
	"github.com/open-mrp/apikit/object"
)

// Deleted adapts a delete service call to the deleted stub every DELETE returns: it runs call and, when it succeeds, answers {id, object, deleted: true} for the record the idParam path parameter names.
func Deleted[TReq, TResp any](call func(context.Context, TReq) (TResp, *apierror.APIError), objectType object.Type, idParam string) func(context.Context, TReq) (*object.Deleted, *apierror.APIError) {
	return func(ctx context.Context, req TReq) (*object.Deleted, *apierror.APIError) {
		if _, apiErr := call(ctx, req); apiErr != nil {
			return nil, apiErr
		}
		params, _ := appctx.GetPathParams(ctx)
		return object.NewDeleted(params[idParam], objectType), nil
	}
}
