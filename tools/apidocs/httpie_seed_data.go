package main

// httpieVariable holds a variable name, its local dev value, and whether it's a secret.
type httpieVariable struct {
	Name     string
	Value    string
	IsSecret bool
}

// v2Variables are the local development seed data variables.
var v2Variables = []httpieVariable{
	{Name: "host", Value: "localhost:8081"},
	{Name: "version", Value: "v1"},
	{Name: "api-version", Value: "1.0.forge-preview.3"},
	{Name: "api-key", Value: "mrp_sk_prod_u6Xh5ZpaUruMAU12EPAs4z_rSA4zJM5NbRqAtalvXMoRWOUPohFKJtX7ZUFUOp36IVwdiUCZu", IsSecret: true},
	{Name: "act-id", Value: "ac_01k0a5smf9ekb8rqg12555zjqa"},
	{Name: "user-id", Value: "us_1wjfmmbwg8l7"},
	{Name: "admin-role-id", Value: "rl_mtg88e6u6fbu"},
	{Name: "key-id", Value: "apky_pajbskcck3cabxajdh8h8"},
	{Name: "address-id", Value: "ad_01k0a5smf9enr81a4zvyht3zw0"},
	{Name: "unit-id", Value: "un_01seedpair000000000"},
	{Name: "unit-group-id", Value: "ungp_01k0a5ecy9edg9za40dnccw53n"},
	{Name: "property-id", Value: "pp_01k0a7ntn1ez6aw8x850femxeh"},
	{Name: "attribute-id", Value: "at_01seedbeige00000000"},
	{Name: "item-id", Value: "it_01k0a7100aeysrs9vxpeq14yxj"},
	{Name: "item-category-id", Value: "itcg_01seedsocks000000"},
	{Name: "product-id", Value: "pd_01k0a65nx2e2crfxrvryyxnmdh"},
	{Name: "product-line-id", Value: "pdln_01k0a735ype5e8nrhv1n5dhq1q"},
	{Name: "product-type-id", Value: "sale"},
	{Name: "material-id", Value: "ml_01seedyrn1mat000000"},
	{Name: "department-id", Value: "dp_01k0a5r01yfx3sj1vy9qgv3dc0"},
	{Name: "machine-id", Value: "mc_01k0a52fb6eqhtbx9hdxj3vvnh"},
	{Name: "customer-account-id", Value: "ac_01k09wm2fgevdsc344gpbcj30f"},
	{Name: "account-group-id", Value: "acgp_01k0a413mjeth8pe1g70t0thax"},
	{Name: "sales-order-id", Value: "or_01k0a8bs2yejxbsvqhrx4drkq1"},
	{Name: "pick-id", Value: "pk_01k0a5tsn7f7psgagr1732fxqa"},
	{Name: "shipment-id", Value: "sh_01k0a87w33emw8pmkz1mf86cg1"},
	{Name: "invoice-id", Value: "iv_01k09wnac0e1ar211e0sy0ba4g"},
	{Name: "payment-term-id", Value: "pytm_01seednet3000000"},
	{Name: "shipping-term-id", Value: "prepaid_billed"},
	{Name: "carrier-id", Value: "delivery"},
	{Name: "location-id", Value: "sglc_01seedbuilding0000"},
	{Name: "sandbox-id", Value: "ac_sandbox_01k0a5smf9ekb8rqg12555zjqb"},
	{Name: "account-status-id", Value: "normal"},
	{Name: "priority-id", Value: "normal"},
	{Name: "adjustment-type-id", Value: "discount"},
	{Name: "place-id", Value: "ChIJN1gggt_t2Z44AR4PVM_67p73Y"},
}

// idPrefixToVariable maps known ID prefixes to HTTPie variable names.
// Used to replace hardcoded IDs in request body examples with {{variable}} references.
var idPrefixToVariable = map[string]string{
	"ac_":   "act-id",
	"ad_":   "address-id",
	"apky_": "key-id",
	"at_":   "attribute-id",
	"or_":   "sales-order-id",
	"un_":   "unit-id",
	"ungp_": "unit-group-id",
	"pp_":   "property-id",
	"it_":   "item-id",
	"itcg_": "item-category-id",
	"pd_":   "product-id",
	"pdln_": "product-line-id",
	"ml_":   "material-id",
	"dp_":   "department-id",
	"mc_":   "machine-id",
	"acgp_": "account-group-id",
	"pk_":   "pick-id",
	"sh_":   "shipment-id",
	"iv_":   "invoice-id",
	"pytm_": "payment-term-id",
	"sglc_": "location-id",
	"us_":   "user-id",
	"rl_":   "admin-role-id",
}

func buildEnvironments() []HTTPieEnvironment {
	// Default: local dev with seed data values
	defaultVars := make([]HTTPieVariable, len(v2Variables))
	for i, v := range v2Variables {
		defaultVars[i] = HTTPieVariable{
			Name:     v.Name,
			Value:    v.Value,
			IsSecret: v.IsSecret,
		}
	}

	// Prod: production host, empty IDs
	prodVars := make([]HTTPieVariable, len(v2Variables))
	for i, v := range v2Variables {
		val := ""
		switch v.Name {
		case "host":
			val = "https://api.augno.com"
		case "version":
			val = "v1"
		}
		prodVars[i] = HTTPieVariable{
			Name:     v.Name,
			Value:    val,
			IsSecret: v.IsSecret,
		}
	}

	return []HTTPieEnvironment{
		{
			Name:        "Default",
			Color:       "gray",
			IsDefault:   true,
			IsLocalOnly: false,
			Variables:   defaultVars,
		},
		{
			Name:        "Prod",
			Color:       "red",
			IsDefault:   false,
			IsLocalOnly: false,
			Variables:   prodVars,
		},
	}
}
