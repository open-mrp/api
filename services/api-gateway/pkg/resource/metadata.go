package apiresource

// MetadataFromProto is a resource's metadata as it is shown: `{}` rather than null when nothing is set,
// since proto delivers an empty map as nil.
func MetadataFromProto(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
