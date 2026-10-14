package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/open-mrp/api/shared/apperr"
	apierror "github.com/open-mrp/apikit/apierror"
)

// RequireTransition rejects a lifecycle action the record's status doesn't allow with a 409 invalid_state_transition. noun names the record ("sales order"), verb the action as its route spells it ("issue"), and from the statuses it may start from. Check it inside the transaction, against the row as locked, so two concurrent actions can't both pass.
func RequireTransition[S ~string](noun string, current S, verb string, from ...S) *apierror.APIError {
	if slices.Contains(from, current) {
		return nil
	}
	allowed := make([]string, len(from))
	for i, s := range from {
		allowed[i] = "`" + string(s) + "`"
	}
	return apperr.NewInvalidStateTransitionError(fmt.Sprintf(
		"This %s is `%s`; `%s` needs it to be %s.", noun, current, verb, joinOr(allowed)))
}

func joinOr(items []string) string {
	switch len(items) {
	case 0:
		return "in another status"
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " or " + items[len(items)-1]
	}
}
