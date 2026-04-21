-- +goose Up

-- Ensure every customer/supplier account_relation and its counterparty account
-- has default_billing_address_id and default_shipping_address_id populated.
--
-- The Supplier and Customer adapters require these fields at map() time, and
-- a relation without them (falling back through the account with nothing set
-- either) causes the list/find endpoints to throw "Default billing address is
-- required". This migration picks an address already linked to the account via
-- account_address and wires it up as the default. Child relations without
-- their own linked addresses inherit from the parent relation, then from the
-- owner account, so parent-child hierarchies resolve correctly.
--
-- Subqueries that read from `account` or `account_relation` while we UPDATE
-- them are wrapped in a derived table to sidestep MySQL error 1093.

-- Pass 1: backfill account_relation.default_billing_address_id from an address
-- linked to the counterparty account. Pick the earliest-linked address for
-- determinism.
UPDATE account_relation AS ar
SET ar.default_billing_address_id = (
    SELECT aa.address_id
    FROM account_address AS aa
    WHERE aa.account_id = ar.counterparty_account_id
    ORDER BY aa.created_at, aa.id
    LIMIT 1
)
WHERE ar.default_billing_address_id IS NULL
  AND ar.account_relation_role_code IN ('customer', 'supplier')
  AND EXISTS (
      SELECT 1 FROM account_address AS aa
      WHERE aa.account_id = ar.counterparty_account_id
  );

UPDATE account_relation AS ar
SET ar.default_shipping_address_id = (
    SELECT aa.address_id
    FROM account_address AS aa
    WHERE aa.account_id = ar.counterparty_account_id
    ORDER BY aa.created_at, aa.id
    LIMIT 1
)
WHERE ar.default_shipping_address_id IS NULL
  AND ar.account_relation_role_code IN ('customer', 'supplier')
  AND EXISTS (
      SELECT 1 FROM account_address AS aa
      WHERE aa.account_id = ar.counterparty_account_id
  );

-- Pass 2: child relations without their own linked addresses inherit from the
-- parent relation. Read parent values via a derived table to avoid error 1093
-- when updating account_relation while reading from it.
UPDATE account_relation AS ar
INNER JOIN (
    SELECT id, default_billing_address_id, default_shipping_address_id
    FROM account_relation
    WHERE default_billing_address_id IS NOT NULL
       OR default_shipping_address_id IS NOT NULL
) AS parent ON parent.id = ar.parent_account_relation_id
SET
    ar.default_billing_address_id  = COALESCE(ar.default_billing_address_id,  parent.default_billing_address_id),
    ar.default_shipping_address_id = COALESCE(ar.default_shipping_address_id, parent.default_shipping_address_id)
WHERE ar.account_relation_role_code IN ('customer', 'supplier')
  AND (ar.default_billing_address_id IS NULL OR ar.default_shipping_address_id IS NULL);

-- Pass 3: any remaining customer/supplier relations inherit from the owner
-- account's defaults. This covers root-level relations where the counterparty
-- account has no account_address link and no parent relation.
UPDATE account_relation AS ar
INNER JOIN account AS owner ON owner.id = ar.owner_account_id
SET
    ar.default_billing_address_id  = COALESCE(ar.default_billing_address_id,  owner.default_billing_address_id),
    ar.default_shipping_address_id = COALESCE(ar.default_shipping_address_id, owner.default_shipping_address_id)
WHERE ar.account_relation_role_code IN ('customer', 'supplier')
  AND (ar.default_billing_address_id IS NULL OR ar.default_shipping_address_id IS NULL);

-- Pass 4: backfill account.default_billing_address_id for counterparty accounts
-- in customer/supplier relations. The account table has a UNIQUE key on
-- default_billing_address_id, so skip addresses already claimed by another
-- account. Each seeded/created account typically owns its own addresses, so
-- this is safe in practice.
UPDATE account AS a
SET a.default_billing_address_id = (
    SELECT aa.address_id
    FROM account_address AS aa
    WHERE aa.account_id = a.id
      AND aa.address_id NOT IN (
          SELECT default_billing_address_id FROM (
              SELECT default_billing_address_id FROM account
              WHERE default_billing_address_id IS NOT NULL
          ) AS claimed
      )
    ORDER BY aa.created_at, aa.id
    LIMIT 1
)
WHERE a.default_billing_address_id IS NULL
  AND a.id IN (
      SELECT counterparty_account_id FROM (
          SELECT DISTINCT ar.counterparty_account_id
          FROM account_relation AS ar
          WHERE ar.account_relation_role_code IN ('customer', 'supplier')
      ) AS counterparties
  );

UPDATE account AS a
SET a.default_shipping_address_id = (
    SELECT aa.address_id
    FROM account_address AS aa
    WHERE aa.account_id = a.id
      AND aa.address_id NOT IN (
          SELECT default_shipping_address_id FROM (
              SELECT default_shipping_address_id FROM account
              WHERE default_shipping_address_id IS NOT NULL
          ) AS claimed
      )
    ORDER BY aa.created_at, aa.id
    LIMIT 1
)
WHERE a.default_shipping_address_id IS NULL
  AND a.id IN (
      SELECT counterparty_account_id FROM (
          SELECT DISTINCT ar.counterparty_account_id
          FROM account_relation AS ar
          WHERE ar.account_relation_role_code IN ('customer', 'supplier')
      ) AS counterparties
  );

-- +goose Down

-- Data backfill — no meaningful rollback. Leaving as a no-op preserves the
-- populated defaults rather than reintroducing the null state that caused
-- the adapter failures.
SELECT 1;
