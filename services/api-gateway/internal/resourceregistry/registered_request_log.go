package resourceregistry

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeRequestLog,
		Load:       resourceloaders.LoadRequestLogs,
		Subs: []include.SubField{
			{Key: "account", Populate: populateAccountOnRequestLog},
			{Key: "request_body", Populate: populateRequestBodyOnRequestLog},
			{Key: "response_body", Populate: populateResponseBodyOnRequestLog},
		},
	})
}

func populateAccountOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "account")
	if !ok {
		return
	}
	rl.Account = v.(*apiresource.Account)
}

func populateRequestBodyOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "request_body")
	if !ok {
		return
	}
	rl.RequestBodyJSON = v.(json.RawMessage)
}

func populateResponseBodyOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "response_body")
	if !ok {
		return
	}
	rl.ResponseBodyJSON = v.(json.RawMessage)
}
