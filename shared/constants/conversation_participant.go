package constants

// ParticipantType identifies what kind of actor a conversation participant is.
type ParticipantType string

const (
	// ParticipantTypeUser is an account user participant.
	ParticipantTypeUser ParticipantType = "user"
	// ParticipantTypeAgent is an AI agent participant.
	ParticipantTypeAgent ParticipantType = "agent"
	// ParticipantTypeSystem is the system pseudo-participant.
	ParticipantTypeSystem ParticipantType = "system"
	// ParticipantTypeCustomer is a customer relation participant (cross-account, no account_user), keyed by the customer's account in a customer-audience portal support case.
	ParticipantTypeCustomer ParticipantType = "customer"
)

func (t ParticipantType) IsValid() bool {
	switch t {
	case ParticipantTypeUser, ParticipantTypeAgent, ParticipantTypeSystem, ParticipantTypeCustomer:
		return true
	default:
		return false
	}
}

func (t ParticipantType) EnumValues() []string {
	return []string{string(ParticipantTypeUser), string(ParticipantTypeAgent), string(ParticipantTypeSystem), string(ParticipantTypeCustomer)}
}

func (t *ParticipantType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

// MessagingGroupMemberType identifies what kind of member a reusable-roster member is.
type MessagingGroupMemberType string

const (
	// MessagingGroupMemberTypeUser is an account user roster member.
	MessagingGroupMemberTypeUser MessagingGroupMemberType = "user"
	// MessagingGroupMemberTypeAgent is an AI agent roster member.
	MessagingGroupMemberTypeAgent MessagingGroupMemberType = "agent"
)

func (t MessagingGroupMemberType) IsValid() bool {
	switch t {
	case MessagingGroupMemberTypeUser, MessagingGroupMemberTypeAgent:
		return true
	default:
		return false
	}
}

func (t MessagingGroupMemberType) EnumValues() []string {
	return []string{string(MessagingGroupMemberTypeUser), string(MessagingGroupMemberTypeAgent)}
}

func (t *MessagingGroupMemberType) StringPtr() *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

// ParticipantRole is a participant's permission level within a conversation.
type ParticipantRole string

const (
	// ParticipantRoleOwner can rename/delete, manage members and roles.
	ParticipantRoleOwner ParticipantRole = "owner"
	// ParticipantRoleAdmin can add/remove members and rename.
	ParticipantRoleAdmin ParticipantRole = "admin"
	// ParticipantRoleMember can post, leave, mute, react.
	ParticipantRoleMember ParticipantRole = "member"
	// ParticipantRoleViewer is read-only.
	ParticipantRoleViewer ParticipantRole = "viewer"
)

func (r ParticipantRole) IsValid() bool {
	switch r {
	case ParticipantRoleOwner, ParticipantRoleAdmin, ParticipantRoleMember, ParticipantRoleViewer:
		return true
	default:
		return false
	}
}

func (r ParticipantRole) EnumValues() []string {
	return []string{string(ParticipantRoleOwner), string(ParticipantRoleAdmin), string(ParticipantRoleMember), string(ParticipantRoleViewer)}
}

func (r *ParticipantRole) StringPtr() *string {
	if r == nil {
		return nil
	}
	v := string(*r)
	return &v
}

// ParticipantMembership is a participant's membership in a conversation.
type ParticipantMembership string

const (
	// ParticipantMembershipActive is an active member.
	ParticipantMembershipActive ParticipantMembership = "active"
	// ParticipantMembershipLeft means the participant voluntarily left.
	ParticipantMembershipLeft ParticipantMembership = "left"
	// ParticipantMembershipRemoved means an admin removed the participant.
	ParticipantMembershipRemoved ParticipantMembership = "removed"
	// ParticipantMembershipHidden means the participant hid the conversation.
	ParticipantMembershipHidden ParticipantMembership = "hidden"
)

func (s ParticipantMembership) IsValid() bool {
	switch s {
	case ParticipantMembershipActive, ParticipantMembershipLeft, ParticipantMembershipRemoved, ParticipantMembershipHidden:
		return true
	default:
		return false
	}
}

func (s ParticipantMembership) EnumValues() []string {
	return []string{string(ParticipantMembershipActive), string(ParticipantMembershipLeft), string(ParticipantMembershipRemoved), string(ParticipantMembershipHidden)}
}

func (s *ParticipantMembership) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
