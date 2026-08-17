package service

import (
	"context"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	factorymock "github.com/augno/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/augno/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// --- ItemCategorySvcTestSuite ---

type ItemCategorySvcTestSuite struct {
	suite.Suite
	itemCategorySvc domain.ItemCategorySvc
	icRepo          *repositorymock.MockItemCategoryRepo
	ugRepo          *repositorymock.MockUnitGroupRepo
	propertyRepo    *repositorymock.MockPropertyRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *ItemCategorySvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.icRepo = repositorymock.NewMockItemCategoryRepo(suite.ctrl)
	suite.ugRepo = repositorymock.NewMockUnitGroupRepo(suite.ctrl)
	suite.propertyRepo = repositorymock.NewMockPropertyRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewItemCategoryRepo().Return(suite.icRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewUnitGroupRepo().Return(suite.ugRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewPropertyRepo().Return(suite.propertyRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(repositorymock.NewMockDeletedRecordRepo(suite.ctrl)).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.itemCategorySvc = NewItemCategorySvc(&ItemCategorySvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *ItemCategorySvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestItemCategorySvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ItemCategorySvcTestSuite))
}

// --- identity helpers ---

func internalCategoryCtx(targetAccountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"item_categories:read":   true,
				"item_categories:create": true,
				"item_categories:update": true,
				"item_categories:delete": true,
			},
		},
	})
}

// --- test data helpers ---

func sampleCategory(id, name, ugID string) *domain.ItemCategoryFull {
	acctID := "ac_test123"
	return &domain.ItemCategoryFull{
		ID:                   id,
		Name:                 name,
		ItemCategoryTypeCode: "material",
		UnitGroupID:          ugID,
		AccountID:            &acctID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}
}

func sampleProperty(id, name string) *domain.Property {
	return &domain.Property{
		ID:        id,
		Name:      name,
		AccountID: "ac_test123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// --- writeBulkUpsertItemCategories (the engine's Write hook, exercised directly) ---

func (suite *ItemCategorySvcTestSuite) writeItemCategories(rows ...domain.ResolvedUpsertItemCategoryRow) (created, updated []string, rowErrs []apierror.RowError, apiErr *apierror.APIError) {
	res, apiErr := writeBulkUpsertItemCategories(internalCategoryCtx("ac_test123"), suite.repoFactory, passthroughSavepoint{}, "ac_test123", rows)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	created, updated = splitJobResults(res.Results)
	return created, updated, res.Errors, nil
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_CreateNoProperties() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// No property names → FindByNames on propertyRepo is NOT called.

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Steel", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Steel", "ug_001"), nil).AnyTimes()

	created, updated, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{Name: "Steel", ItemCategoryTypeCode: "material", UnitGroupID: "ug_001"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_001"}, created)
	suite.Empty(updated)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_AttachExistingProperties() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// Properties are found — no creation needed.
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.Property{
			sampleProperty("prop_001", "Color"),
			sampleProperty("prop_002", "Material"),
		}, nil).Times(1)

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Steel", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Steel", "ug_001"), nil).AnyTimes()

	// UpsertProperty called once per requested property.
	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_001", "prop_001").Return(nil).Times(1)
	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_001", "prop_002").Return(nil).Times(1)

	created, _, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Steel",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Color", "Material"},
		},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_001"}, created)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_CreateMissingProperty() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// FindByNames returns nothing — property must be created.
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleProperty("prop_new", "Hardness"), nil).Times(1)

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_002", "Aluminum", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_002", "Aluminum", "ug_001"), nil).AnyTimes()

	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_002", "prop_new").Return(nil).Times(1)

	created, _, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Aluminum",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Hardness"},
		},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_002"}, created)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_MixedExistingAndNewProperties() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// FindByNames returns one of the two properties.
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.Property{sampleProperty("prop_001", "Color")}, nil).Times(1)

	// "Density" is not found → created.
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleProperty("prop_new", "Density"), nil).Times(1)

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_003", "Copper", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_003", "Copper", "ug_001"), nil).AnyTimes()

	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_003", gomock.Any()).Return(nil).Times(2)

	created, _, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Copper",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Color", "Density"},
		},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_003"}, created)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_DeduplicatesPropertyNames() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// "Color" is shared between both categories → FindByNames called once with deduplicated list.
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, names []string) ([]*domain.Property, *apierror.APIError) {
			suite.Len(names, 1) // "color" deduplicated
			return []*domain.Property{sampleProperty("prop_001", "Color")}, nil
		}).Times(1)

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, id string, params domain.CreateItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
			return sampleCategory(id, params.Name, params.UnitGroupID), nil
		}).Times(2)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p domain.GetItemCategoryParams) (*domain.ItemCategoryFull, *apierror.APIError) {
			return sampleCategory(p.ItemCategoryID, "Test", "ug_001"), nil
		}).AnyTimes()

	// Each category attaches the shared property once.
	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), gomock.Any(), "prop_001").Return(nil).Times(2)

	created, _, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{Name: "Iron", ItemCategoryTypeCode: "material", UnitGroupID: "ug_001", PropertyNames: []string{"Color"}},
		domain.ResolvedUpsertItemCategoryRow{Name: "Lead", ItemCategoryTypeCode: "material", UnitGroupID: "ug_001", PropertyNames: []string{"Color"}},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Len(created, 2)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_UpdateCategoryAttachesProperties() {
	acctID := "ac_test123"
	existing := sampleCategory("ic_existing", "Steel", "ug_001")

	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.ItemCategoryFull{existing}, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.Property{sampleProperty("prop_001", "Hardness")}, nil).Times(1)

	suite.icRepo.EXPECT().
		UpdateWithUnitGroup(gomock.Any(), gomock.Any()).
		Return(&domain.ItemCategoryFull{
			ID:                   "ic_existing",
			Name:                 "Steel",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			AccountID:            &acctID,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}, nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_existing", "Steel", "ug_001"), nil).AnyTimes()

	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_existing", "prop_001").Return(nil).Times(1)

	created, updated, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Steel",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Hardness"},
		},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_existing"}, updated)
	suite.Empty(created)
}

// --- writeBulkUpsertItemCategories: failures before the row loop sink the whole batch ---

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_PropertyFindByNamesError() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// Property resolution runs before the row loop, so its failure fails the batch.
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "db error")).Times(1)

	_, _, _, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Titanium",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Shine"},
		},
	)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_PropertyCreateError() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(nil, "create error")).Times(1)

	_, _, _, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Zinc",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Conductivity"},
		},
	)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

// A failure inside the row's savepoint drops only that row — the batch still completes.
func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_UpsertPropertyErrorIsRowScoped() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.Property{sampleProperty("prop_001", "Weight")}, nil).Times(1)

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Silver", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_001", "Silver", "ug_001"), nil).AnyTimes()

	suite.icRepo.EXPECT().
		UpsertProperty(gomock.Any(), "ic_001", "prop_001").
		Return(apierror.NewInternalError(nil, "upsert failed")).Times(1)

	created, updated, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{
			Name:                 "Silver",
			ItemCategoryTypeCode: "material",
			UnitGroupID:          "ug_001",
			PropertyNames:        []string{"Weight"},
		},
	)

	suite.Nil(err, "a row failure is recorded on the job, not returned")
	suite.Empty(created)
	suite.Empty(updated)
	suite.Len(rowErrs, 1)
	suite.Equal(0, rowErrs[0].Index)
}

func (suite *ItemCategorySvcTestSuite) TestWriteBulkUpsertItemCategories_NilPropertyNames_SkipsPropertyResolve() {
	suite.icRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)

	suite.ugRepo.EXPECT().
		GetTypesByIDs(gomock.Any(), "ac_test123", gomock.Any()).
		Return(map[string]string{"ug_001": "weight"}, nil).Times(1)

	// No PropertyNames on the row → property repo FindByNames must NOT be called.

	suite.icRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_004", "Nickel", "ug_001"), nil).Times(1)

	suite.icRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(sampleCategory("ic_004", "Nickel", "ug_001"), nil).AnyTimes()

	created, _, rowErrs, err := suite.writeItemCategories(
		domain.ResolvedUpsertItemCategoryRow{Name: "Nickel", ItemCategoryTypeCode: "material", UnitGroupID: "ug_001"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"ic_004"}, created)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// reopens the export so assertions read what a spreadsheet would
func (suite *ItemCategorySvcTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Categories")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *ItemCategorySvcTestSuite) buildExport(ctx context.Context, params domain.ExportItemCategoriesParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.itemCategorySvc.(*itemCategorySvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func (suite *ItemCategorySvcTestSuite) TestExportItemCategories_JoinsPropertyNames() {
	ctx := internalCategoryCtx("ac_test123")
	notes := "Yarn only"
	suite.icRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.ItemCategoryFull{
		{
			ID:                   "ic_1",
			Name:                 "Yarn",
			ItemCategoryTypeCode: "material",
			Notes:                &notes,
			UnitGroup:            &domain.ItemCategoryUnitGroup{Name: "Mass"},
			Properties: []*domain.ItemCategoryProperty{
				{ID: "prp_1", Name: "Color"},
				{ID: "prp_2", Name: "Weight"},
			},
		},
		{ID: "ic_2", Name: "Trim", ItemCategoryTypeCode: "part"},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportItemCategoriesParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	suite.Equal([]string{"ID", "Name", "Type", "Unit Group", "Properties", "Notes"}, rows[0])
	suite.Equal([]string{"ic_1", "Yarn", "material", "Mass", "Color; Weight", "Yarn only"}, rows[1])
	// A category with no unit group or properties stops at its type once blanks are trimmed.
	suite.Equal([]string{"ic_2", "Trim", "part"}, rows[2])
}

// the Properties header carries the semicolon guidance the importer relies on
func (suite *ItemCategorySvcTestSuite) TestExportItemCategories_NotesThePropertiesColumn() {
	ctx := internalCategoryCtx("ac_test123")
	suite.icRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportItemCategoriesParams{})
	suite.Require().Nil(apiErr)

	comments, err := openExportFile(suite.T(), export).GetComments("Categories")
	suite.Require().NoError(err)
	suite.Require().Len(comments, 1)
	suite.Equal("E1", comments[0].Cell)
	suite.Contains(comments[0].Text, "separated by semicolons")
}

func (suite *ItemCategorySvcTestSuite) TestExportItemCategories_ScopesToTheIdentitysAccount() {
	ctx := internalCategoryCtx("ac_owner")
	suite.icRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportItemCategoriesParams) ([]*domain.ItemCategoryFull, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportItemCategoriesParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}
