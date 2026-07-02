package constants

// ReplyDraftStatus is the lifecycle of a structured customer-reply draft. Draft-first is the safe default: an agent or user proposes (Draft); a human approves & sends (Sent) or discards (Rejected).
// Superseded marks a draft stale (its source thread changed) so it can no longer be sent.
type ReplyDraftStatus string

const (
	// ReplyDraftStatusDraft is an open, editable draft awaiting review.
	ReplyDraftStatusDraft ReplyDraftStatus = "draft"
	// ReplyDraftStatusApproved is reserved for a two-step approve-then-send flow (approved, not yet sent).
	ReplyDraftStatusApproved ReplyDraftStatus = "approved"
	// ReplyDraftStatusSent has been materialized into a customer-visible message (and delivered).
	ReplyDraftStatusSent ReplyDraftStatus = "sent"
	// ReplyDraftStatusRejected was discarded without sending.
	ReplyDraftStatusRejected ReplyDraftStatus = "rejected"
	// ReplyDraftStatusSuperseded was invalidated because the thread it was drafted from changed.
	ReplyDraftStatusSuperseded ReplyDraftStatus = "superseded"
)

func (s ReplyDraftStatus) IsValid() bool {
	switch s {
	case ReplyDraftStatusDraft, ReplyDraftStatusApproved, ReplyDraftStatusSent, ReplyDraftStatusRejected, ReplyDraftStatusSuperseded:
		return true
	default:
		return false
	}
}

func (s ReplyDraftStatus) EnumValues() []string {
	return []string{
		string(ReplyDraftStatusDraft), string(ReplyDraftStatusApproved), string(ReplyDraftStatusSent),
		string(ReplyDraftStatusRejected), string(ReplyDraftStatusSuperseded),
	}
}

func (s *ReplyDraftStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
