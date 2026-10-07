package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	itemReadSellerID   = "ac_seller"
	itemReadCustomerID = "ac_customer"
)

// itemReadCtx is a user of actorAccount reaching target as relation, holding exactly perms.
func itemReadCtx(relation types.IdentityRelationType, actorAccount, target string, perms ...string) context.Context {
	granted := make(map[string]bool, len(perms))
	for _, p := range perms {
		granted[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: target},
		Actor: &types.IdentityActor{
			RelationType: relation,
			ID:           "us_actor",
			AccountID:    &actorAccount,
			Permissions:  granted,
		},
	})
}

func newItemReadSvc(t *testing.T) (domain.ItemSvc, *repositorymock.MockItemRepo, *mediatormock.MockReadAccessMed) {
	t.Helper()
	ctrl := gomock.NewController(t)
	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	readAccess := mediatormock.NewMockReadAccessMed(ctrl)
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{ReadAccess: readAccess}).AnyTimes()
	return NewItemSvc(&ItemSvcConfig{Repos: repos, MediatorFactory: meds, TxManager: &stubTxManager{factory: repos}}), itemRepo, readAccess
}

// An item record, cost included, is the seller's own: a portal user is refused it whatever permissions it holds, before anything is read.
func TestItemSvc_GetItemRefusesAnyoneButTheSellersOwnUsers(t *testing.T) {
	t.Parallel()

	actors := map[string]context.Context{
		"customer portal user":    itemReadCtx(types.IdentityRelationTypeCustomer, itemReadCustomerID, itemReadSellerID, "items:read"),
		"supplier portal user":    itemReadCtx(types.IdentityRelationTypeSupplier, itemReadCustomerID, itemReadSellerID, "items:read"),
		"seller staff at a buyer": itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadCustomerID, "items:read", "customers:read"),
		"seller staff without it": itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "products:read"),
	}
	for name, ctx := range actors {
		svc, _, _ := newItemReadSvc(t)
		_, _, apiErr := svc.GetItem(ctx, "itm_1", nil, domain.ItemEmbeds{})
		if assert.NotNil(t, apiErr, name) {
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code, name)
		}
	}
}

func TestItemSvc_GetItemReadsTheSellersItem(t *testing.T) {
	t.Parallel()

	includes := []string{"unit_value", "unit_cost", "burn_rate", "attributes"}
	svc, itemRepo, _ := newItemReadSvc(t)
	itemRepo.EXPECT().Get(gomock.Any(), domain.GetItemParams{AccountID: itemReadSellerID, ItemID: "itm_1", Includes: includes}).
		Return(&domain.Item{ID: "itm_1"}, nil)

	item, _, apiErr := svc.GetItem(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read"), "itm_1", includes, domain.ItemEmbeds{})
	require.Nil(t, apiErr)
	assert.Equal(t, "itm_1", item.ID)
}

func TestItemSvc_GetItemNotFoundNamesTheItem(t *testing.T) {
	t.Parallel()

	svc, itemRepo, _ := newItemReadSvc(t)
	itemRepo.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, apierror.NewResourceNotFoundError("Resource not found."))

	_, _, apiErr := svc.GetItem(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read"), "itm_missing", nil, domain.ItemEmbeds{})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)
	assert.Equal(t, "Item not found.", apiErr.PublicMessage)
}

// The include loader's batch read is what lets a portal user see the items on its own orders and invoices.
func TestItemSvc_BatchGetItemsAdmitsACounterpartyResolvingIncludes(t *testing.T) {
	t.Parallel()

	svc, itemRepo, readAccess := newItemReadSvc(t)
	readAccess.EXPECT().CheckCounterpartyReadAccess(gomock.Any(), itemReadCustomerID, itemReadSellerID).Return(nil)
	itemRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"itm_1"}).Return([]*domain.Item{{ID: "itm_1"}}, nil)

	items, _, apiErr := svc.BatchGetItemsByIDs(itemReadCtx(types.IdentityRelationTypeCustomer, itemReadCustomerID, itemReadSellerID), []string{"itm_1"}, domain.ItemEmbeds{})
	require.Nil(t, apiErr)
	require.Len(t, items, 1)
}

// An item's category and the properties its attributes name are part of the item: a role that may read items gets them without item_categories:read or properties:read.
func TestItemSvc_BatchGetItemsEmbedsCategoriesAndAttributePropertiesForAnItemReader(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	itemRepo := repositorymock.NewMockItemRepo(ctrl)
	categoryRepo := repositorymock.NewMockItemCategoryRepo(ctrl)
	propertyRepo := repositorymock.NewMockPropertyRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewItemRepo().Return(itemRepo).AnyTimes()
	repos.EXPECT().NewItemCategoryRepo().Return(categoryRepo).AnyTimes()
	repos.EXPECT().NewPropertyRepo().Return(propertyRepo).AnyTimes()
	meds := factorymock.NewMockMediatorFactory(ctrl)
	meds.EXPECT().Build(gomock.Any()).Return(domain.Mediators{}).AnyTimes()
	svc := NewItemSvc(&ItemSvcConfig{Repos: repos, MediatorFactory: meds, TxManager: &stubTxManager{factory: repos}})

	itemRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"itm_1", "itm_2"}).Return([]*domain.Item{
		{ID: "itm_1", ItemCategoryID: "itcg_socks", Attributes: []*domain.ItemAttribute{{ID: "at_beige", PropertyID: "pp_color"}}},
		{ID: "itm_2", ItemCategoryID: "itcg_socks", Attributes: []*domain.ItemAttribute{{ID: "at_navy", PropertyID: "pp_color"}, {ID: "at_large", PropertyID: "pp_size"}}},
	}, nil)
	categoryRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"itcg_socks"}).
		Return([]*domain.ItemCategoryFull{{ID: "itcg_socks", UnitGroupID: "ungp_pairs"}}, nil)
	categoryRepo.EXPECT().GetPropertiesForCategories(gomock.Any(), []string{"itcg_socks"}).
		Return(map[string][]*domain.ItemCategoryProperty{"itcg_socks": {{ID: "pp_color", Name: "Color"}}}, nil)
	categoryRepo.EXPECT().GetUnitGroups(gomock.Any(), []string{"ungp_pairs"}, gomock.Any()).
		Return(map[string]*domain.ItemCategoryUnitGroup{"ungp_pairs": {ID: "ungp_pairs"}}, nil)
	propertyRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"pp_color", "pp_size"}).
		Return([]*domain.Property{{ID: "pp_color"}, {ID: "pp_size"}}, nil)

	ctx := itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read")
	items, embedded, apiErr := svc.BatchGetItemsByIDs(ctx, []string{"itm_1", "itm_2"}, domain.ItemEmbeds{Categories: true, AttributeProperties: true})
	require.Nil(t, apiErr)
	require.Len(t, items, 2)
	require.Len(t, embedded.Categories, 1)
	assert.Equal(t, "itcg_socks", embedded.Categories[0].ID)
	assert.Equal(t, "Color", embedded.Categories[0].Properties[0].Name)
	assert.Equal(t, "ungp_pairs", embedded.Categories[0].UnitGroup.ID)
	require.Len(t, embedded.AttributeProperties, 2)
}

// Without embeds the read touches nothing beyond the items, and the item permission still gates it.
func TestItemSvc_BatchGetItemsReadsNoEmbedsUnlessAskedAndStillNeedsItemsRead(t *testing.T) {
	t.Parallel()

	svc, itemRepo, _ := newItemReadSvc(t)
	itemRepo.EXPECT().GetByIDs(gomock.Any(), itemReadSellerID, []string{"itm_1"}).
		Return([]*domain.Item{{ID: "itm_1", ItemCategoryID: "itcg_socks", Attributes: []*domain.ItemAttribute{{PropertyID: "pp_color"}}}}, nil)

	_, embedded, apiErr := svc.BatchGetItemsByIDs(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "items:read"), []string{"itm_1"}, domain.ItemEmbeds{})
	require.Nil(t, apiErr)
	assert.Empty(t, embedded.Categories)
	assert.Empty(t, embedded.AttributeProperties)

	svc, _, _ = newItemReadSvc(t)
	_, _, apiErr = svc.BatchGetItemsByIDs(itemReadCtx(types.IdentityRelationTypeInternal, itemReadSellerID, itemReadSellerID, "item_categories:read", "properties:read"), []string{"itm_1"}, domain.ItemEmbeds{Categories: true, AttributeProperties: true})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}
