package constants

// ConversationWorkflowStatus is the triage lane of an external (audience=customer) customer-service case. It is orthogonal to the per-caller ConversationStatus (active/hidden/archived): a case is archived via is_archived, while the workflow status drives the support inbox. Null on internal conversations.
type ConversationWorkflowStatus string

const (
	// ConversationWorkflowStatusNew is a freshly opened case nobody has triaged yet.
	ConversationWorkflowStatusNew ConversationWorkflowStatus = "new"
	// ConversationWorkflowStatusOpen is an actively worked case.
	ConversationWorkflowStatusOpen ConversationWorkflowStatus = "open"
	// ConversationWorkflowStatusWaitingInternal is blocked on the internal team.
	ConversationWorkflowStatusWaitingInternal ConversationWorkflowStatus = "waiting_internal"
	// ConversationWorkflowStatusWaitingExternal is blocked on an external reply.
	ConversationWorkflowStatusWaitingExternal ConversationWorkflowStatus = "waiting_external"
	// ConversationWorkflowStatusNeedsApproval has a draft reply awaiting human approval.
	ConversationWorkflowStatusNeedsApproval ConversationWorkflowStatus = "needs_approval"
	// ConversationWorkflowStatusResolved is a closed-out case.
	ConversationWorkflowStatusResolved ConversationWorkflowStatus = "resolved"
)

func (s ConversationWorkflowStatus) IsValid() bool {
	switch s {
	case ConversationWorkflowStatusNew, ConversationWorkflowStatusOpen, ConversationWorkflowStatusWaitingInternal,
		ConversationWorkflowStatusWaitingExternal, ConversationWorkflowStatusNeedsApproval, ConversationWorkflowStatusResolved:
		return true
	default:
		return false
	}
}

func (s ConversationWorkflowStatus) EnumValues() []string {
	return []string{
		string(ConversationWorkflowStatusNew), string(ConversationWorkflowStatusOpen),
		string(ConversationWorkflowStatusWaitingInternal), string(ConversationWorkflowStatusWaitingExternal),
		string(ConversationWorkflowStatusNeedsApproval), string(ConversationWorkflowStatusResolved),
	}
}

func (s *ConversationWorkflowStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
