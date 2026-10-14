package repository

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

var referenceRepoTracer = tracing.GetTracer("core-service.reference_repository")

type referenceRepoImpl struct {
	db sqlc.DBTX
}

func NewReferenceRepo(queries *sqlc.Queries) domain.ReferenceRepo {
	return &referenceRepoImpl{db: queries.DB()}
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// probeSQL is the probe's lookup of one referencing row, with id as its only parameter.
func probeSQL(p domain.ReferenceProbe) (string, error) {
	if p.Label == "" || !identifierPattern.MatchString(p.Table) || !identifierPattern.MatchString(p.Column) {
		return "", fmt.Errorf("invalid reference probe %+v", p)
	}
	q := fmt.Sprintf("SELECT 1 FROM `%s` WHERE `%s` = ?", p.Table, p.Column)
	if p.Where != "" {
		q += " AND (" + p.Where + ")"
	}
	return q + " LIMIT 1", nil
}

// InUseBy checks every probe in one round trip: one indexed EXISTS per probe.
func (r *referenceRepoImpl) InUseBy(ctx context.Context, id string, probes []domain.ReferenceProbe) ([]string, *apierror.APIError) {
	ctx, span := referenceRepoTracer.Start(ctx, "repository.reference.in_use_by")
	defer span.End()

	if len(probes) == 0 {
		return nil, nil
	}
	exists := make([]string, len(probes))
	args := make([]any, len(probes))
	for i, p := range probes {
		q, err := probeSQL(p)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Invalid reference probe."))
		}
		exists[i] = "EXISTS(" + q + ")"
		args[i] = id
	}

	found := make([]bool, len(probes))
	dest := make([]any, len(probes))
	for i := range found {
		dest[i] = &found[i]
	}
	if err := r.db.QueryRowContext(ctx, "SELECT "+strings.Join(exists, ", "), args...).Scan(dest...); err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check references."))
	}

	var labels []string
	for i, p := range probes {
		if found[i] {
			labels = append(labels, p.Label)
		}
	}
	return labels, nil
}
