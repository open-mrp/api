package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type territorySvcSetup struct {
	svc          domain.TerritorySvc
	territories  *repositorymock.MockTerritoryRepo
	accountUsers *repositorymock.MockAccountUserRepo
	productLines *repositorymock.MockProductLineRepo
	idempotency  *mediatormock.MockIdempotencyMed
	outbox       *recordingOutboxRepo
}

func newTerritorySvcSetup(t *testing.T) *territorySvcSetup {
	ctrl := gomock.NewController(t)
	s := &territorySvcSetup{
		territories:  repositorymock.NewMockTerritoryRepo(ctrl),
		accountUsers: repositorymock.NewMockAccountUserRepo(ctrl),
		productLines: repositorymock.NewMockProductLineRepo(ctrl),
		idempotency:  mediatormock.NewMockIdempotencyMed(ctrl),
		outbox:       &recordingOutboxRepo{},
	}
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewTerritoryRepo().Return(s.territories).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(s.accountUsers).AnyTimes()
	repos.EXPECT().NewProductLineRepo().Return(s.productLines).AnyTimes()
	repos.EXPECT().NewOutboxRepo().Return(s.outbox).AnyTimes()
	mediators := factorymock.NewMockMediatorFactory(ctrl)
	mediators.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: s.idempotency}).AnyTimes()

	s.svc = NewTerritorySvc(&TerritorySvcConfig{
		Repos:           repos,
		MediatorFactory: mediators,
		TxManager:       &stubTxManager{factory: repos},
	})
	return s
}

// expectWrite lets one idempotent write run, handing back any error it caches.
func (s *territorySvcSetup) expectWrite() {
	s.idempotency.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_terr", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	s.idempotency.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_terr", gomock.Any()).Return(nil).AnyTimes()
	s.idempotency.EXPECT().CacheErrorResponse(gomock.Any(), "idk_terr", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, apiErr *apierror.APIError) *apierror.APIError { return apiErr }).AnyTimes()
}

// auditChanges returns the fields each published audit event changed, in order.
func (s *territorySvcSetup) auditChanges(t *testing.T) [][]string {
	t.Helper()
	var events [][]string
	for _, msg := range s.outbox.messages {
		var evt struct {
			Changes []audit.FieldChange `json:"changes"`
		}
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &evt))
		var fields []string
		for _, c := range evt.Changes {
			fields = append(fields, c.Field)
		}
		events = append(events, fields)
	}
	return events
}

func territoryInternalCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test",
			AccountID:    &accountID,
			RoleType:     &adminCode,
		},
	})
}

func territoryZip(v int32) *int32 { return &v }

func assertTerritoryParam(t *testing.T, apiErr *apierror.APIError, param string) {
	t.Helper()
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeValidationFailed, apiErr.Code)
	assert.Equal(t, param, apiErr.Param)
}

// storedTerritory is a territory as the repo reads it with the audit includes.
func storedTerritory(repID string, productLineID *string, start, end *int32) *domain.Territory {
	t := &domain.Territory{
		ID: "tr_1", State: "NY", StartZipcode: start, EndZipcode: end,
		SalesRepID: repID, SalesRep: &domain.TerritorySalesRep{ID: repID},
	}
	if productLineID != nil {
		t.ProductLineID = productLineID
		t.ProductLine = &domain.TerritoryProductLine{ID: *productLineID}
	}
	return t
}

func TestTerritorySvc_APathNamingAnotherAccountIsNotFound(t *testing.T) {
	t.Parallel()
	s := newTerritorySvcSetup(t)
	ctx := territoryInternalCtx("ac_mine")

	_, apiErr := s.svc.ListTerritories(ctx, domain.ListTerritoriesParams{AccountID: "ac_other"})
	assert.True(t, apierror.IsNotFound(apiErr), "list: %v", apiErr)
	_, apiErr = s.svc.GetTerritory(ctx, domain.GetTerritoryParams{AccountID: "ac_other", TerritoryID: "tr_1"})
	assert.True(t, apierror.IsNotFound(apiErr), "get: %v", apiErr)
	_, apiErr = s.svc.CreateTerritory(ctx, domain.CreateTerritoryParams{AccountID: "ac_other", State: "NY", SalesRepID: "acu_1"})
	assert.True(t, apierror.IsNotFound(apiErr), "create: %v", apiErr)
	state := "NY"
	_, apiErr = s.svc.UpdateTerritory(ctx, domain.UpdateTerritoryParams{AccountID: "ac_other", TerritoryID: "tr_1", State: &state})
	assert.True(t, apierror.IsNotFound(apiErr), "update: %v", apiErr)
	apiErr = s.svc.DeleteTerritory(ctx, domain.DeleteTerritoryParams{AccountID: "ac_other", TerritoryID: "tr_1"})
	assert.True(t, apierror.IsNotFound(apiErr), "delete: %v", apiErr)
}

func TestTerritorySvc_CreateRejectsARangeThatEndsBeforeItStarts(t *testing.T) {
	t.Parallel()
	s := newTerritorySvcSetup(t)

	_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{
		AccountID: "ac_1", State: "NY", SalesRepID: "acu_1", StartZipcode: territoryZip(20000), EndZipcode: territoryZip(10000),
	})
	assertTerritoryParam(t, apiErr, "end_zipcode")
}

func TestTerritorySvc_CreateRejectsReferencesTheAccountDoesNotHave(t *testing.T) {
	t.Parallel()
	notFound := apierror.NewResourceNotFoundError("Resource not found.")
	plID := "pdln_1"

	t.Run("a sales rep not in the account", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_elsewhere", nil).Return(nil, notFound)

		_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{AccountID: "ac_1", State: "NY", SalesRepID: "acu_elsewhere"})
		assertTerritoryParam(t, apiErr, "sales_rep_id")
	})

	t.Run("a sales rep removed from the account", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_gone", nil).
			Return(&domain.AccountUserDetail{ID: "acu_gone", StatusCode: constants.AccountUserStatusRemoved}, nil)

		_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{AccountID: "ac_1", State: "NY", SalesRepID: "acu_gone"})
		assertTerritoryParam(t, apiErr, "sales_rep_id")
	})

	t.Run("a product line not in the account", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_1", nil).
			Return(&domain.AccountUserDetail{ID: "acu_1", StatusCode: constants.AccountUserStatusActive}, nil)
		s.productLines.EXPECT().Get(gomock.Any(), domain.GetProductLineParams{AccountID: "ac_1", ProductLineID: plID}).Return(nil, notFound)

		_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{AccountID: "ac_1", State: "NY", SalesRepID: "acu_1", ProductLineID: &plID})
		assertTerritoryParam(t, apiErr, "product_line_id")
	})
}

func TestTerritorySvc_CreateAuditsTheSalesRepAndProductLine(t *testing.T) {
	t.Parallel()
	s := newTerritorySvcSetup(t)
	s.expectWrite()
	plID := "pdln_1"
	s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_1", nil).
		Return(&domain.AccountUserDetail{ID: "acu_1", StatusCode: constants.AccountUserStatusActive}, nil)
	s.productLines.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ProductLineFull{}, nil)
	s.territories.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params domain.CreateTerritoryParams) (*domain.Territory, *apierror.APIError) {
			assert.ElementsMatch(t, []string{"sales_rep", "product_line"}, params.Includes, "the audit diff needs the references read")
			return storedTerritory("acu_1", &plID, nil, nil), nil
		})

	_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{
		AccountID: "ac_1", State: "NY", SalesRepID: "acu_1", ProductLineID: &plID, EndZipcode: territoryZip(10999),
	})
	require.Nil(t, apiErr)
	events := s.auditChanges(t)
	require.Len(t, events, 1)
	assert.Subset(t, events[0], []string{"sales_rep_id", "product_line_id"})
}

func TestTerritorySvc_CreateDropsAnEndZipcodeWithoutAStart(t *testing.T) {
	t.Parallel()
	s := newTerritorySvcSetup(t)
	s.expectWrite()
	s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_1", nil).
		Return(&domain.AccountUserDetail{ID: "acu_1", StatusCode: constants.AccountUserStatusActive}, nil)
	s.territories.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, params domain.CreateTerritoryParams) (*domain.Territory, *apierror.APIError) {
			assert.Nil(t, params.EndZipcode)
			return storedTerritory("acu_1", nil, nil, nil), nil
		})

	_, apiErr := s.svc.CreateTerritory(territoryInternalCtx("ac_1"), domain.CreateTerritoryParams{
		AccountID: "ac_1", State: "NY", SalesRepID: "acu_1", EndZipcode: territoryZip(10999),
	})
	require.Nil(t, apiErr)
}

func TestTerritorySvc_UpdateRejectsClearingAndSettingTheSameField(t *testing.T) {
	t.Parallel()
	plID := "pdln_1"
	cases := []struct {
		name   string
		params domain.UpdateTerritoryParams
		param  string
	}{
		{"start ZIP code", domain.UpdateTerritoryParams{ClearStartZipcode: true, StartZipcode: territoryZip(20001)}, "clear_start_zipcode"},
		{"end ZIP code", domain.UpdateTerritoryParams{ClearEndZipcode: true, EndZipcode: territoryZip(20999)}, "clear_end_zipcode"},
		{"product line", domain.UpdateTerritoryParams{ClearProductLine: true, ProductLineID: &plID}, "clear_product_line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newTerritorySvcSetup(t)
			tc.params.AccountID, tc.params.TerritoryID = "ac_1", "tr_1"

			_, apiErr := s.svc.UpdateTerritory(territoryInternalCtx("ac_1"), tc.params)
			assertTerritoryParam(t, apiErr, tc.param)
		})
	}
}

func TestTerritorySvc_UpdateChecksOnlyTheReferencesItChanges(t *testing.T) {
	t.Parallel()
	plID, otherPL := "pdln_1", "pdln_2"
	notFound := apierror.NewResourceNotFoundError("Resource not found.")

	t.Run("an unchanged rep and product line are not looked up", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		stored := storedTerritory("acu_removed", &plID, nil, nil)
		s.territories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(stored, nil)
		s.territories.EXPECT().Update(gomock.Any(), gomock.Any()).Return(stored, nil)
		rep := "acu_removed"

		_, apiErr := s.svc.UpdateTerritory(territoryInternalCtx("ac_1"), domain.UpdateTerritoryParams{
			AccountID: "ac_1", TerritoryID: "tr_1", SalesRepID: &rep, ProductLineID: &plID,
		})
		require.Nil(t, apiErr)
	})

	t.Run("a new rep the account does not have", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		s.territories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(storedTerritory("acu_1", nil, nil, nil), nil)
		s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_elsewhere", nil).Return(nil, notFound)
		rep := "acu_elsewhere"

		_, apiErr := s.svc.UpdateTerritory(territoryInternalCtx("ac_1"), domain.UpdateTerritoryParams{AccountID: "ac_1", TerritoryID: "tr_1", SalesRepID: &rep})
		assertTerritoryParam(t, apiErr, "sales_rep_id")
	})

	t.Run("a new product line the account does not have", func(t *testing.T) {
		t.Parallel()
		s := newTerritorySvcSetup(t)
		s.expectWrite()
		s.territories.EXPECT().Get(gomock.Any(), gomock.Any()).Return(storedTerritory("acu_1", &plID, nil, nil), nil)
		s.productLines.EXPECT().Get(gomock.Any(), domain.GetProductLineParams{AccountID: "ac_1", ProductLineID: otherPL}).Return(nil, notFound)

		_, apiErr := s.svc.UpdateTerritory(territoryInternalCtx("ac_1"), domain.UpdateTerritoryParams{AccountID: "ac_1", TerritoryID: "tr_1", ProductLineID: &otherPL})
		assertTerritoryParam(t, apiErr, "product_line_id")
	})
}

func TestTerritorySvc_UpdateAuditsAReassignedRepAndAClearedProductLine(t *testing.T) {
	t.Parallel()
	s := newTerritorySvcSetup(t)
	s.expectWrite()
	plID := "pdln_1"
	s.territories.EXPECT().Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.GetTerritoryParams) (*domain.Territory, *apierror.APIError) {
			assert.ElementsMatch(t, []string{"sales_rep", "product_line"}, params.Includes)
			return storedTerritory("acu_1", &plID, nil, nil), nil
		})
	s.accountUsers.EXPECT().GetDetailByAccountAndID(gomock.Any(), "ac_1", "acu_2", nil).
		Return(&domain.AccountUserDetail{ID: "acu_2", StatusCode: constants.AccountUserStatusActive}, nil)
	s.territories.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.UpdateTerritoryParams) (*domain.Territory, *apierror.APIError) {
			assert.ElementsMatch(t, []string{"sales_rep", "product_line"}, params.Includes)
			return storedTerritory("acu_2", nil, nil, nil), nil
		})
	rep := "acu_2"

	_, apiErr := s.svc.UpdateTerritory(territoryInternalCtx("ac_1"), domain.UpdateTerritoryParams{
		AccountID: "ac_1", TerritoryID: "tr_1", SalesRepID: &rep, ClearProductLine: true,
	})
	require.Nil(t, apiErr)
	assert.Equal(t, [][]string{{"sales_rep_id", "product_line_id"}}, s.auditChanges(t))
}

func TestResolveUpdatedZipcodes(t *testing.T) {
	t.Parallel()
	ranged := &domain.Territory{StartZipcode: territoryZip(10001), EndZipcode: territoryZip(10999)}
	stateWide := &domain.Territory{}

	cases := []struct {
		name      string
		old       *domain.Territory
		params    domain.UpdateTerritoryParams
		param     string
		wantEnd   *int32
		wantClear bool
	}{
		{name: "start moved past the stored end", old: ranged, params: domain.UpdateTerritoryParams{StartZipcode: territoryZip(11000)}, param: "start_zipcode"},
		{name: "end moved before the stored start", old: ranged, params: domain.UpdateTerritoryParams{EndZipcode: territoryZip(10000)}, param: "end_zipcode"},
		{name: "both sent reversed", old: ranged, params: domain.UpdateTerritoryParams{StartZipcode: territoryZip(12000), EndZipcode: territoryZip(11000)}, param: "end_zipcode"},
		{name: "a narrowed range", old: ranged, params: domain.UpdateTerritoryParams{StartZipcode: territoryZip(10500), EndZipcode: territoryZip(10600)}, wantEnd: territoryZip(10600)},
		{name: "a start moved past a cleared end", old: ranged, params: domain.UpdateTerritoryParams{StartZipcode: territoryZip(20500), ClearEndZipcode: true}, wantClear: true},
		{name: "an end with no start is dropped", old: stateWide, params: domain.UpdateTerritoryParams{EndZipcode: territoryZip(10999)}, wantClear: true},
		{name: "clearing the start clears the end", old: ranged, params: domain.UpdateTerritoryParams{ClearStartZipcode: true}, wantClear: true},
		{name: "an end sent with a start clear is dropped", old: ranged, params: domain.UpdateTerritoryParams{ClearStartZipcode: true, EndZipcode: territoryZip(10500)}, wantClear: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := tc.params
			apiErr := resolveUpdatedZipcodes(tc.old, &params)
			if tc.param != "" {
				assertTerritoryParam(t, apiErr, tc.param)
				return
			}
			require.Nil(t, apiErr)
			assert.Equal(t, tc.wantEnd, params.EndZipcode)
			assert.Equal(t, tc.wantClear, params.ClearEndZipcode)
		})
	}
}
