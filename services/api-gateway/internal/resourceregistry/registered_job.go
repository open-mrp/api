package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAsyncJob,
		Load:       resourceloaders.LoadJobs,
		Subs: []include.SubField{
			{
				Key:         "created_by",
				Target:      constants.ObjectTypeActor,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractCreatedByIDFromJob,
				Populate:    populateCreatedByOnJob,
			},
		},
	})
}

// the creator is stored as a bare identity-actor id, so the job endpoint's service stashes
// it and preheats the Actor built from it (see jobep.StashJobMeta)
func jobCreatedByID(ctx context.Context, parent any) string {
	job, ok := parent.(*apiresource.AsyncJob)
	if !ok {
		return ""
	}
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeAsyncJob, job.ID, "created_by_id")
	return id
}

func extractCreatedByIDFromJob(ctx context.Context, parent any) []string {
	id := jobCreatedByID(ctx, parent)
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCreatedByOnJob(ctx context.Context, parent any, loaded map[string]any) {
	job, ok := parent.(*apiresource.AsyncJob)
	if !ok {
		return
	}
	if v, ok := loaded[jobCreatedByID(ctx, parent)]; ok {
		job.CreatedBy = v.(*apiresource.Actor)
	}
}
