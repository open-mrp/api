package domain_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

var termRefs = domain.References{
	Noun:       "payment term",
	Archivable: true,
	Probes: []domain.ReferenceProbe{
		{Label: "customers", Table: "account_relation", Column: "payment_term_id"},
		{Label: "sales orders", Table: "sales_order", Column: "payment_term_id"},
	},
}

func TestReferences_InUseError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		refs   domain.References
		labels []string
		want   string
	}{
		{"one", termRefs, []string{"customers"},
			"This payment term can't be deleted while customers use it. Archive the payment term instead by setting `status` to `archived`."},
		{"two", termRefs, []string{"customers", "sales orders"},
			"This payment term can't be deleted while customers and sales orders use it. Archive the payment term instead by setting `status` to `archived`."},
		{"three with a repeat", domain.References{Noun: "address"}, []string{"sales orders", "invoices", "sales orders", "shipments"},
			"This address can't be deleted while sales orders, invoices and shipments use it."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.refs.InUseError(tc.labels)
			assert.Equal(t, apierror.CodeResourceInUse, err.Code)
			assert.Equal(t, http.StatusConflict, err.Status())
			assert.Equal(t, tc.want, err.PublicMessage)
		})
	}
}

func TestReferences_RequireUnused(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := repositorymock.NewMockReferenceRepo(ctrl)
	ctx := context.Background()

	repo.EXPECT().InUseBy(ctx, "pt_free", termRefs.Probes).Return(nil, nil)
	assert.Nil(t, termRefs.RequireUnused(ctx, repo, "pt_free"))

	repo.EXPECT().InUseBy(ctx, "pt_used", termRefs.Probes).Return([]string{"sales orders"}, nil)
	err := termRefs.RequireUnused(ctx, repo, "pt_used")
	require.NotNil(t, err)
	assert.Equal(t, apierror.CodeResourceInUse, err.Code)

	failure := apierror.NewInternalError(assert.AnError, "probe failed")
	repo.EXPECT().InUseBy(ctx, "pt_err", termRefs.Probes).Return(nil, failure)
	assert.Same(t, failure, termRefs.RequireUnused(ctx, repo, "pt_err"))
}

func TestRequireAssignable(t *testing.T) {
	assert.Nil(t, domain.RequireAssignable("payment_term_id", "payment term", constants.ArchiveStatusActive))

	err := domain.RequireAssignable("defaults.payment_term_id", "payment term", constants.ArchiveStatusArchived)
	require.NotNil(t, err)
	assert.Equal(t, apperr.CodeResourceArchived, err.Code)
	assert.Equal(t, http.StatusUnprocessableEntity, err.Status())
	assert.Equal(t, "defaults.payment_term_id", err.Param)
	require.Len(t, err.Errors, 1)
	assert.Equal(t, apperr.CodeResourceArchived, err.Errors[0].Code)
	assert.Equal(t, "The payment term is archived, so it can't be newly assigned. Choose an active payment term.", err.Errors[0].Message)
}

func TestArchivedField(t *testing.T) {
	_, archived := domain.ArchivedField("carrier_id", "carrier", constants.ArchiveStatusActive)
	assert.False(t, archived)

	f, archived := domain.ArchivedField("carrier_id", "carrier", constants.ArchiveStatusArchived)
	assert.True(t, archived)
	assert.Equal(t, "carrier_id", f.Param)
	assert.Equal(t, apperr.CodeResourceArchived, f.Code)
}

func TestRegisterReferences(t *testing.T) {
	refs := domain.RegisterReferences(domain.References{Noun: "test widget"})
	assert.Equal(t, "test widget", refs.Noun)
	assert.Contains(t, domain.RegisteredReferences(), refs)
}
