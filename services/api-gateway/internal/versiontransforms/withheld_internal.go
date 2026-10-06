package versiontransforms

import "github.com/open-mrp/api/shared/constants"

// everyObjectType is every object type a response can be served as. A resource that rides inside a great many others, through includes and embedded references, is downgraded on all of them: the walk touches nothing but objects of that resource.
func everyObjectType() []constants.ObjectType {
	values := constants.ObjectType("").EnumValues()
	out := make([]constants.ObjectType, len(values))
	for i, v := range values {
		out[i] = constants.ObjectType(v)
	}
	return out
}

// zeroWithheldIn gives, in place, every object in node whose `object` is objectType the zero value preview.5 typed for each key in zeros that preview.6 withheld as null.
//
// preview.6 withholds the seller's internal data from customer and supplier portal users on fields preview.5 typed as a plain string or boolean. A preview.5 caller cannot be handed a null its contract never allowed, and must not be handed the value, so it reads the type's zero value.
func zeroWithheldIn(node any, objectType constants.ObjectType, zeros map[string]any) {
	switch v := node.(type) {
	case map[string]any:
		if asString(v["object"]) == string(objectType) {
			for key, zero := range zeros {
				if value, present := v[key]; present && value == nil {
					v[key] = zero
				}
			}
		}
		for _, child := range v {
			zeroWithheldIn(child, objectType, zeros)
		}
	case []any:
		for _, child := range v {
			zeroWithheldIn(child, objectType, zeros)
		}
	}
}
