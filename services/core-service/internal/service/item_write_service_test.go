package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/apperr"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

type itemWriteRepos struct {
	factory    *factorymock.MockRepoFactory
	items      *repositorymock.MockItemRepo
	attributes *repositorymock.MockAttributeRepo
	refs       *repositorymock.MockReferenceRepo
	deleted    *repositorymock.MockDeletedRecordRepo
	categories *repositorymock.MockItemCategoryRepo
}

func newItemWriteRepos(t *testing.T) itemWriteRepos {
	t.Helper()
	ctrl := gomock.NewController(t)
	r := itemWriteRepos{
		factory:    factorymock.NewMockRepoFactory(ctrl),
		items:      repositorymock.NewMockItemRepo(ctrl),
		attributes: repositorymock.NewMockAttributeRepo(ctrl),
		refs:       repositorymock.NewMockReferenceRepo(ctrl),
		deleted:    repositorymock.NewMockDeletedRecordRepo(ctrl),
		categories: repositorymock.NewMockItemCategoryRepo(ctrl),
	}
	r.factory.EXPECT().NewItemRepo().Return(r.items).AnyTimes()
	r.factory.EXPECT().NewAttributeRepo().Return(r.attributes).AnyTimes()
	r.factory.EXPECT().NewReferenceRepo().Return(r.refs).AnyTimes()
	r.factory.EXPECT().NewDeletedRecordRepo().Return(r.deleted).AnyTimes()
	r.factory.EXPECT().NewItemCategoryRepo().Return(r.categories).AnyTimes()
	return r
}

func requireFieldError(t *testing.T, apiErr *apierror.APIError, code apierror.Code, param string) {
	t.Helper()
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.Status())
	require.NotEmpty(t, apiErr.Errors)
	assert.Equal(t, code, apiErr.Errors[0].Code)
	assert.Equal(t, param, apiErr.Param)
}

func TestCreateItem_MaterialSectionOnlyOnMaterials(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	_, apiErr := createItemInTx(context.Background(), r.factory, domain.CreateItemParams{
		AccountID: "ac_1", Type: constants.ItemTypeCodeProduct, SKU: "P-1", CategoryID: "itcg_1",
		Material: &domain.ItemMaterialInput{},
	})
	requireFieldError(t, apiErr, apierror.CodeInvalidFormat, "material")
}

func TestCreateItem_CategoryMustSuitTheType(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.items.EXPECT().CheckSKUExists(gomock.Any(), "ac_1", "SVC-1", "").Return(false, nil)
	r.categories.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"itcg_mat"}).Return([]*domain.ItemCategoryFull{{ID: "itcg_mat"}}, nil)
	r.items.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "itcg_mat").Return("each", constants.ItemCategoryTypeMaterial.Code(), nil)

	_, apiErr := createItemInTx(context.Background(), r.factory, domain.CreateItemParams{
		AccountID: "ac_1", Type: constants.ItemTypeCodeService, SKU: "SVC-1", CategoryID: "itcg_mat",
	})
	requireFieldError(t, apiErr, apierror.CodeInvalidFormat, "category_id")
}

func TestCreateItem_DuplicateSKUConflicts(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.items.EXPECT().CheckSKUExists(gomock.Any(), "ac_1", "DUP", "").Return(true, nil)

	_, apiErr := createItemInTx(context.Background(), r.factory, domain.CreateItemParams{
		AccountID: "ac_1", Type: constants.ItemTypeCodePart, SKU: "DUP", CategoryID: "itcg_1",
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Status())
	assert.Equal(t, "sku", apiErr.Param)
}

func TestValidateItemAttributes_OnePerProperty(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.attributes.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"at_red", "at_blue"}).Return([]*domain.Attribute{
		{ID: "at_red", PropertyID: "pp_color", IsActive: true},
		{ID: "at_blue", PropertyID: "pp_color", IsActive: true},
	}, nil)

	apiErr := validateItemAttributes(context.Background(), r.factory, "ac_1", "itcg_1", []string{"at_red", "at_blue", "at_red"}, "attribute_ids")
	requireFieldError(t, apiErr, apierror.CodeInvalidFormat, "attribute_ids")
}

func TestValidateItemAttributes_EveryAttributeMustExist(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.attributes.EXPECT().GetByIDs(gomock.Any(), "ac_1", []string{"at_gone"}).Return(nil, nil)

	apiErr := validateItemAttributes(context.Background(), r.factory, "ac_1", "itcg_1", []string{"at_gone"}, "attribute_ids")
	requireFieldError(t, apiErr, apierror.CodeInvalidFormat, "attribute_ids")
}

func TestUpdateItem_MaterialSectionOnlyOnMaterials(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.items.EXPECT().GetByIDsWithIncludes(gomock.Any(), "ac_1", []string{"itm_part"}, domain.ItemReadIncludes).
		Return([]*domain.Item{{ID: "itm_part", ItemTypeCode: string(constants.ItemTypeCodePart)}}, nil)

	_, apiErr := updateItemInTx(context.Background(), r.factory, domain.UpdateItemParams{
		AccountID: "ac_1", ItemID: "itm_part", Material: &domain.ItemMaterialInput{},
	})
	requireFieldError(t, apiErr, apierror.CodeInvalidFormat, "material")
}

// A material's suppliers reference its material row, so they are probed by the material id and reported with the
// item's other uses.
func TestDeleteItem_InUseNamesEveryReference(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	item := &domain.Item{ID: "itm_mat", AccountID: "ac_1", ItemTypeCode: string(constants.ItemTypeCodeMaterial),
		MaterialSection: &domain.ItemMaterialSection{MaterialID: "ml_1"}}
	r.items.EXPECT().GetByIDsWithIncludes(gomock.Any(), "ac_1", []string{"itm_mat"}, domain.ItemReadIncludes).Return([]*domain.Item{item}, nil)
	r.refs.EXPECT().InUseBy(gomock.Any(), "itm_mat", domain.ItemReferences.Probes).Return([]string{"orders"}, nil)
	r.refs.EXPECT().InUseBy(gomock.Any(), "ml_1", domain.ItemMaterialReferences.Probes).Return([]string{"supplier materials"}, nil)

	_, apiErr := deleteItemInTx(context.Background(), r.factory, "ac_1", "itm_mat")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apiErr.Status())
	assert.Contains(t, apiErr.PublicMessage, "orders and supplier materials")
	assert.Contains(t, apiErr.PublicMessage, "`archived`")
}

func TestDeleteItem_UnusedIsHardDeleted(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	item := &domain.Item{ID: "itm_free", AccountID: "ac_1", ItemTypeCode: string(constants.ItemTypeCodePart)}
	r.items.EXPECT().GetByIDsWithIncludes(gomock.Any(), "ac_1", []string{"itm_free"}, domain.ItemReadIncludes).Return([]*domain.Item{item}, nil)
	r.refs.EXPECT().InUseBy(gomock.Any(), "itm_free", domain.ItemReferences.Probes).Return(nil, nil)
	gomock.InOrder(
		r.deleted.EXPECT().CreateInAccount(gomock.Any(), constants.DeletedRecordResourceTypeItem, "itm_free", "ac_1", item).Return(nil),
		r.items.EXPECT().HardDelete(gomock.Any(), item).Return(nil),
	)
	r.factory.EXPECT().NewOutboxRepo().Return(&productStubOutboxRepo{}).AnyTimes()

	deleted, apiErr := deleteItemInTx(internalProductIdentityCtx("ac_1"), r.factory, "ac_1", "itm_free")
	require.Nil(t, apiErr)
	assert.Same(t, item, deleted)
}

func TestDeleteItem_ReplayIsGone(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.items.EXPECT().GetByIDsWithIncludes(gomock.Any(), "ac_1", []string{"itm_gone"}, domain.ItemReadIncludes).Return(nil, nil)
	r.deleted.EXPECT().ExistsInAccount(gomock.Any(), constants.DeletedRecordResourceTypeItem, "itm_gone", "ac_1").Return(true, nil)

	_, apiErr := deleteItemInTx(context.Background(), r.factory, "ac_1", "itm_gone")
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusGone, apiErr.Status())
}

func TestRequireItemsAssignable_RefusesAnArchivedItem(t *testing.T) {
	t.Parallel()

	r := newItemWriteRepos(t)
	r.items.EXPECT().GetAssignability(gomock.Any(), "ac_1", []string{"itm_a", "itm_b"}).Return([]domain.ItemAssignability{
		{ItemID: "itm_a", IsAssignable: true},
		{ItemID: "itm_b", IsAssignable: false},
	}, nil)

	apiErr := requireItemsAssignable(context.Background(), r.factory, "ac_1", []string{"itm_a", "itm_b", "itm_a"}, "lines")
	requireFieldError(t, apiErr, apperr.CodeResourceArchived, "lines")
}

func TestItemIsSold(t *testing.T) {
	t.Parallel()

	price := func(amount string) *domain.UnitPrice {
		return &domain.UnitPrice{Amount: amount, Currency: constants.CurrencyUSD}
	}
	cases := map[string]struct {
		item domain.Item
		want bool
	}{
		"product without a loaded row":   {domain.Item{ItemTypeCode: "product"}, true},
		"service":                        {domain.Item{ItemTypeCode: "service"}, true},
		"material with a product row":    {domain.Item{ItemTypeCode: "material", Sales: &domain.ItemSales{}}, true},
		"material with a selling price":  {domain.Item{ItemTypeCode: "material", SalesUnitPrice: price("4.25")}, true},
		"material priced at zero":        {domain.Item{ItemTypeCode: "material", SalesUnitPrice: price("0.000000")}, false},
		"part with no price read":        {domain.Item{ItemTypeCode: "part"}, false},
		"part with an unparsable amount": {domain.Item{ItemTypeCode: "part", SalesUnitPrice: price("n/a")}, false},
	}
	for name, tc := range cases {
		assert.Equal(t, tc.want, tc.item.IsSold(), name)
	}
}

func TestItemStatus(t *testing.T) {
	t.Parallel()

	deletedAt := time.Now()
	assert.Equal(t, constants.ArchiveStatusActive, domain.ItemStatus(true, nil))
	assert.Equal(t, constants.ArchiveStatusArchived, domain.ItemStatus(false, nil))
	assert.Equal(t, constants.ArchiveStatusArchived, domain.ItemStatus(true, &deletedAt), "a soft-deleted item reads as archived")
}
