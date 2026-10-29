package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeFeedbackReport,
		Load:       resourceloaders.LoadFeedbackReports,
		Subs: []resourcekit.SubField{
			{
				Key:         "user",
				Target:      constants.ObjectTypeUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractUserIDFromFeedbackReport,
				Populate:    populateUserOnFeedbackReport,
			},
		},
	})
}

func extractUserIDFromFeedbackReport(ctx context.Context, parent any) []string {
	r := parent.(*apiresource.FeedbackReport)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeFeedbackReport, r.ID, "user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUserOnFeedbackReport(ctx context.Context, parent any, loaded map[string]any) {
	r := parent.(*apiresource.FeedbackReport)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeFeedbackReport, r.ID, "user_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		r.User = v.(*apiresource.User)
	}
}
