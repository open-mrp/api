package apiresource

import "github.com/augno/api/shared/constants"

// NotificationTargetInput selects what a notification send is aimed at.
//
// The target is a polymorphic reference carrying a `type` and the `id` it refers to. Modeling it this way, rather than a single id or a broadcast flag, lets new target kinds be added without a breaking change to the send API.
//
// Supported types:
//   - `account_user`: `id` is an account_user id; delivers a per-user notification.
//   - `account`: `id` is an account id; broadcasts an announcement to every user in the account.
type NotificationTargetInput struct {
	// The kind of target.
	Type constants.NotificationTargetType `json:"type" validate:"required"`
	// The id of the target (an account_user id or an account id, matching `type`).
	ID string `json:"id" validate:"required"`
}
