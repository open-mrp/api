package apiresource

import "github.com/open-mrp/api/shared/constants"

// Who a notification is aimed at.
type NotificationTargetInput struct {
	// The kind of recipient being addressed.
	//
	// - `account_user`: one member of the account, who receives a personal notification in their feed.
	// - `account`: every member of the account, who all receive a single shared announcement.
	Type constants.NotificationTargetType `json:"type" validate:"required"`
	// The id of the recipient, matching `type`: an account user id, or an account id.
	//
	// An account target must be the account you are currently acting in — you cannot broadcast into another account.
	ID string `json:"id" validate:"required"`
}
