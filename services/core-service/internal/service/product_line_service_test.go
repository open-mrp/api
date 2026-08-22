package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const productLineSheet = "Product Lines"

type ProductLineSvcTestSuite struct {
	suite.Suite
	svc             domain.ProductLineSvc
	productLineRepo *repositorymock.MockProductLineRepo
	repoFactory     *factorymock.MockRepoFactory
	ctrl            *gomock.Controller
}

func (suite *ProductLineSvcTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.productLineRepo = repositorymock.NewMockProductLineRepo(suite.ctrl)

	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewProductLineRepo().Return(suite.productLineRepo).AnyTimes()

	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()

	suite.svc = NewProductLineSvc(&ProductLineSvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: mediatorFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
		JobSvcFactory:   NewJobSvcFactory(),
	})
}

func (suite *ProductLineSvcTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestProductLineSvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ProductLineSvcTestSuite))
}

// reopens the export so assertions read what a spreadsheet would
func (suite *ProductLineSvcTestSuite) rows(export *domain.Export) [][]string {
	return exportedSheetRows(suite.T(), export, productLineSheet)
}

// renders the workbook the export consumer would build, for the account the caller is
// acting for. The job machinery around it belongs to the engine.
func (suite *ProductLineSvcTestSuite) buildExport(ctx context.Context, params domain.ExportProductLinesParams) (*domain.Export, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	suite.Require().True(ok)
	impl := suite.svc.(*productLineSvcImpl)
	spec := impl.exportSpec()
	// Accept narrows before it stores, so the consumer builds from filters the
	// caller could not have widened.
	if spec.NarrowFilters != nil {
		params = spec.NarrowFilters(identity, params)
	}
	return buildExport(ctx, suite.repoFactory, spec, identity.Target.AccountID, params)
}

func (suite *ProductLineSvcTestSuite) TestExport_WritesHeadersAndRows() {
	ctx := internalIdentityCtx("ac_test123")
	suite.productLineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return([]*domain.ProductLineFull{
		{
			ID:               "prl_1",
			Name:             "Performance Knits",
			CommissionPolicy: constants.CommissionPolicyExempt,
			FreightPolicy:    constants.FreightPolicyFromBool(false),
			UnitGroup:        &domain.ProductLineUnitGroup{Name: "Mass"},
		},
		{
			ID:               "prl_2",
			Name:             "Base Layer",
			CommissionPolicy: constants.CommissionPolicyFromBool(false),
			FreightPolicy:    constants.FreightPolicyFromBool(true),
		},
	}, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportProductLinesParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)

	rows := suite.rows(export)
	suite.Require().Len(rows, 3)
	// No ID column here: the export and the import template carry the same columns.
	suite.Equal([]string{"Name", "Unit Group", "Commission Exempt", "Freight Exempt"}, rows[0])
	suite.Equal([]string{"Performance Knits", "Mass", "Yes", "No"}, rows[1])
	// A line with no unit group still exports; the cell is simply blank.
	suite.Equal([]string{"Base Layer", "", "No", "Yes"}, rows[2])
}

// an account with nothing to export still gets a usable, header-only workbook
func (suite *ProductLineSvcTestSuite) TestExport_EmptyAccountYieldsHeaderOnlyFile() {
	ctx := internalIdentityCtx("ac_test123")
	suite.productLineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).Return(nil, nil)

	export, apiErr := suite.buildExport(ctx, domain.ExportProductLinesParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(0), export.RowCount)

	rows := suite.rows(export)
	suite.Require().Len(rows, 1)
	suite.Equal("Name", rows[0][0])
}

// the account comes from the caller's identity, never from the request
func (suite *ProductLineSvcTestSuite) TestExport_ScopesToTheIdentitysAccount() {
	ctx := internalIdentityCtx("ac_owner")
	suite.productLineRepo.EXPECT().Export(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, params domain.ExportProductLinesParams) ([]*domain.ProductLineFull, error) {
			suite.Equal("ac_owner", params.AccountID)
			return nil, nil
		})

	_, apiErr := suite.buildExport(ctx, domain.ExportProductLinesParams{AccountID: "ac_attacker"})
	suite.Require().Nil(apiErr)
}

func (suite *ProductLineSvcTestSuite) TestExport_RejectsAnIdentitylessContext() {
	export, apiErr := suite.svc.ExportProductLines(suite.T().Context(), domain.ExportProductLinesParams{})
	suite.Nil(export)
	suite.NotNil(apiErr)
}
