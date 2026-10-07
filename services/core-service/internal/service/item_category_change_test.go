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
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type categoryChangeRepos struct {
	factory  *factorymock.MockRepoFactory
	items    *repositorymock.MockItemRepo
	category *repositorymock.MockItemCategoryRepo
	parts    *repositorymock.MockPartRepo
	outbox   *capturingOutboxRepo
}

func newCategoryChangeRepos(t *testing.T) *categoryChangeRepos {
	t.Helper()
	ctrl := gomock.NewController(t)
	r := &categoryChangeRepos{
		factory:  factorymock.NewMockRepoFactory(ctrl),
		items:    repositorymock.NewMockItemRepo(ctrl),
		category: repositorymock.NewMockItemCategoryRepo(ctrl),
		parts:    repositorymock.NewMockPartRepo(ctrl),
		outbox:   &capturingOutboxRepo{},
	}
	r.factory.EXPECT().NewItemRepo().Return(r.items).AnyTimes()
	r.factory.EXPECT().NewItemCategoryRepo().Return(r.category).AnyTimes()
	r.factory.EXPECT().NewPartRepo().Return(r.parts).AnyTimes()
	r.factory.EXPECT().NewOutboxRepo().Return(r.outbox).AnyTimes()
	return r
}

// auditedResources lists the audit events written to the outbox as "action resource_type resource_id".
func (r *categoryChangeRepos) auditedResources(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, msg := range r.outbox.messages {
		var payload struct {
			Action       string `json:"action"`
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
		}
		require.NoError(t, json.Unmarshal(msg.Payload.Data, &payload))
		out = append(out, payload.Action+" "+payload.ResourceType+" "+payload.ResourceID)
	}
	return out
}

// partsWriterCtx is a non-admin user holding parts:update and nothing on items.
func partsWriterCtx(accountID string) context.Context {
	roleType := string(constants.RoleTypeCustom)
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_parts",
			AccountID:    &accountID,
			RoleType:     &roleType,
			Permissions:  map[string]bool{"parts:update": true},
		},
	})
}

func partItem(categoryID, categoryName string) *domain.Item {
	return &domain.Item{ID: "itm_part", ItemTypeCode: string(constants.ItemTypeCodePart), ItemCategoryID: categoryID, CategoryName: categoryName}
}

func TestChangeItemCategoryInTx_MovesTheItemAndItsUnits(t *testing.T) {
	t.Parallel()
	r := newCategoryChangeRepos(t)
	ctx := partsWriterCtx("ac_cat")

	gomock.InOrder(
		r.items.EXPECT().Get(gomock.Any(), domain.GetItemParams{AccountID: "ac_cat", ItemID: "itm_part", Includes: []string{"attributes"}}).Return(partItem("ic_old", "Old"), nil),
		r.category.EXPECT().Get(gomock.Any(), domain.GetItemCategoryParams{AccountID: "ac_cat", ItemCategoryID: "ic_new"}).
			Return(&domain.ItemCategoryFull{ID: "ic_new", ItemCategoryTypeCode: string(constants.ItemCategoryTypeProduct)}, nil),
		r.items.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "ic_new").Return("un_pair", "", nil),
		r.items.EXPECT().ChangeCategory(gomock.Any(), domain.ChangeItemCategoryParams{AccountID: "ac_cat", ItemID: "itm_part", CategoryID: "ic_new"}).Return(nil),
		r.items.EXPECT().UpdateRateUnits(gomock.Any(), "ac_cat", "itm_part", "un_pair").Return(nil),
		r.items.EXPECT().UpdateMaterialOrderPointUnit(gomock.Any(), "ac_cat", "itm_part", "un_pair").Return(nil),
		r.items.EXPECT().UpdateConsumptionProductionQuantityUnits(gomock.Any(), "ac_cat", "itm_part", "un_pair").Return(nil),
		r.items.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partItem("ic_new", "New"), nil),
	)

	moved, apiErr := changeItemCategoryInTx(ctx, r.factory, "ac_cat", "itm_part", "ic_new", nil)

	require.Nil(t, apiErr)
	assert.Equal(t, "ic_new", moved.ItemCategoryID)
	assert.Equal(t, []string{"update item itm_part"}, r.auditedResources(t))
}

func TestChangeItemCategoryInTx_CurrentCategoryPublishesNothing(t *testing.T) {
	t.Parallel()
	r := newCategoryChangeRepos(t)

	r.items.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partItem("ic_same", "Same"), nil).Times(2)
	r.category.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ItemCategoryFull{ID: "ic_same", ItemCategoryTypeCode: string(constants.ItemCategoryTypeProduct)}, nil)
	r.items.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "ic_same").Return("un_each", "", nil)
	r.items.EXPECT().ChangeCategory(gomock.Any(), gomock.Any()).Return(nil)
	r.items.EXPECT().UpdateRateUnits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	r.items.EXPECT().UpdateMaterialOrderPointUnit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	r.items.EXPECT().UpdateConsumptionProductionQuantityUnits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	_, apiErr := changeItemCategoryInTx(partsWriterCtx("ac_cat"), r.factory, "ac_cat", "itm_part", "ic_same", nil)

	require.Nil(t, apiErr)
	assert.Empty(t, r.auditedResources(t))
}

// A refused move writes nothing: the mocks fail the test on any write they were not told to expect.
func TestChangeItemCategoryInTx_RefusesWithoutWriting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		item     *domain.Item
		category *domain.ItemCategoryFull
		catErr   *apierror.APIError
		props    []*domain.ItemCategoryProperty
		wantCode apierror.ErrorCode
		param    string
	}{
		{
			name:     "a part to a material category",
			item:     partItem("ic_old", "Old"),
			category: &domain.ItemCategoryFull{ID: "ic_new", ItemCategoryTypeCode: string(constants.ItemCategoryTypeMaterial)},
			wantCode: apierror.ErrorCodeValidationFailed,
			param:    "category_id",
		},
		{
			name: "a category that strands an attribute",
			item: &domain.Item{ID: "itm_part", ItemTypeCode: string(constants.ItemTypeCodePart), ItemCategoryID: "ic_old",
				Attributes: []*domain.ItemAttribute{{ID: "at_red", Value: "Red", PropertyID: "pp_color"}}},
			category: &domain.ItemCategoryFull{ID: "ic_new", ItemCategoryTypeCode: string(constants.ItemCategoryTypeProduct)},
			props:    []*domain.ItemCategoryProperty{{ID: "pp_size"}},
			wantCode: apierror.ErrorCodeValidationFailed,
			param:    "category_id",
		},
		{
			name:     "a category outside the account",
			item:     partItem("ic_old", "Old"),
			catErr:   apierror.NewResourceNotFoundError("Resource not found."),
			wantCode: apierror.ErrorCodeResourceNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newCategoryChangeRepos(t)
			r.items.EXPECT().Get(gomock.Any(), gomock.Any()).Return(tt.item, nil)
			r.category.EXPECT().Get(gomock.Any(), gomock.Any()).Return(tt.category, tt.catErr)
			r.category.EXPECT().GetProperties(gomock.Any(), "ic_new").Return(tt.props, nil).AnyTimes()

			_, apiErr := changeItemCategoryInTx(partsWriterCtx("ac_cat"), r.factory, "ac_cat", "itm_part", "ic_new", nil)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.param, apiErr.Param)
			assert.Empty(t, r.auditedResources(t))
		})
	}
}

// Moving a part's category as part of its update needs parts:update alone, and records the item's event and then the part's, as the two separate calls did.
func TestUpdatePart_MovesTheCategoryUnderThePartPermission(t *testing.T) {
	t.Parallel()
	r := newCategoryChangeRepos(t)
	ctrl := gomock.NewController(t)
	idempotencyMed := mediatormock.NewMockIdempotencyMed(ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: idempotencyMed}).AnyTimes()
	idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{TypeID: "idk_part", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_part", gomock.Any()).Return(nil)

	svc := NewPartSvc(&PartSvcConfig{
		Repos:           r.factory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   NewJobSvcFactory(),
		TxManager:       &stubTxManager{factory: r.factory},
	})

	sku := "PART-RENAMED"
	partBefore := &domain.Part{ID: "pt_part", ItemID: "itm_part", Item: &domain.Item{ID: "itm_part", SKU: "PART-1", ItemCategoryID: "ic_old"}}
	partMoved := &domain.Part{ID: "pt_part", ItemID: "itm_part", Item: &domain.Item{ID: "itm_part", SKU: "PART-1", ItemCategoryID: "ic_new"}}
	partSaved := &domain.Part{ID: "pt_part", ItemID: "itm_part", Item: &domain.Item{ID: "itm_part", SKU: sku, ItemCategoryID: "ic_new"}}

	gomock.InOrder(
		r.parts.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partBefore, nil),
		r.items.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partItem("ic_old", "Old"), nil),
		r.category.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&domain.ItemCategoryFull{ID: "ic_new", ItemCategoryTypeCode: string(constants.ItemCategoryTypeProduct)}, nil),
		r.items.EXPECT().GetCategoryBaseUnitID(gomock.Any(), "ic_new").Return("un_pair", "", nil),
		r.items.EXPECT().ChangeCategory(gomock.Any(), gomock.Any()).Return(nil),
		r.items.EXPECT().UpdateRateUnits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
		r.items.EXPECT().UpdateMaterialOrderPointUnit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
		r.items.EXPECT().UpdateConsumptionProductionQuantityUnits(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
		r.items.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partItem("ic_new", "New"), nil),
		r.parts.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partMoved, nil),
		r.parts.EXPECT().ExistsBySKU(gomock.Any(), "ac_cat", sku, gomock.Any()).Return(false, nil),
		r.parts.EXPECT().UpdateItem(gomock.Any(), gomock.Any()).Return(nil),
		r.parts.EXPECT().TouchUpdatedAt(gomock.Any(), "pt_part").Return(nil),
		r.parts.EXPECT().Get(gomock.Any(), gomock.Any()).Return(partSaved, nil),
	)

	categoryID := "ic_new"
	updated, apiErr := svc.UpdatePart(partsWriterCtx("ac_cat"), domain.UpdatePartParams{PartID: "pt_part", SKU: &sku, CategoryID: &categoryID})

	require.Nil(t, apiErr)
	assert.Equal(t, "ic_new", updated.Item.ItemCategoryID)
	assert.Equal(t, []string{"update item itm_part", "update part pt_part"}, r.auditedResources(t), "the part's own event diffs only what followed the move")
}
