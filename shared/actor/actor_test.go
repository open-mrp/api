package actor

import (
	"database/sql"
	"testing"

	authtypes "github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	client := "Claude Code"
	for _, tc := range []struct {
		name string
		a    Actor
		ok   bool
	}{
		{"user", New(constants.ActorTypeUser, "usr_1"), true},
		{"device", New(constants.ActorTypeDevice, "dev_1"), true},
		{"system", System(), true},
		{"api key with client", Actor{Type: constants.ActorTypeAPIKey, ID: "apke_1", Client: &client}, true},
		{"system with id", Actor{Type: constants.ActorTypeSystem, ID: "x"}, false},
		{"user without id", Actor{Type: constants.ActorTypeUser}, false},
		{"group is not a record actor", New(constants.ActorTypeGroup, "grp_1"), false},
		{"unknown type", New("robot", "r_1"), false},
		{"client on a user", Actor{Type: constants.ActorTypeUser, ID: "usr_1", Client: &client}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ok, tc.a.Validate() == nil)
		})
	}
}

func TestColumns_RoundTrip(t *testing.T) {
	for _, a := range []Actor{New(constants.ActorTypeAgent, "agt_1"), System()} {
		typ, id := a.Columns()
		got, err := FromColumns(typ, id)
		require.NoError(t, err)
		assert.Equal(t, a, got)
	}

	typ, id := System().Columns()
	assert.Equal(t, "system", typ)
	assert.False(t, id.Valid, "a system actor stores a NULL id")

	_, err := FromColumns("user", sql.NullString{})
	assert.Error(t, err)
}

func TestProto_RoundTrip(t *testing.T) {
	name, client := "Order sync", "Claude Code"
	for _, a := range []Actor{
		{Type: constants.ActorTypeAPIKey, ID: "apke_1", Name: &name, Client: &client},
		System(),
	} {
		got, err := FromProto(a.ToProto())
		require.NoError(t, err)
		assert.Equal(t, a, *got)
	}

	assert.Nil(t, System().ToProto().Id)
	none, err := FromProto(nil)
	require.NoError(t, err)
	assert.Nil(t, none)

	_, err = FromProto(&pb.Actor{Type: "user"})
	assert.Error(t, err)
}

func TestFromIdentity(t *testing.T) {
	accountID := "ac_1"
	identity := func(typ authtypes.IdentityActorType, id string) *authtypes.Identity {
		return &authtypes.Identity{Type: typ, Actor: &authtypes.IdentityActor{ID: id, AccountID: &accountID}}
	}

	for _, tc := range []struct {
		identity *authtypes.Identity
		want     Actor
	}{
		{identity(authtypes.IdentityActorTypeUser, "usr_1"), New(constants.ActorTypeUser, "usr_1")},
		{identity(authtypes.IdentityActorTypeAPIKey, "apke_1"), New(constants.ActorTypeAPIKey, "apke_1")},
		{identity(authtypes.IdentityActorTypeAgent, "agt_1"), New(constants.ActorTypeAgent, "agt_1")},
		{authtypes.NewSystemIdentity(constants.SystemActorStripe, accountID), System()},
	} {
		got, ok := FromIdentity(tc.identity)
		require.True(t, ok)
		assert.Equal(t, tc.want, got)
	}

	_, ok := FromIdentity(nil)
	assert.False(t, ok)
	_, ok = FromIdentity(identity(authtypes.IdentityActorTypeUnauthenticated, ""))
	assert.False(t, ok)
}
