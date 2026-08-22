package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PartBulkUpsertTestSuite struct {
	suite.Suite
	partSvc         domain.PartSvc
	partRepo        *repositorymock.MockPartRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *PartBulkUpsertTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.partRepo = repositorymock.NewMockPartRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewPartRepo().Return(suite.partRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.partSvc = NewPartSvc(&PartSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *PartBulkUpsertTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestPartBulkUpsertTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PartBulkUpsertTestSuite))
}

func internalPartCtx(accountID string) context.Context {
	adminCode := string(constants.RoleTypeAdmin)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &accountID,
			RoleType:     &adminCode,
			Permissions: map[string]bool{
				"parts:read":   true,
				"parts:create": true,
				"parts:update": true,
				"parts:delete": true,
			},
		},
	})
}

// --- BulkUpsertParts validation guards (reject before idempotency / tx) ---

func (suite *PartBulkUpsertTestSuite) TestBulkUpsertParts_EmptyRejected() {
	ctx := internalPartCtx("ac_test123")

	result, err := suite.partSvc.BulkUpsertParts(ctx, domain.BulkUpsertPartsParams{Parts: nil})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *PartBulkUpsertTestSuite) TestBulkUpsertParts_TooManyRejected() {
	ctx := internalPartCtx("ac_test123")

	parts := make([]domain.UpsertPartParams, 1001)
	for i := range parts {
		parts[i] = domain.UpsertPartParams{SKU: "SKU-" + strconv.Itoa(i), Category: domain.ObjectIdentifier{ID: "ic_001"}}
	}

	result, err := suite.partSvc.BulkUpsertParts(ctx, domain.BulkUpsertPartsParams{Parts: parts})

	suite.Nil(result)
	suite.NotNil(err)
}

func (suite *PartBulkUpsertTestSuite) TestBulkUpsertParts_DuplicateSKURejected() {
	ctx := internalPartCtx("ac_test123")

	result, err := suite.partSvc.BulkUpsertParts(ctx, domain.BulkUpsertPartsParams{
		Parts: []domain.UpsertPartParams{
			{SKU: "SKU-DUP", Category: domain.ObjectIdentifier{ID: "ic_001"}},
			{SKU: "SKU-DUP", Category: domain.ObjectIdentifier{ID: "ic_001"}},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
}

// --- ExportParts ---

func (suite *PartBulkUpsertTestSuite) exportedRows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, "Parts")
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *PartBulkUpsertTestSuite) buildExport(ctx context.Context, params domain.ExportPartsParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.partSvc.(*partSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func (suite *PartBulkUpsertTestSuite) TestExportParts_WritesHeadersAndRows() {
	ctx := internalPartCtx("ac_test123")
	description := "Knitted panel"
	suite.partRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.Part{{
		ID: "pt_1",
		Item: &domain.Item{
			SKU:          "PNL-1",
			Description:  &description,
			CategoryName: "Panels",
			// Full DECIMAL scale, as a real row comes back.
			UnitValue: &domain.Rate{Value: "2.250000000000000000000000000000"},
			UnitCost:  &domain.Rate{Value: "1.000000000000000000000000000000"},
			Category: &domain.ItemCategory{Name: "Panels", Properties: []domain.ItemCategoryProperty{
				{ID: "pp_gauge", Name: "Gauge"},
			}},
			Attributes: []*domain.ItemAttribute{{PropertyID: "pp_gauge", Value: "12"}},
		},
	}}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportPartsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(1), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 2)
	suite.Equal([]string{
		"ID", "SKU", "Description", "Notes", "Category", "Unit Price", "Unit Cost", "Gauge",
	}, rows[0])
	suite.Equal([]string{"pt_1", "PNL-1", "Knitted panel", "", "Panels", "2.25", "1", "12"}, rows[1])
}

// two parts in different categories whose properties share a name write into one
// column, because the header is what the importer matches on
func (suite *PartBulkUpsertTestSuite) TestExportParts_SharesAColumnForASharedPropertyName() {
	ctx := internalPartCtx("ac_test123")
	panel := exportItemWithProperties("Panels",
		[]domain.ItemCategoryProperty{{ID: "pp_panel_gauge", Name: "Gauge"}},
		map[string]string{"Gauge": "12"})
	panel.SKU = "PNL-1"
	cuff := exportItemWithProperties("Cuffs",
		[]domain.ItemCategoryProperty{{ID: "pp_cuff_gauge", Name: "Gauge"}},
		map[string]string{"Gauge": "8"})
	cuff.SKU = "CFF-1"

	suite.partRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.Part{
		{ID: "pt_1", Item: panel},
		{ID: "pt_2", Item: cuff},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportPartsParams{})
	suite.Require().Nil(apiErr)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 3)
	// One "Gauge" column, not one per category, and each part's value under it.
	suite.Equal([]string{"ID", "SKU", "Description", "Notes", "Category", "Unit Price", "Unit Cost", "Gauge"}, rows[0])
	suite.Equal([]string{"pt_1", "PNL-1", "", "", "Panels", "", "", "12"}, rows[1])
	suite.Equal([]string{"pt_2", "CFF-1", "", "", "Cuffs", "", "", "8"}, rows[2])
}

// an account with nothing to export still gets a usable, header-only workbook
func (suite *PartBulkUpsertTestSuite) TestExportParts_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalPartCtx("ac_test123")
	suite.partRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportPartsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(0), export.RowCount)

	rows := suite.exportedRows(export)
	suite.Require().Len(rows, 1)
	suite.Equal("ID", rows[0][0])
}

// the account comes from the caller's identity, never from the request
func (suite *PartBulkUpsertTestSuite) TestExportParts_ScopesToTheIdentitysAccount() {
	ctx := internalPartCtx("ac_owner")
	suite.partRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params domain.ExportPartsParams) ([]*domain.Part, *apierror.APIError) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportPartsParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func (suite *PartBulkUpsertTestSuite) TestExportParts_RejectsAnIdentitylessContext() {
	export, apiErr := suite.partSvc.ExportParts(context.Background(), domain.ExportPartsParams{})
	suite.Nil(export)
	suite.NotNil(apiErr)
}
