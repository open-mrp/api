package resourceregistry

import (
	"context"
	"encoding/json"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeRequestLog,
		Load:       resourceloaders.LoadRequestLogs,
		Subs: []resourcekit.SubField{
			{Key: "account", Populate: populateAccountOnRequestLog},
			{
				Key:         "actor",
				Target:      constants.ObjectTypeActor,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractActorIDFromRequestLog,
				Populate:    populateActorOnRequestLog,
			},
			{Key: "query_params", Populate: populateQueryParamsOnRequestLog},
			{Key: "request_body", Populate: populateRequestBodyOnRequestLog},
			{Key: "response_body", Populate: populateResponseBodyOnRequestLog},
		},
	})
}

func populateAccountOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "account")
	if !ok {
		return
	}
	rl.Account = v.(*apiresource.Account)
}

func extractActorIDFromRequestLog(ctx context.Context, parent any) []string {
	rl := parent.(*apiresource.RequestLog)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeRequestLog, rl.ID, "actor_id")
	if !ok || id == "" {
		return nil
	}
	return []string{id}
}

func populateActorOnRequestLog(ctx context.Context, parent any, loaded map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeRequestLog, rl.ID, "actor_id")
	if !ok || id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		rl.Actor = v.(*apiresource.Actor)
	}
}

func populateQueryParamsOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "query_params")
	if !ok {
		return
	}
	rl.QueryJSON = v.(json.RawMessage)
}

func populateRequestBodyOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "request_body")
	if !ok {
		return
	}
	rl.RequestBodyJSON = v.(json.RawMessage)
}

func populateResponseBodyOnRequestLog(ctx context.Context, parent any, _ map[string]any) {
	rl := parent.(*apiresource.RequestLog)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeRequestLog, rl.ID, "response_body")
	if !ok {
		return
	}
	rl.ResponseBodyJSON = v.(json.RawMessage)
}
