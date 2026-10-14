package recordstub

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/include"
)

var (
	callsMu sync.Mutex
	calls   = map[constants.RecordType][][]string{}
)

func fakeLoader(t constants.RecordType) Loader {
	return func(_ context.Context, ids []string) (map[string]*apiresource.Record, *apierror.APIError) {
		callsMu.Lock()
		calls[t] = append(calls[t], slices.Clone(ids))
		callsMu.Unlock()
		out := map[string]*apiresource.Record{}
		for _, id := range ids {
			if id == "gone" {
				continue
			}
			out[id] = &apiresource.Record{Number: new("N-" + id)}
		}
		return out, nil
	}
}

func init() {
	Register(constants.RecordTypeJob, fakeLoader(constants.RecordTypeJob))
	Register(constants.RecordTypeWorkOrder, fakeLoader(constants.RecordTypeWorkOrder))
}

func TestKeyRoundTrip(t *testing.T) {
	t.Parallel()
	typ, id, ok := ParseKey(Key(constants.RecordTypeBatch, "bt_1"))
	assert.True(t, ok)
	assert.Equal(t, constants.RecordTypeBatch, typ)
	assert.Equal(t, "bt_1", id)
	_, _, ok = ParseKey("no-type")
	assert.False(t, ok)
}

func TestRegister_RejectsUnknownAndDuplicate(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { Register("not_a_record", fakeLoader("not_a_record")) })
	assert.Panics(t, func() { Register(constants.RecordTypeJob, fakeLoader(constants.RecordTypeJob)) })
	assert.True(t, Registered(constants.RecordTypeJob))
	assert.False(t, Registered(constants.RecordTypeBill))
}

// Keys of several types load with one call per type, and every stub comes back as a record of its own type.
func TestLoad_GroupsByType(t *testing.T) {
	t.Parallel()
	keys := []string{
		Key(constants.RecordTypeJob, "a"), Key(constants.RecordTypeWorkOrder, "b"),
		Key(constants.RecordTypeJob, "c"), Key(constants.RecordTypeJob, "a"), Key(constants.RecordTypeJob, "gone"),
	}
	loaded, apiErr := Load(context.Background(), keys)
	require.Nil(t, apiErr)

	job := One(loaded, constants.RecordTypeJob, "a")
	require.NotNil(t, job)
	assert.Equal(t, apiresource.Record{ID: "a", Object: constants.ObjectTypeRecord, Type: constants.RecordTypeJob, Number: new("N-a")}, *job)
	assert.Nil(t, One(loaded, constants.RecordTypeJob, "gone"), "a record that no longer exists has no stub")
	assert.Len(t, List(loaded, constants.RecordTypeJob, []string{"a", "gone", "c"}), 2)

	callsMu.Lock()
	defer callsMu.Unlock()
	assert.Contains(t, calls[constants.RecordTypeJob], []string{"a", "c", "gone"})
}

func TestLoad_UnregisteredTypeIsAnInvariantViolation(t *testing.T) {
	t.Parallel()
	_, apiErr := Load(context.Background(), []string{Key(constants.RecordTypeBill, "x")})
	require.NotNil(t, apiErr)
}

type parent struct {
	jobID string
	refs  []Ref
	job   *apiresource.Record
	all   []apiresource.Record
}

func TestSubFields(t *testing.T) {
	t.Parallel()
	one := SubOne("related.job", constants.RecordTypeJob,
		func(_ context.Context, p any) string { return p.(*parent).jobID },
		func(p any, s *apiresource.Record) { p.(*parent).job = s })
	refs := SubRefs("record", include.CardinalityList,
		func(_ context.Context, p any) []Ref { return p.(*parent).refs },
		func(p any, s []apiresource.Record) { p.(*parent).all = s })

	p := &parent{jobID: "j1", refs: []Ref{{constants.RecordTypeWorkOrder, "n1"}, {constants.RecordTypeJob, "gone"}}}
	ctx := context.Background()
	assert.Equal(t, []string{Key(constants.RecordTypeJob, "j1")}, one.ExtractIDs(ctx, p))
	assert.Equal(t, []string{Key(constants.RecordTypeWorkOrder, "n1"), Key(constants.RecordTypeJob, "gone")}, refs.ExtractIDs(ctx, p))

	loaded, apiErr := Load(ctx, append(one.ExtractIDs(ctx, p), refs.ExtractIDs(ctx, p)...))
	require.Nil(t, apiErr)
	one.Populate(ctx, p, loaded)
	refs.Populate(ctx, p, loaded)
	require.NotNil(t, p.job)
	assert.Equal(t, "j1", p.job.ID)
	require.Len(t, p.all, 1)
	assert.Equal(t, "n1", p.all[0].ID)

	empty := &parent{}
	assert.Nil(t, one.ExtractIDs(ctx, empty))
}
