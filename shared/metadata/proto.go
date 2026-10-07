package metadata

import (
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// FromCreateRequest is a create body's metadata as stored. A key sent as null has nothing to remove on a
// new record, so it is left out.
func FromCreateRequest(m map[string]*string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

// PatchToProto is an update body's metadata as the RPC carries it: nil when the field was left out,
// clear for `metadata: null`, and per key, null to remove it or a string to write it.
func PatchToProto(c field.Clearable[map[string]*string]) *pb.MetadataPatch {
	if c.IsUnset() {
		return nil
	}
	if c.IsClear() {
		return &pb.MetadataPatch{Clear: true}
	}
	m, _ := c.Value()
	patch := &pb.MetadataPatch{Set: map[string]string{}}
	for k, v := range m {
		if v == nil {
			patch.Remove = append(patch.Remove, k)
		} else {
			patch.Set[k] = *v
		}
	}
	return patch
}

// PatchFromProto is the RPC's patch as an Update; nil changes nothing.
func PatchFromProto(p *pb.MetadataPatch) Update {
	if p == nil {
		return Update{}
	}
	return Update{Clear: p.Clear, Set: p.Set, Remove: p.Remove}
}
