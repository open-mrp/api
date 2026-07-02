package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeMessagingBlock,
		Load:       resourceloaders.LoadMessagingBlocks,
		Subs: []resourcekit.SubField{
			// blocked_user carries only its id inline (the block gRPC returns no profile), so it is fetched by id via the global account_user loader — which in turn stashes its own user/role/department FKs, letting blocked_user.user/role/department recurse.
			{
				Key:         "blocked_user",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractBlockedUserIDFromBlock,
				Populate:    populateBlockedUserOnBlock,
			},
		},
	})
}

func extractBlockedUserIDFromBlock(ctx context.Context, parent any) []string {
	b := parent.(*apiresource.MessagingBlock)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeMessagingBlock, b.ID, "blocked_user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateBlockedUserOnBlock(ctx context.Context, parent any, loaded map[string]any) {
	b := parent.(*apiresource.MessagingBlock)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeMessagingBlock, b.ID, "blocked_user_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		b.BlockedUser = v.(*apiresource.AccountUser)
	}
}
