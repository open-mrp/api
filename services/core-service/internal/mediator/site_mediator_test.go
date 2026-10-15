package mediator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func newSiteMedFixture(t *testing.T) (domain.SiteMed, *repositorymock.MockLocationRepo) {
	ctrl := gomock.NewController(t)
	locations := repositorymock.NewMockLocationRepo(ctrl)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewLocationRepo().Return(locations).AnyTimes()
	return NewSiteMed(&SiteMedConfig{Repos: factory}), locations
}

func TestSiteMed_ResolveDefaultsToTheAccountsDefaultSite(t *testing.T) {
	t.Parallel()
	med, locations := newSiteMedFixture(t)
	def := "sglc_main"
	locations.EXPECT().DefaultSiteID(gomock.Any(), "ac_1").Return(&def, nil)

	got, apiErr := med.Resolve(context.Background(), "ac_1", nil, "site_id")
	require.Nil(t, apiErr)
	assert.Equal(t, &def, got)
}

func TestSiteMed_ResolveChecksTheNamedSite(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		found     []*domain.Location
		wantCode  apierror.Code
		wantFound bool
	}{
		{"active site", []*domain.Location{{ID: "sglc_2", TypeCode: "site", Status: constants.ArchiveStatusActive}}, "", true},
		{"missing", nil, apierror.CodeValidationFailed, false},
		{"not a site", []*domain.Location{{ID: "sglc_2", TypeCode: "building", Status: constants.ArchiveStatusActive}}, apierror.CodeValidationFailed, false},
		{"archived", []*domain.Location{{ID: "sglc_2", TypeCode: "site", Status: constants.ArchiveStatusArchived}}, "resource_archived", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			med, locations := newSiteMedFixture(t)
			locations.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"sglc_2"}).Return(tc.found, nil)
			id := "sglc_2"

			got, apiErr := med.Resolve(context.Background(), "ac_1", &id, "site_id")
			if !tc.wantFound {
				require.NotNil(t, apiErr)
				assert.Equal(t, tc.wantCode, apiErr.Code)
				assert.Equal(t, "site_id", apiErr.Param)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, "sglc_2", *got)
		})
	}
}

func TestSiteMed_LocationIDsInSites(t *testing.T) {
	t.Parallel()
	med, locations := newSiteMedFixture(t)
	p := func(s string) *string { return &s }
	locations.EXPECT().ListTreeNodes(gomock.Any(), "ac_1").Return([]domain.LocationTreeNode{
		{ID: "main", TypeCode: "site"},
		{ID: "bin", ParentID: p("main"), TypeCode: "bin"},
		{ID: "loose", TypeCode: "building"},
		{ID: "plant2", TypeCode: "site"},
		{ID: "rack", ParentID: p("plant2"), TypeCode: "rack"},
	}, nil)
	locations.EXPECT().DefaultSiteID(gomock.Any(), "ac_1").Return(p("main"), nil)

	got, apiErr := med.LocationIDsInSites(context.Background(), "ac_1", []string{"main"})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"main", "bin", "loose"}, got)
}
