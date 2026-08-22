package service

import (
	"context"
	"errors"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// --- PropertySvcTestSuite ---

type PropertySvcTestSuite struct {
	suite.Suite
	propertySvc     domain.PropertySvc
	propertyRepo    *repositorymock.MockPropertyRepo
	attributeRepo   *repositorymock.MockAttributeRepo
	repoFactory     *factorymock.MockRepoFactory
	mediatorFactory *factorymock.MockMediatorFactory
	idempotencyMed  *mediatormock.MockIdempotencyMed
	ctrl            *gomock.Controller
}

func (suite *PropertySvcTestSuite) SetupSuite() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.propertyRepo = repositorymock.NewMockPropertyRepo(suite.ctrl)
	suite.attributeRepo = repositorymock.NewMockAttributeRepo(suite.ctrl)
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewPropertyRepo().Return(suite.propertyRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewAttributeRepo().Return(suite.attributeRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(&stubOutboxRepo{}).AnyTimes()
	suite.repoFactory.EXPECT().NewDeletedRecordRepo().Return(repositorymock.NewMockDeletedRecordRepo(suite.ctrl)).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	suite.mediatorFactory = factorymock.NewMockMediatorFactory(suite.ctrl)
	suite.mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{
		Idempotency: suite.idempotencyMed,
	}).AnyTimes()

	suite.propertySvc = NewPropertySvc(&PropertySvcConfig{
		Repos:           suite.repoFactory,
		MediatorFactory: suite.mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	})
}

func (suite *PropertySvcTestSuite) TearDownSuite() {
	suite.ctrl.Finish()
}

func TestPropertySvcTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PropertySvcTestSuite))
}

// --- identity helpers ---

func internalPropertyCtx(targetAccountID string) context.Context {
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
				"properties:read":   true,
				"properties:create": true,
				"properties:update": true,
				"properties:delete": true,
			},
		},
	})
}

func readOnlyPropertyCtx(targetAccountID string) context.Context {
	customCode := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: targetAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			AccountID:    &targetAccountID,
			RoleType:     &customCode,
			Permissions: map[string]bool{
				"properties:read": true,
			},
		},
	})
}

// --- BulkUpsertProperties: accept-phase guard checks ---

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_MissingIdentity() {
	result, err := suite.propertySvc.BulkUpsertProperties(context.Background(), domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{{Name: "Color"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_InsufficientPermissions() {
	ctx := readOnlyPropertyCtx("ac_test123")

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{{Name: "Color"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInsufficientPerms, err.Code)
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_MissingTargetAccount() {
	customCode := string(constants.RoleTypeCustom)
	ctx := appctx.WithIdentity(context.Background(), &types.Identity{
		Type: types.IdentityActorTypeUser,
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_test123",
			RoleType:     &customCode,
			Permissions: map[string]bool{
				"properties:read":   true,
				"properties:create": true,
			},
		},
	})

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{{Name: "Color"}},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInvalidCredentials, err.Code)
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_Empty() {
	ctx := internalPropertyCtx("ac_test123")

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "No properties provided")
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_TooMany() {
	ctx := internalPropertyCtx("ac_test123")
	props := make([]domain.UpsertPropertyParams, 1001)

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: props,
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, "1000")
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_DuplicateNameInRequest() {
	ctx := internalPropertyCtx("ac_test123")

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{
			{Name: "Color"},
			{Name: "color"}, // same name case-insensitive
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, `duplicate name "color" in request`)
	suite.Equal("properties[1].name", err.Param)
}

// One value belongs to one property per account, so no job is raised at all
func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_RejectsValueClaimedByTwoProperties() {
	ctx := internalPropertyCtx("ac_test123")

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{
			{Name: "Color", Attributes: []domain.UpsertPropertyAttributeParams{{Value: "Red"}}},
			{Name: "Accent", Attributes: []domain.UpsertPropertyAttributeParams{{Value: "red"}}},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, `already used under property "Color"`)
	suite.Equal("properties[1].attributes[0].value", err.Param)
}

func (suite *PropertySvcTestSuite) TestBulkUpsertProperties_RejectsDuplicateValueWithinAProperty() {
	ctx := internalPropertyCtx("ac_test123")

	result, err := suite.propertySvc.BulkUpsertProperties(ctx, domain.BulkUpsertPropertiesParams{
		Properties: []domain.UpsertPropertyParams{
			{Name: "Color", Attributes: []domain.UpsertPropertyAttributeParams{{Value: "Red"}, {Value: " red "}}},
		},
	})

	suite.Nil(result)
	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeValidationFailed, err.Code)
	suite.Contains(err.PublicMessage, `duplicate value`)
}

// --- resolveBulkUpsertPropertyRows (the engine's Resolve hook) ---

// An omitted color is derived from the (property, value) pair, so a redelivery re-derives it
func (suite *PropertySvcTestSuite) TestResolveBulkUpsertPropertyRows_SettlesEverySwatch() {
	explicit := string(constants.ColorBlue)

	rows, err := resolveBulkUpsertPropertyRows(context.Background(), suite.repoFactory, "ac_test123", []domain.UpsertPropertyParams{
		{Name: "  Color  ", Attributes: []domain.UpsertPropertyAttributeParams{
			{Value: " Red "},
			{Value: "Blue", ColorCode: &explicit},
		}},
	})

	suite.Nil(err)
	suite.Require().Len(rows, 1)
	suite.Equal("Color", rows[0].Name)
	suite.Require().Len(rows[0].Attributes, 2)
	suite.Equal("Red", rows[0].Attributes[0].Value)
	suite.NotEmpty(rows[0].Attributes[0].ColorCode)
	suite.Equal(attributeColorFor("Color", "Red"), rows[0].Attributes[0].ColorCode)
	suite.Equal(explicit, rows[0].Attributes[1].ColorCode)
}

// --- writeBulkUpsertProperties (the engine's Write hook, exercised directly) ---

func (suite *PropertySvcTestSuite) writeProperties(rows ...domain.ResolvedUpsertPropertyRow) (created, updated []string, rowErrs []apierror.RowError, apiErr *apierror.APIError) {
	res, apiErr := writeBulkUpsertProperties(internalPropertyCtx("ac_test123"), suite.repoFactory, passthroughSavepoint{}, "ac_test123", rows)
	if apiErr != nil {
		return nil, nil, nil, apiErr
	}
	created, updated = splitJobResults(res.Results)
	return created, updated, res.Errors, nil
}

func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_CreateWithoutAttributes() {
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), domain.CreatePropertyParams{AccountID: "ac_test123", Name: "Color"}).
		Return(&domain.Property{ID: "pp_color1", Name: "Color", AccountID: "ac_test123"}, nil).Times(1)

	created, updated, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "Color"},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"pp_color1"}, created)
	suite.Empty(updated)
}

func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_CreateWithAttributes() {
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)
	suite.attributeRepo.EXPECT().
		FindByTextsInAccount(gomock.Any(), "ac_test123", gomock.InAnyOrder([]string{"Red", "Blue"})).
		Return(nil, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Property{ID: "pp_color2", Name: "Color", AccountID: "ac_test123"}, nil).Times(1)

	suite.attributeRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), domain.CreateAttributeParams{
			Value: "Red", PropertyID: "pp_color2", AccountID: "ac_test123", ColorCode: "red", SortOrder: 1,
		}).Return(&domain.Attribute{ID: "at_red", Value: "Red", PropertyID: "pp_color2"}, nil).Times(1)
	suite.attributeRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), domain.CreateAttributeParams{
			Value: "Blue", PropertyID: "pp_color2", AccountID: "ac_test123", ColorCode: "blue", SortOrder: 2,
		}).Return(&domain.Attribute{ID: "at_blue", Value: "Blue", PropertyID: "pp_color2"}, nil).Times(1)

	created, _, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "Color", Attributes: []domain.ResolvedUpsertPropertyAttribute{
			{Value: "Red", ColorCode: "red"},
			{Value: "Blue", ColorCode: "blue"},
		}},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Equal([]string{"pp_color2"}, created)
}

// Matched case-insensitively, renamed to the request's casing, missing attributes added
func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_UpdateAddsOnlyMissingAttributes() {
	existing := &domain.Property{ID: "pp_existing", Name: "Color", AccountID: "ac_test123"}

	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.InAnyOrder([]string{"color"})).
		Return([]*domain.Property{existing}, nil).Times(1)
	suite.attributeRepo.EXPECT().
		ListByPropertyIDs(gomock.Any(), "ac_test123", []string{"pp_existing"}).
		Return([]*domain.Attribute{{ID: "at_red", Value: "Red", PropertyID: "pp_existing", SortOrder: 1}}, nil).Times(1)
	suite.attributeRepo.EXPECT().
		FindByTextsInAccount(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.AttributeTextMatch{{ID: "at_red", Text: "Red", PropertyID: "pp_existing", PropertyName: "Color"}}, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(&domain.Property{ID: "pp_existing", Name: "COLOR", AccountID: "ac_test123"}, nil).Times(1)

	// "Red" is already defined, so only "Blue" is written.
	suite.attributeRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), domain.CreateAttributeParams{
			Value: "Blue", PropertyID: "pp_existing", AccountID: "ac_test123", ColorCode: "blue", SortOrder: 2,
		}).Return(&domain.Attribute{ID: "at_blue", Value: "Blue", PropertyID: "pp_existing"}, nil).Times(1)

	created, updated, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "COLOR", Attributes: []domain.ResolvedUpsertPropertyAttribute{
			{Value: "Red", ColorCode: "red"},
			{Value: "Blue", ColorCode: "blue"},
		}},
	)

	suite.Nil(err)
	suite.Empty(rowErrs)
	suite.Empty(created)
	suite.Equal([]string{"pp_existing"}, updated)
}

// A value another property owns fails that row alone; the rest of the batch commits
func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_ValueOwnedByAnotherPropertyFailsOnlyItsRow() {
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)
	suite.attributeRepo.EXPECT().
		FindByTextsInAccount(gomock.Any(), "ac_test123", gomock.Any()).
		Return([]*domain.AttributeTextMatch{{ID: "at_red", Text: "Red", PropertyID: "pp_other", PropertyName: "Accent"}}, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Property{ID: "pp_color3", Name: "Color", AccountID: "ac_test123"}, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Property{ID: "pp_size3", Name: "Size", AccountID: "ac_test123"}, nil).Times(1)

	created, _, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "Color", Attributes: []domain.ResolvedUpsertPropertyAttribute{
			{Value: "Red", ColorCode: "red"},
		}},
		domain.ResolvedUpsertPropertyRow{Name: "Size"},
	)

	suite.Nil(err)
	suite.Require().Len(rowErrs, 1)
	suite.Equal(0, rowErrs[0].Index)
	suite.Equal("properties[0].attributes[0].value", *rowErrs[0].Error.Param)
	suite.Equal([]string{"pp_size3"}, created)
}

// The bulk read is infrastructure, not a row, so its failure sinks the whole batch
func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_FindByNamesErrorSinksTheBatch() {
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("db error"), "query failed")).Times(1)

	created, updated, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "Color"},
	)

	suite.NotNil(err)
	suite.Equal(apierror.ErrorCodeInternalError, err.Code)
	suite.Empty(created)
	suite.Empty(updated)
	suite.Empty(rowErrs)
}

func (suite *PropertySvcTestSuite) TestWriteBulkUpsertProperties_CreateErrorFailsOnlyItsRow() {
	suite.propertyRepo.EXPECT().
		FindByNames(gomock.Any(), "ac_test123", gomock.Any()).
		Return(nil, nil).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, apierror.NewInternalError(errors.New("insert failed"), "db error")).Times(1)
	suite.propertyRepo.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.Property{ID: "pp_size4", Name: "Size", AccountID: "ac_test123"}, nil).Times(1)

	created, _, rowErrs, err := suite.writeProperties(
		domain.ResolvedUpsertPropertyRow{Name: "Color"},
		domain.ResolvedUpsertPropertyRow{Name: "Size"},
	)

	suite.Nil(err)
	suite.Require().Len(rowErrs, 1)
	suite.Equal(0, rowErrs[0].Index)
	suite.Equal([]string{"pp_size4"}, created)
}

// --- Export column derivation ---

// builds an item in a category that declares the given properties in that order,
// with a value for each named in values
func exportItemWithProperties(categoryName string, properties []domain.ItemCategoryProperty, values map[string]string) *domain.Item {
	category := &domain.ItemCategory{Name: categoryName, Properties: properties}
	attributes := []*domain.ItemAttribute{}
	for _, property := range properties {
		if value, ok := values[property.Name]; ok {
			attributes = append(attributes, &domain.ItemAttribute{PropertyID: property.ID, Value: value})
		}
	}
	return &domain.Item{CategoryName: categoryName, Category: category, Attributes: attributes}
}

// reads the headers off a derived column list
func columnHeaders(columns []excel.ColumnSpec) []string {
	headers := make([]string, len(columns))
	for i, column := range columns {
		headers[i] = column.Header
	}
	return headers
}

func TestItemPropertyColumns_OrdersByNameNotByEncounter(t *testing.T) {
	t.Parallel()
	items := []*domain.Item{
		// Declared in reverse alphabetical order, so an unsorted result cannot pass.
		exportItemWithProperties("Yarn", []domain.ItemCategoryProperty{
			{ID: "pp_1", Name: "Weight"}, {ID: "pp_2", Name: "Fibre"},
		}, nil),
		exportItemWithProperties("Trim", []domain.ItemCategoryProperty{{ID: "pp_3", Name: "Colour"}}, nil),
	}

	// Encounter order depends on which rows the query returned; sorting is what keeps
	// two exports of the same data byte-identical.
	assert.Equal(t, []string{"Colour", "Fibre", "Weight"}, columnHeaders(itemPropertyColumns(items)))
}

// the header is the importer's key, so two categories declaring the same property
// name must share one column rather than producing a duplicate header
func TestItemPropertyColumns_CollapsesTheSameNameAcrossCategories(t *testing.T) {
	t.Parallel()
	items := []*domain.Item{
		exportItemWithProperties("Yarn", []domain.ItemCategoryProperty{{ID: "pp_yarn_fibre", Name: "Fibre"}}, nil),
		exportItemWithProperties("Trim", []domain.ItemCategoryProperty{{ID: "pp_trim_fibre", Name: "Fibre"}}, nil),
	}

	assert.Equal(t, []string{"Fibre"}, columnHeaders(itemPropertyColumns(items)))
}

// a category declares the property, so the column exists even where no item filled it
func TestItemPropertyColumns_IncludesAPropertyNoItemHasAValueFor(t *testing.T) {
	t.Parallel()
	items := []*domain.Item{
		exportItemWithProperties("Yarn", []domain.ItemCategoryProperty{
			{ID: "pp_1", Name: "Fibre"}, {ID: "pp_2", Name: "Weight"},
		}, map[string]string{"Fibre": "Merino"}),
	}

	assert.Equal(t, []string{"Fibre", "Weight"}, columnHeaders(itemPropertyColumns(items)))
}

func TestItemPropertyColumns_SkipsUnnamedPropertiesAndAbsentCategories(t *testing.T) {
	t.Parallel()
	items := []*domain.Item{
		exportItemWithProperties("Yarn", []domain.ItemCategoryProperty{
			{ID: "pp_1", Name: "  "}, {ID: "pp_2", Name: "Fibre"},
		}, nil),
		{CategoryName: "Uncategorised"}, // no category loaded
		nil,
	}

	assert.Equal(t, []string{"Fibre"}, columnHeaders(itemPropertyColumns(items)))
}

// two items in different categories whose properties share a name write into the
// one shared column, which is what makes the exported file re-importable
func TestAddItemPropertyCells_WritesEachValueUnderItsOwnName(t *testing.T) {
	t.Parallel()
	yarn := exportItemWithProperties("Yarn",
		[]domain.ItemCategoryProperty{{ID: "pp_yarn_fibre", Name: "Fibre"}, {ID: "pp_yarn_weight", Name: "Weight"}},
		map[string]string{"Fibre": "Merino", "Weight": "Worsted"})
	trim := exportItemWithProperties("Trim",
		[]domain.ItemCategoryProperty{{ID: "pp_trim_fibre", Name: "Fibre"}},
		map[string]string{"Fibre": "Nylon"})

	yarnRow, trimRow := excel.Row{}, excel.Row{}
	addItemPropertyCells(yarnRow, yarn)
	addItemPropertyCells(trimRow, trim)

	assert.Equal(t, "Merino", yarnRow[propertyKeyPrefix+"Fibre"])
	assert.Equal(t, "Worsted", yarnRow[propertyKeyPrefix+"Weight"])
	assert.Equal(t, "Nylon", trimRow[propertyKeyPrefix+"Fibre"])
	// The trim item has no Weight property, so the cell is absent and the sheet blank.
	assert.NotContains(t, trimRow, propertyKeyPrefix+"Weight")
}

// an attribute whose property is not on the item's category has no column to land
// in; writing it anyway would put a value under someone else's header
func TestAddItemPropertyCells_IgnoresAnAttributeOffTheCategory(t *testing.T) {
	t.Parallel()
	item := exportItemWithProperties("Yarn", []domain.ItemCategoryProperty{{ID: "pp_1", Name: "Fibre"}}, map[string]string{"Fibre": "Merino"})
	item.Attributes = append(item.Attributes, &domain.ItemAttribute{PropertyID: "pp_stale", Value: "Orphan"})

	row := excel.Row{}
	addItemPropertyCells(row, item)

	assert.Len(t, row, 1)
	assert.Equal(t, "Merino", row[propertyKeyPrefix+"Fibre"])
}
