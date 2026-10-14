package resourceloaders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/actor"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

func TestActorRefFromProto(t *testing.T) {
	t.Parallel()
	assert.Nil(t, ActorRefFromProto(nil))
	assert.Nil(t, ActorRefFromProto(&pb.Actor{}))

	system := ActorRefFromProto(&pb.Actor{Type: string(constants.ActorTypeSystem)})
	assert.Equal(t, &apiresource.ActorRef{Type: constants.ActorTypeSystem}, system, "a system actor has no id")

	key := ActorRefFromProto(&pb.Actor{Type: string(constants.ActorTypeAPIKey), Id: new("apke_1"), Name: new("Sync"), Client: new("Claude Code")})
	assert.Equal(t, &apiresource.ActorRef{Type: constants.ActorTypeAPIKey, ID: new("apke_1"), Name: new("Sync"), Client: new("Claude Code")}, key)
}

func TestActorRefOf(t *testing.T) {
	t.Parallel()
	assert.Equal(t, &apiresource.ActorRef{Type: constants.ActorTypeSystem}, ActorRefOf(actor.System()))
	assert.Equal(t, &apiresource.ActorRef{Type: constants.ActorTypeDevice, ID: new("dx_1")}, ActorRefOf(actor.New(constants.ActorTypeDevice, "dx_1")))
}

// Actors that need no lookup (system, already named) never reach a loader, so this runs without any client set.
func TestResolveActorRefs_NothingToLoad(t *testing.T) {
	t.Parallel()
	named := &apiresource.ActorRef{Type: constants.ActorTypeUser, ID: new("us_1"), Name: new("Ada")}
	system := &apiresource.ActorRef{Type: constants.ActorTypeSystem}
	ResolveActorRefs(context.Background(), []*apiresource.ActorRef{nil, named, system})
	assert.Equal(t, "Ada", *named.Name)
	assert.Nil(t, system.Name)
}
