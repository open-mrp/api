package versiontransforms

import (
	"fmt"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/open-mrp/apikit/object"
	"github.com/open-mrp/apikit/version"
)

// forge typed an order discount's amount: `discount_type` + `percentage` (a fraction, 0.1 = 10%) + `amount` became
// `discount` { type: percent_off | amount_off, percent_off (a whole percent) | amount_off (Money) }, and `order_count`
// became `times_redeemed`. preview.9 callers read the old fields and their writes are upgraded. `status` is additive.
type orderDiscountForgeToPreview9 struct {
	forgeToPreview9
	everyObject
}

func init() {
	version.RegisterTransformer(orderDiscountForgeToPreview9{})
}

const (
	orderDiscountsRoute = "/v1/sales/order-discounts"
	orderDiscountRoute  = "/v1/sales/order-discounts/{id}"
)

var hundred = decimal.NewFromInt(100)

func (orderDiscountForgeToPreview9) Transform(_ object.Type, data map[string]any) map[string]any {
	walkObjects(data, func(obj map[string]any) {
		if obj["object"] != "order_discount" {
			return
		}
		if _, ok := obj["discount"]; !ok {
			return
		}
		discount, _ := obj["discount"].(map[string]any)
		delete(obj, "discount")
		obj["percentage"], obj["amount"], obj["discount_type"] = "0", "0", "percentage"
		switch discount["type"] {
		case "percent_off":
			if pct, err := decimal.NewFromString(fmt.Sprint(discount["percent_off"])); err == nil {
				obj["percentage"] = pct.Div(hundred).String()
			}
		case "amount_off":
			obj["discount_type"] = "amount"
			if money, ok := discount["amount_off"].(map[string]any); ok {
				obj["amount"] = fmt.Sprint(money["amount"])
			}
		}
		renameKey(obj, "times_redeemed", "order_count")
	})
	return data
}

func (orderDiscountForgeToPreview9) TransformRequest(_ object.Type, data map[string]any) map[string]any {
	return data
}

func (orderDiscountForgeToPreview9) TransformRequestBody(_ object.Type, method, route string, data map[string]any) map[string]any {
	if (method == http.MethodPost && route == orderDiscountsRoute) || (method == http.MethodPatch && route == orderDiscountRoute) {
		upgradeOrderDiscountWrite(data)
	}
	return data
}

// upgradeOrderDiscountWrite turns preview.9's discount_type, percentage and amount into a discount. Without a type, a
// sent percentage means a percent discount and a sent amount a fixed one; an omitted value is 0, as preview.9 stored it.
func upgradeOrderDiscountWrite(body map[string]any) {
	discountType, hasType := body["discount_type"]
	percentage, hasPct := body["percentage"]
	amount, hasAmount := body["amount"]
	delete(body, "discount_type")
	delete(body, "percentage")
	delete(body, "amount")
	if !hasType && !hasPct && !hasAmount {
		return
	}
	isAmount := discountType == "amount" || (!hasType && hasAmount && !hasPct)
	if isAmount {
		value := "0"
		if hasAmount && amount != nil {
			value = fmt.Sprint(amount)
		}
		body["discount"] = map[string]any{"type": "amount_off", "amount_off": map[string]any{"amount": value, "currency": "usd"}}
		return
	}
	pct := decimal.Zero
	if hasPct && percentage != nil {
		if d, err := decimal.NewFromString(fmt.Sprint(percentage)); err == nil {
			pct = d.Mul(hundred)
		}
	}
	body["discount"] = map[string]any{"type": "percent_off", "percent_off": pct.String()}
}
