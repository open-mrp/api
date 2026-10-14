package metadatapb

import (
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/metadata"
)

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
func PatchFromProto(p *pb.MetadataPatch) metadata.Update {
	if p == nil {
		return metadata.Update{}
	}
	return metadata.Update{Clear: p.Clear, Set: p.Set, Remove: p.Remove}
}
