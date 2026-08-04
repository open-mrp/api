package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const (
	SampleMessagingGroupID       = "cvgp_wjlypugna7s4"
	SampleMessagingGroupMemberID = "cvgppt_obu4df48t1xx"
)

// A reusable roster: a named set of members (users and/or agents) that seeds new conversations.
//
// Starting a conversation from a group snapshots its current members into that conversation, so the same group can back many conversations (each with its own title); later edits to the group never change conversations already created from it.
type MessagingGroup struct {
	// Messaging group ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=messaging_group"`
	// The roster's display name.
	Name string `json:"name" validate:"required"`
	// The roster's members (users and agents).
	Members *List[MessagingGroupMember] `json:"members"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

// A member of a reusable roster: either a user or an agent, represented by its actor.
type MessagingGroupMember struct {
	// Membership ID.
	//
	// This identifies the member's place on the roster, not the user or agent themselves; it is the id to pass when removing them from the roster.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=messaging_group_member"`
	// The member: a `user` (account user) or an `agent`.
	Actor *Actor `json:"actor" validate:"required"`
}

var SampleMessagingGroupMember = &MessagingGroupMember{
	ID:     SampleMessagingGroupMemberID,
	Object: constants.ObjectTypeMessagingGroupMember,
	Actor:  NewActor(SampleAccountUserID, constants.ActorTypeUser, new("Jie Yan"), nil),
}

var SampleMessagingGroup = &MessagingGroup{
	ID:        SampleMessagingGroupID,
	Object:    constants.ObjectTypeMessagingGroup,
	Name:      "Operations Team",
	Members:   NewList([]MessagingGroupMember{*SampleMessagingGroupMember}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*MessagingGroup) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessagingGroup)
}

func (*MessagingGroupMember) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessagingGroupMember)
}
