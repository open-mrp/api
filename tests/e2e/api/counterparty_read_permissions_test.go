//go:build e2e

package api_test

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A read asks for the endpoint's own permission in the seller's own account. customers:read and suppliers:read open it
// only while the request acts in a customer's or a supplier's account, and the own permission does not open it there.

// counterpartyTarget addresses an endpoint inside a customer's or supplier's account and the status a permitted read
// gets there: a list answers 200, and a record the account does not hold reads as not found, past the permission check.
type counterpartyTarget struct {
	path   string
	status int
}

type counterpartyRead struct {
	name string
	own  string
	// path lists or reads a record in the seller's own account.
	path string
	// inCustomer and inSupplier leave path empty where nothing can be read in that kind of account.
	inCustomer, inSupplier counterpartyTarget
}

func counterpartyList(name, own, path string) counterpartyRead {
	return counterpartyRead{name: name, own: own, path: path,
		inCustomer: counterpartyTarget{path, http.StatusOK}, inSupplier: counterpartyTarget{path, http.StatusOK}}
}

// counterpartySellerRecord reads one of the seller's own records, which no customer or supplier account holds.
func counterpartySellerRecord(name, own, path string) counterpartyRead {
	return counterpartyRead{name: name, own: own, path: path,
		inCustomer: counterpartyTarget{path, http.StatusNotFound}, inSupplier: counterpartyTarget{path, http.StatusNotFound}}
}

var counterpartyReads = []counterpartyRead{
	counterpartyList("list catalog product lines", "products:read", "/v1/catalog/catalog/product-lines"),
	counterpartyList("list catalog products", "products:read", "/v1/catalog/catalog/product-lines/"+SeedProductLineID+"/products"),
	counterpartyList("list materials", "materials:read", materialsPath),
	{name: "retrieve material", own: "materials:read", path: materialsPath + "/" + SeedMaterialID},
	counterpartyList("list parts", "parts:read", "/v1/catalog/parts"),
	{name: "retrieve part", own: "parts:read", path: "/v1/catalog/parts/" + SeedPartID},
	counterpartyList("list product lines", "product_lines:read", productLinesPath),
	counterpartySellerRecord("retrieve product line", "product_lines:read", productLinesPath+"/"+SeedProductLineID),
	counterpartyList("list products", "items:read", productsPath),
	counterpartySellerRecord("retrieve product", "items:read", productsPath+"/"+SeedProductID),
	counterpartySellerRecord("retrieve job", "jobs:read", "/v1/core/jobs/"+SeedJobID),
	counterpartyList("list account users", "team:read", accountUsersPath),
	{name: "retrieve account user", own: "team:read", path: accountUsersPath + "/" + SeedAccountUserID,
		inCustomer: counterpartyTarget{accountUsersPath + "/" + SeedCustomerAccountUserID, http.StatusOK},
		inSupplier: counterpartyTarget{accountUsersPath + "/" + SeedSupplierAccountUserID, http.StatusOK}},
	counterpartyList("list carriers", "carriers:read", "/v1/operations/carriers"),
	counterpartySellerRecord("retrieve carrier", "carriers:read", "/v1/operations/carriers/"+SeedCarrierID),
	counterpartyList("list service levels", "carriers:read", "/v1/operations/carriers/"+SeedCarrierID+"/service-levels"),
	counterpartySellerRecord("retrieve service level", "carriers:read", "/v1/operations/carriers/"+SeedCarrierID+"/service-levels/"+SeedServiceLevelID),
	counterpartyList("list picks", "picks:read", "/v1/operations/picks"),
	counterpartySellerRecord("list shipment lines", "shipments:read", "/v1/operations/shipments/"+SeedShipmentID+"/lines"),
	counterpartySellerRecord("retrieve shipment line", "shipments:read", "/v1/operations/shipments/"+SeedShipmentID+"/lines/"+SeedShipmentLineID),
	counterpartyList("list account prices", "discounts:read", "/v1/sales/account-prices"),
	counterpartySellerRecord("retrieve account price", "discounts:read", "/v1/sales/account-prices/"+SeedAccountPriceID),
	counterpartyList("list addresses", "addresses:read", addressesPath),
	{name: "retrieve address", own: "addresses:read", path: addressesPath + "/" + SeedAddressID,
		inCustomer: counterpartyTarget{addressesPath + "/" + SeedAddressID, http.StatusOK},
		inSupplier: counterpartyTarget{addressesPath + "/" + SeedSupplierAddressID, http.StatusOK}},
	counterpartySellerRecord("retrieve customer", "customers:read", customersPath+"/"+SeedCustomerAccountID),
	counterpartySellerRecord("list frequently ordered products", "items:read", customersPath+"/"+SeedCustomerAccountID+"/frequently-ordered-products"),
	counterpartySellerRecord("list notification recipients", "customers:read", customersPath+"/"+SeedCustomerAccountID+"/notification-recipients"),
	counterpartyList("list order discounts", "discounts:read", "/v1/sales/order-discounts"),
	counterpartyList("list sales orders", "sales_orders:read", salesOrdersPath),
	counterpartySellerRecord("retrieve sales order", "sales_orders:read", salesOrdersPath+"/"+SeedSalesOrderID),
	counterpartyList("list volume discounts", "discounts:read", "/v1/sales/volume-discounts"),
	counterpartySellerRecord("retrieve volume discount", "discounts:read", "/v1/sales/volume-discounts/"+SeedVolumeDiscountID),
}

// counterpartyRoles hands out one role-scoped client per permission set, shared by the subtests that need it.
type counterpartyRoles struct {
	mu      sync.Mutex
	clients map[string]*Client
}

func (r *counterpartyRoles) holding(t *testing.T, perms ...string) *Client {
	t.Helper()
	key := strings.Join(perms, ",")
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.clients == nil {
		r.clients = map[string]*Client{}
	}
	if c, ok := r.clients[key]; ok {
		return c
	}
	c := customRoleClient(t, perms...)
	r.clients[key] = c
	return c
}

// requirePermissionRefused checks the refusal is for want of permission and names the one permission that would have let the caller through, and none of the others.
func requirePermissionRefused(t *testing.T, status int, body []byte, want string, notOffered ...string) {
	t.Helper()
	requireStatus(t, http.StatusForbidden, status, body)
	message, _ := requireErrorResponse(t, body, "insufficient_permissions", "invalid_request_error")["message"].(string)
	assert.Contains(t, message, want, "the refusal names the permission that is needed")
	for _, other := range notOffered {
		if other != want {
			assert.NotContains(t, message, other, "the refusal offers only the permission this account needs")
		}
	}
}

func counterpartyGet(t *testing.T, c *Client, path string) (int, []byte) {
	t.Helper()
	status, body, err := c.GetListRaw(path, nil)
	require.NoError(t, err)
	return status, body
}

func TestCounterpartyReadPermissions(t *testing.T) {
	t.Parallel()
	roles := &counterpartyRoles{}
	alternates := []string{"customers:read", "suppliers:read"}
	for _, r := range counterpartyReads {
		roles.holding(t, r.own)
	}
	for _, alt := range alternates {
		roles.holding(t, alt)
	}

	for _, r := range counterpartyReads {
		t.Run(r.name, func(t *testing.T) {
			t.Parallel()
			own := roles.holding(t, r.own)

			status, body := counterpartyGet(t, own, r.path)
			requireStatus(t, http.StatusOK, status, body)
			status, body = counterpartyGet(t, apiClient, r.path)
			requireStatus(t, http.StatusOK, status, body)
			for _, alt := range alternates {
				if alt == r.own {
					continue
				}
				status, body = counterpartyGet(t, roles.holding(t, alt), r.path)
				requirePermissionRefused(t, status, body, r.own, alternates...)
			}

			for _, kind := range []struct {
				account, perm string
				target        counterpartyTarget
			}{
				{SeedCustomerAccountID, "customers:read", r.inCustomer},
				{SeedSupplierAccountID, "suppliers:read", r.inSupplier},
			} {
				refusedPath := kind.target.path
				if refusedPath == "" {
					refusedPath = r.path
				}
				if r.own != kind.perm {
					status, body = counterpartyGet(t, own.WithAccountID(kind.account), refusedPath)
					requirePermissionRefused(t, status, body, kind.perm, r.own)
				}
				if kind.target.path == "" {
					continue
				}
				status, body = counterpartyGet(t, roles.holding(t, kind.perm).WithAccountID(kind.account), kind.target.path)
				requireStatus(t, kind.target.status, status, body)
				status, body = counterpartyGet(t, apiClient.WithAccountID(kind.account), kind.target.path)
				requireStatus(t, kind.target.status, status, body)
			}
		})
	}
}
