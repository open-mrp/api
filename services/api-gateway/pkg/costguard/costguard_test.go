package costguard

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
)

type rate struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type item struct {
	ID        string `json:"id"`
	UnitValue *rate  `json:"unit_value"`
	UnitCost  *rate  `json:"unit_cost" sensitive:"cost"`
}

type product struct {
	ID   string `json:"id"`
	Item *item  `json:"item"`
}

type line struct {
	ID       string   `json:"id"`
	Product  *product `json:"product"`
	UnitCost *rate    `json:"unit_cost" sensitive:"cost"`
}

type list[T any] struct {
	Data []T `json:"data"`
}

type order struct {
	ID    string      `json:"id"`
	Lines *list[line] `json:"lines"`
}

type account struct {
	ID       string          `json:"id"`
	Children []*account      `json:"children"`
	Margin   *string         `json:"margin" sensitive:"cost"`
	ByCode   map[string]item `json:"by_code"`
	Extra    any             `json:"extra"`
}

type plain struct {
	ID    string    `json:"id"`
	Inner *struct{} `json:"inner"`
	Rates []rate    `json:"rates"`
}

func ptr[T any](v T) *T { return &v }

func sampleOrder() *order {
	shared := &item{ID: "itm_1", UnitValue: &rate{ID: "rt_v", Value: "5"}, UnitCost: &rate{ID: "rt_c", Value: "2"}}
	return &order{
		ID: "so_1",
		Lines: &list[line]{Data: []line{
			{ID: "sol_1", Product: &product{ID: "pd_1", Item: shared}, UnitCost: &rate{ID: "rt_l1", Value: "3"}},
			{ID: "sol_2", Product: &product{ID: "pd_2", Item: shared}, UnitCost: &rate{ID: "rt_l2", Value: "4"}},
			{ID: "sol_3"},
		}},
	}
}

func ctxWith(identity *types.Identity) context.Context {
	return appctx.WithIdentity(context.Background(), identity)
}

func internalIdentity(roleType constants.RoleType, perms ...string) *types.Identity {
	accountID := "ac_seller"
	roleTypeCode := string(roleType)
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "usr_staff",
			AccountID:    &accountID,
			RoleType:     &roleTypeCode,
			Permissions:  granted,
		},
	}
}

func relationIdentity(relation types.IdentityRelationType, perms ...string) *types.Identity {
	actorAccountID := "ac_buyer"
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	return &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_seller"},
		Actor: &types.IdentityActor{
			RelationType: relation,
			ID:           "usr_portal",
			AccountID:    &actorAccountID,
			Permissions:  granted,
		},
	}
}

func TestRedact_ByCaller(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		identity *types.Identity
		visible  bool
	}{
		{"internal without costs:read", internalIdentity(constants.RoleTypeCustom, "items:read", "sales_orders:read"), false},
		{"internal with costs:read", internalIdentity(constants.RoleTypeCustom, "costs:read"), true},
		{"internal with costs:update only", internalIdentity(constants.RoleTypeCustom, "costs:update"), false},
		{"admin", internalIdentity(constants.RoleTypeAdmin), true},
		{"customer portal actor", relationIdentity(types.IdentityRelationTypeCustomer), false},
		{"customer portal actor whose own role grants costs:read", relationIdentity(types.IdentityRelationTypeCustomer, "costs:read"), false},
		{"supplier portal actor whose own role grants costs:read", relationIdentity(types.IdentityRelationTypeSupplier, "costs:read"), false},
		{"unauthenticated", types.GetUnauthenticatedIdentity(ptr("ac_seller")), false},
		{"no identity", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.identity != nil {
				ctx = ctxWith(tt.identity)
			}
			got := Redact(ctx, sampleOrder()).(*order)

			lines := got.Lines.Data
			if tt.visible {
				assert.Equal(t, "3", lines[0].UnitCost.Value)
				assert.Equal(t, "2", lines[0].Product.Item.UnitCost.Value)
				return
			}
			for _, l := range lines {
				assert.Nil(t, l.UnitCost, l.ID)
				if l.Product != nil {
					assert.Nil(t, l.Product.Item.UnitCost, l.ID)
					assert.Equal(t, "5", l.Product.Item.UnitValue.Value, "a field that is not cost data is left alone")
				}
			}
			assert.Equal(t, "sol_1", lines[0].ID)
		})
	}
}

func TestStrip_SerializesNull(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(Strip(sampleOrder()))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"so_1","lines":{"data":[
		{"id":"sol_1","product":{"id":"pd_1","item":{"id":"itm_1","unit_value":{"id":"rt_v","value":"5"},"unit_cost":null}},"unit_cost":null},
		{"id":"sol_2","product":{"id":"pd_2","item":{"id":"itm_1","unit_value":{"id":"rt_v","value":"5"},"unit_cost":null}},"unit_cost":null},
		{"id":"sol_3","product":null,"unit_cost":null}]}}`, string(body))
}

func TestStrip_RecursiveTypesMapsAndInterfaces(t *testing.T) {
	t.Parallel()

	root := &account{
		ID:     "ac_1",
		Margin: ptr("0.4"),
		Children: []*account{
			{ID: "ac_2", Margin: ptr("0.3"), Children: []*account{{ID: "ac_3", Margin: ptr("0.2")}}},
			nil,
		},
		ByCode: map[string]item{"a": {ID: "itm_a", UnitCost: &rate{Value: "1"}}},
		Extra:  &item{ID: "itm_x", UnitCost: &rate{Value: "9"}},
	}
	Strip(root)

	assert.Nil(t, root.Margin)
	assert.Nil(t, root.Children[0].Margin)
	assert.Nil(t, root.Children[0].Children[0].Margin)
	assert.Nil(t, root.ByCode["a"].UnitCost)
	assert.Equal(t, "itm_a", root.ByCode["a"].ID)
	assert.Nil(t, root.Extra.(*item).UnitCost)
}

func TestStrip_ValueRootAndInterfaceHoldingAValue(t *testing.T) {
	t.Parallel()

	got := Strip(item{ID: "itm_v", UnitCost: &rate{Value: "1"}}).(item)
	assert.Nil(t, got.UnitCost)

	holder := &account{Extra: item{ID: "itm_i", UnitCost: &rate{Value: "1"}}}
	Strip(holder)
	assert.Nil(t, holder.Extra.(item).UnitCost)

	inMap := map[string]any{"x": item{UnitCost: &rate{Value: "1"}}, "y": &item{UnitCost: &rate{Value: "2"}}}
	Strip(inMap)
	assert.Nil(t, inMap["x"].(item).UnitCost)
	assert.Nil(t, inMap["y"].(*item).UnitCost)
}

func TestStrip_NilSafety(t *testing.T) {
	t.Parallel()

	assert.Nil(t, Strip(nil))
	assert.NotPanics(t, func() {
		Strip((*order)(nil))
		Strip(&order{})
		Strip(&order{Lines: &list[line]{}})
		Strip([]*item{nil, {}})
		Strip(&account{Extra: (*item)(nil)})
		Redact(context.Background(), (*item)(nil))
	})
}

func TestHasCostFields(t *testing.T) {
	t.Parallel()

	assert.True(t, HasCostFields(reflect.TypeFor[*order]()))
	assert.True(t, HasCostFields(reflect.TypeFor[[]line]()))
	assert.True(t, HasCostFields(reflect.TypeFor[*account]()), "an any field may hold cost data")
	assert.False(t, HasCostFields(reflect.TypeFor[*plain]()))
	assert.False(t, HasCostFields(reflect.TypeFor[string]()))
}

func BenchmarkRedact_List(b *testing.B) {
	ctx := ctxWith(internalIdentity(constants.RoleTypeCustom, "items:read"))
	items := make([]item, 100)
	for i := range items {
		items[i] = item{ID: "itm", UnitValue: &rate{Value: "1"}, UnitCost: &rate{Value: "1"}}
	}
	resp := &list[item]{Data: items}
	b.ResetTimer()
	for range b.N {
		Redact(ctx, resp)
	}
}

type change struct {
	Field    string          `json:"field"`
	NewValue json.RawMessage `json:"new_value"`
}

func (c *change) RedactCosts() {
	if IsCostName(c.Field) {
		c.NewValue = nil
	}
}

type event struct {
	ID      string            `json:"id"`
	Changes *list[change]     `json:"changes"`
	Pinned  map[string]change `json:"pinned"`
}

func TestStrip_CallsRedactorsWhereverTheyAppear(t *testing.T) {
	t.Parallel()

	ev := &event{
		Changes: &list[change]{Data: []change{
			{Field: "unit_cost_value", NewValue: json.RawMessage(`"4.10"`)},
			{Field: "labor_rate", NewValue: json.RawMessage(`{"value":"22"}`)},
			{Field: "name", NewValue: json.RawMessage(`"Sock"`)},
		}},
		Pinned: map[string]change{"x": {Field: "gross_margin", NewValue: json.RawMessage(`"0.3"`)}},
	}
	Strip(ev)

	assert.Nil(t, ev.Changes.Data[0].NewValue)
	assert.Nil(t, ev.Changes.Data[1].NewValue)
	assert.JSONEq(t, `"Sock"`, string(ev.Changes.Data[2].NewValue))
	assert.Nil(t, ev.Pinned["x"].NewValue)
}

func TestIsCostName(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"unit_cost", "unit_cost_value", "total_cost", "costs_per_unit", "gross_margin", "annual_cogs", "unit_profit", "labor_rate", "overhead_rate", "changeover_labor_rate", "inventory_value"} {
		assert.True(t, IsCostName(name), name)
	}
	for _, name := range []string{"unit_value", "unit_price", "costume", "burn_rate", "labor_time", "freight_amount", "name"} {
		assert.False(t, IsCostName(name), name)
	}
}

type batchSummary struct {
	SKU      string `json:"sku"`
	Quantity string `json:"quantity"`
}

type run struct {
	ID          string              `json:"id"`
	Number      string              `json:"number"`
	BatchCount  *int32              `json:"batch_count" sensitive:"internal"`
	Summaries   *list[batchSummary] `json:"summaries" sensitive:"internal"`
	Responsible *item               `json:"responsible" sensitive:"internal"`
	Lines       *list[line]         `json:"lines"`
}

type releasedWeek struct {
	Run *run `json:"run"`
}

func sampleRun() *run {
	return &run{
		ID:          "prru_1",
		Number:      "7",
		BatchCount:  ptr(int32(3)),
		Summaries:   &list[batchSummary]{Data: []batchSummary{{SKU: "other-buyers-sku", Quantity: "300"}}},
		Responsible: &item{ID: "itm_r", UnitCost: &rate{Value: "8"}},
		Lines:       &list[line]{Data: []line{{ID: "sol_1", UnitCost: &rate{Value: "3"}}}},
	}
}

func agentIdentity(perms ...string) *types.Identity {
	identity := internalIdentity(constants.RoleTypeAgent, perms...)
	identity.Type = types.IdentityActorTypeAgent
	return identity
}

func TestRedact_InternalFieldsReachOnlyTheSellersOwnActors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		identity     *types.Identity
		internal     bool
		costsVisible bool
	}{
		{"internal without costs:read", internalIdentity(constants.RoleTypeCustom, "production_runs:read"), true, false},
		{"internal with costs:read", internalIdentity(constants.RoleTypeCustom, "costs:read"), true, true},
		{"admin", internalIdentity(constants.RoleTypeAdmin), true, true},
		{"agent whose role grants costs:read", agentIdentity("costs:read", "production_runs:read"), true, false},
		{"customer portal actor", relationIdentity(types.IdentityRelationTypeCustomer), false, false},
		{"supplier portal actor whose own role grants costs:read", relationIdentity(types.IdentityRelationTypeSupplier, "costs:read"), false, false},
		{"unauthenticated", types.GetUnauthenticatedIdentity(ptr("ac_seller")), false, false},
		{"no identity", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if tt.identity != nil {
				ctx = ctxWith(tt.identity)
			}
			got := Redact(ctx, &releasedWeek{Run: sampleRun()}).(*releasedWeek).Run

			assert.Equal(t, "prru_1", got.ID)
			assert.Equal(t, "7", got.Number, "the run's number is never withheld")
			if tt.internal {
				require.NotNil(t, got.BatchCount)
				assert.Equal(t, int32(3), *got.BatchCount)
				require.NotNil(t, got.Summaries)
				assert.Equal(t, "other-buyers-sku", got.Summaries.Data[0].SKU)
				require.NotNil(t, got.Responsible)
				assert.Equal(t, tt.costsVisible, got.Responsible.UnitCost != nil, "cost below an internal field still follows costs:read")
			} else {
				assert.Nil(t, got.BatchCount)
				assert.Nil(t, got.Summaries)
				assert.Nil(t, got.Responsible)
			}
			assert.Equal(t, tt.costsVisible, got.Lines.Data[0].UnitCost != nil)
		})
	}
}

func TestRedact_InternalFieldsSerializeNull(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(Redact(ctxWith(relationIdentity(types.IdentityRelationTypeCustomer)), sampleRun()))
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"prru_1","number":"7","batch_count":null,"summaries":null,"responsible":null,
		"lines":{"data":[{"id":"sol_1","product":null,"unit_cost":null}]}}`, string(body))
}

func TestStrip_LeavesInternalFields(t *testing.T) {
	t.Parallel()

	got := Strip(sampleRun()).(*run)
	assert.NotNil(t, got.BatchCount)
	assert.NotNil(t, got.Summaries)
	require.NotNil(t, got.Responsible)
	assert.Nil(t, got.Responsible.UnitCost)
}

func TestHasInternalFields(t *testing.T) {
	t.Parallel()

	assert.True(t, HasInternalFields(reflect.TypeFor[*releasedWeek]()))
	assert.True(t, HasInternalFields(reflect.TypeFor[[]run]()))
	assert.True(t, HasInternalFields(reflect.TypeFor[*account]()), "an any field may hold internal data")
	assert.False(t, HasInternalFields(reflect.TypeFor[*order]()), "cost fields are not internal ones")
	assert.True(t, HasCostFields(reflect.TypeFor[*run]()), "a run still leads to cost fields")
}
