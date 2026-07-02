package domain

import "time"

// Messaging group member types: "user" for an internal account_user, "agent" for an AI agent config.
const (
	MessagingGroupMemberTypeUser  = "user"
	MessagingGroupMemberTypeAgent = "agent"
)

// MessagingGroup is a reusable, named roster of members (users and/or agents). It seeds new
// conversations: at create time its members are snapshotted into conversation_participant, so the
// same roster can back many conversations (each with its own title) and later edits never alter
// already-created conversations. The group stores only who belongs; roles and agent trigger config are per-conversation.
type MessagingGroup struct {
	ID                     string
	AccountID              string
	Name                   string `audit:"name"`
	CreatedByAccountUserID *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	// Members is populated on get/create/mutation paths; nil on bare list rows.
	Members []*MessagingGroupMember
}

// MessagingGroupMember is one member of a roster: either a user (AccountUserID) or an agent (AgentConfigID), discriminated by MemberType.
type MessagingGroupMember struct {
	ID            string
	GroupID       string `audit:"group_id"`
	AccountID     string
	MemberType    string  `audit:"member_type"`
	AccountUserID *string `audit:"account_user_id"`
	AgentConfigID *string `audit:"agent_config_id"`
	CreatedAt     time.Time
}

// CreateMessagingGroupInput is the validated input for creating a roster.
type CreateMessagingGroupInput struct {
	Name                 string
	MemberAccountUserIDs []string
	MemberAgentConfigIDs []string
}

// AddMessagingGroupMemberInput is the validated input for adding a single roster member. Exactly one of AccountUserID / AgentConfigID is set, matching MemberType.
type AddMessagingGroupMemberInput struct {
	GroupID       string
	MemberType    string
	AccountUserID string
	AgentConfigID string
}
