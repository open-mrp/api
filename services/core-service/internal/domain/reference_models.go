package domain

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// ReferenceProbe is one place a resource is referenced from: rows of Table whose Column holds its ID. Each probe
// runs as an indexed EXISTS; TestReferenceProbes_AreIndexed holds every registered probe to that.
type ReferenceProbe struct {
	// Label (required) names the referencing records in the in-use message, plural: "sales orders".
	Label string
	// Table and Column (required) are plain identifiers.
	Table, Column string
	// Where (optional) narrows the rows when Column is shared by several kinds, e.g. "scope_type = 'item'". It is
	// pasted into the SQL, so it must be a constant, never caller input.
	Where string
}

// References is everything that blocks deleting one kind of resource while it is in use.
type References struct {
	// Noun (required) is the resource in the in-use message: "payment term".
	Noun string
	// Archivable resources point the caller to archiving instead.
	Archivable bool
	Probes     []ReferenceProbe
}

var (
	referencesMu sync.Mutex
	references   []References
)

// RegisterReferences records refs for the probe index test and returns it. Declare each resource's references in
// this package, so the test sees them: `var PaymentTermReferences = RegisterReferences(References{...})`.
func RegisterReferences(refs References) References {
	referencesMu.Lock()
	defer referencesMu.Unlock()
	references = append(references, refs)
	return refs
}

// RegisteredReferences is every References registered so far.
func RegisteredReferences() []References {
	referencesMu.Lock()
	defer referencesMu.Unlock()
	return append([]References(nil), references...)
}

// RequireUnused is the 409 resource_in_use when any probe finds a row referencing id. Run it in the delete's
// transaction.
func (r References) RequireUnused(ctx context.Context, repo ReferenceRepo, id string) *apierror.APIError {
	labels, apiErr := repo.InUseBy(ctx, id, r.Probes)
	if apiErr != nil {
		return apiErr
	}
	if len(labels) > 0 {
		return r.InUseError(labels)
	}
	return nil
}

// InUseError is the 409 resource_in_use for deleting a resource that the records named by labels still use.
func (r References) InUseError(labels []string) *apierror.APIError {
	msg := fmt.Sprintf("This %s can't be deleted while %s use it.", r.Noun, joinLabels(labels))
	if r.Archivable {
		msg += fmt.Sprintf(" Archive the %s instead by setting `status` to `archived`.", r.Noun)
	}
	return apierror.NewInUseError(msg)
}

// joinLabels lists labels once each, in order: "a", "a and b", "a, b and c".
func joinLabels(labels []string) string {
	var uniq []string
	seen := map[string]bool{}
	for _, l := range labels {
		if !seen[l] {
			seen[l] = true
			uniq = append(uniq, l)
		}
	}
	switch len(uniq) {
	case 0:
		return "other records"
	case 1:
		return uniq[0]
	default:
		return strings.Join(uniq[:len(uniq)-1], ", ") + " and " + uniq[len(uniq)-1]
	}
}

// RequireAssignable rejects newly assigning an archived resource to param with a 422 resource_archived. noun names
// the resource: "payment term". Records already using it are unaffected.
func RequireAssignable(param, noun string, status constants.ArchiveStatus) *apierror.APIError {
	if f, archived := ArchivedField(param, noun, status); archived {
		return apperr.NewResourceArchivedError(f.Param, f.Message)
	}
	return nil
}

// ArchivedField is RequireAssignable's failure as a field error, for collecting several into one 422.
func ArchivedField(param, noun string, status constants.ArchiveStatus) (apierror.FieldError, bool) {
	if status.IsActive() {
		return apierror.FieldError{}, false
	}
	return apperr.ResourceArchivedField(param,
		fmt.Sprintf("The %s is archived, so it can't be newly assigned. Choose an active %s.", noun, noun)), true
}
