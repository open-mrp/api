package constants

// ProductTypeCode represents the type code of a product.
type ProductTypeCode string

const (
	// ProductTypeCodeSale indicates a sellable product.
	ProductTypeCodeSale ProductTypeCode = "sale"
	// ProductTypeCodeService indicates a service product.
	ProductTypeCodeService ProductTypeCode = "service"
	// ProductTypeCodeShipping indicates a shipping charge product.
	ProductTypeCodeShipping ProductTypeCode = "shipping"
	// ProductTypeCodeCredit indicates a credit product.
	ProductTypeCodeCredit ProductTypeCode = "credit"
	// ProductTypeCodeReturn indicates a return product.
	ProductTypeCodeReturn ProductTypeCode = "return"
	// ProductTypeCodeTax indicates a tax product.
	ProductTypeCodeTax ProductTypeCode = "tax"
)

func (m ProductTypeCode) IsValid() bool {
	switch m {
	case ProductTypeCodeSale, ProductTypeCodeService, ProductTypeCodeShipping, ProductTypeCodeCredit, ProductTypeCodeReturn, ProductTypeCodeTax:
		return true
	default:
		return false
	}
}

func (m *ProductTypeCode) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

func (m ProductTypeCode) EnumValues() []string {
	return []string{
		string(ProductTypeCodeSale),
		string(ProductTypeCodeService),
		string(ProductTypeCodeShipping),
		string(ProductTypeCodeCredit),
		string(ProductTypeCodeReturn),
		string(ProductTypeCodeTax),
	}
}
