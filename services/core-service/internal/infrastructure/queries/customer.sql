-- name: ListCustomersForward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
    ar.external_number,
    ar.is_edi_enabled,
    ar.notes,
    ar.account_status_code AS status,
    ar.commission_status_code,
    ar.freight_status_code,
    ar.carrier_billing_type,
    ar.carrier_billing_account,
    ar.stripe_customer_id,
    ar.stripe_email,
    ab.support_email AS email,
    ab.phone_number,
    ab.website_url,
    EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id) AS is_parent_account,
    ar.parent_account_relation_id,
    pa.name AS parent_account_name,
    par.counterparty_account_id AS parent_account_id,
    par.external_number AS parent_account_number,
    c.id AS default_carrier_id,
    c.name AS default_carrier_name,
    c.is_portal_enabled AS default_carrier_is_portal_enabled,
    c.created_at AS default_carrier_created_at,
    c.updated_at AS default_carrier_updated_at,
    co.id AS default_carrier_option_id,
    co.name AS default_carrier_option_name,
    co.service_level_token AS default_carrier_option_service_level_token,
    co.is_portal_enabled AS default_carrier_option_is_portal_enabled,
    co.created_at AS default_carrier_option_created_at,
    co.updated_at AS default_carrier_option_updated_at,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    pt.created_at AS payment_term_created_at,
    pt.updated_at AS payment_term_updated_at,
    st.id AS shipping_term_id,
    st.name AS shipping_term_name,
    st.is_freight_exempt AS shipping_term_is_freight_exempt,
    st.is_carrier_rate AS shipping_term_is_carrier_rate,
    st.created_at AS shipping_term_created_at,
    st.updated_at AS shipping_term_updated_at,
    p.id AS priority_id,
    p.code AS priority_code,
    p.name AS priority_name,
    sr.id AS default_sales_rep_id,
    sru.name AS default_sales_rep_name,
    sr.status_code AS default_sales_rep_status_code,
    sr.created_at AS default_sales_rep_created_at,
    sr.updated_at AS default_sales_rep_updated_at,
    tg.id AS type_group_id,
    tg.name AS type_group_name,
    tg.commission_status_code AS type_group_commission_status_code,
    tg.freight_status_code AS type_group_freight_status_code,
    tg.account_group_type_code AS type_group_type_code,
    tg.created_at AS type_group_created_at,
    tg.updated_at AS type_group_updated_at,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    par.created_at AS parent_account_created_at,
    par.updated_at AS parent_account_updated_at,
    clq.id AS credit_limit_id,
    clq.value AS credit_limit_value,
    clu.id AS credit_limit_unit_id,
    clu.abbreviation AS credit_limit_unit_abbreviation,
    clu.name AS credit_limit_unit_name,
    clu.unit_dimension_code AS credit_limit_unit_type,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
LEFT JOIN account pa ON pa.id = par.counterparty_account_id
LEFT JOIN carrier c ON c.id = ar.default_carrier_id
LEFT JOIN carrier_option co ON co.id = ar.default_carrier_option_id
LEFT JOIN payment_term pt ON pt.id = ar.payment_term_id
LEFT JOIN shipping_term st ON st.id = ar.shipping_term_id
LEFT JOIN priority p ON p.code = ar.priority_code
LEFT JOIN account_user sr ON sr.id = ar.default_sales_rep_id
LEFT JOIN `user` sru ON sru.id = sr.user_id
LEFT JOIN account_group tg ON tg.id = ar.account_group_id AND tg.account_group_type_code = 'type_group'
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
LEFT JOIN quantity clq ON clq.id = ar.credit_limit_id
LEFT JOIN unit clu ON clu.id = clq.unit_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'customer'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.alias LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
    OR ar.notes LIKE sqlc.narg('search_query')
    OR ab.support_email LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
  )
  AND (
    sqlc.arg('include_pricing_group_filter') = false
    OR EXISTS (
      SELECT 1 FROM account_relation_price_group arpg
      WHERE arpg.account_relation_id = ar.id
      AND arpg.account_group_id IN (sqlc.slice('pricing_group_ids'))
    )
  )
  AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
  )
  AND (
    sqlc.arg('include_status_filter') = false
    OR ar.account_status_code IN (sqlc.slice('status_codes'))
  )
  AND (
    sqlc.arg('include_shipping_term_filter') = false
    OR ar.shipping_term_id IN (sqlc.slice('shipping_term_ids'))
  )
  AND (
    sqlc.arg('include_payment_term_filter') = false
    OR ar.payment_term_id IN (sqlc.slice('payment_term_ids'))
  )
  AND (
    sqlc.arg('include_commission_status_filter') = false
    OR ar.commission_status_code IN (sqlc.slice('commission_status_codes'))
  )
  AND (
    sqlc.arg('include_freight_status_filter') = false
    OR ar.freight_status_code IN (sqlc.slice('freight_status_codes'))
  )
  AND (
    sqlc.arg('include_carrier_filter') = false
    OR ar.default_carrier_id IN (sqlc.slice('carrier_ids'))
  )
  AND (
    sqlc.arg('include_carrier_option_filter') = false
    OR ar.default_carrier_option_id IN (sqlc.slice('carrier_option_ids'))
  )
  AND (
    sqlc.arg('include_parent_account_filter') = false
    OR (
      sqlc.arg('parent_account_filter_value') = true
      AND EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
    OR (
      sqlc.arg('parent_account_filter_value') = false
      AND NOT EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
  )
  AND (
    (sqlc.narg('city') IS NULL AND sqlc.narg('state') IS NULL AND sqlc.narg('postal_code') IS NULL)
    OR EXISTS (
      SELECT 1 FROM account_address aa
      JOIN address addr ON addr.id = aa.address_id
      JOIN geolocation g ON g.id = addr.geolocation_id
      WHERE aa.account_id = ar.counterparty_account_id
      AND (sqlc.narg('city') IS NULL OR g.locality = sqlc.narg('city'))
      AND (sqlc.narg('state') IS NULL OR g.state = sqlc.narg('state'))
      AND (sqlc.narg('postal_code') IS NULL OR g.postal_code = sqlc.narg('postal_code'))
    )
  )
  AND (sqlc.narg('start_date') IS NULL OR ar.created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date') IS NULL OR ar.created_at <= sqlc.narg('end_date'))
  AND (
    sqlc.narg('cursor_created_at') IS NULL
    OR ar.created_at < sqlc.narg('cursor_created_at')
    OR (ar.created_at = sqlc.narg('cursor_created_at') AND ar.counterparty_account_id < sqlc.narg('cursor_id'))
  )
ORDER BY ar.created_at DESC, ar.counterparty_account_id DESC
LIMIT ?;

-- name: ListCustomersBackward :many
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
    ar.external_number,
    ar.is_edi_enabled,
    ar.notes,
    ar.account_status_code AS status,
    ar.commission_status_code,
    ar.freight_status_code,
    ar.carrier_billing_type,
    ar.carrier_billing_account,
    ar.stripe_customer_id,
    ar.stripe_email,
    ab.support_email AS email,
    ab.phone_number,
    ab.website_url,
    EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id) AS is_parent_account,
    ar.parent_account_relation_id,
    pa.name AS parent_account_name,
    par.counterparty_account_id AS parent_account_id,
    par.external_number AS parent_account_number,
    c.id AS default_carrier_id,
    c.name AS default_carrier_name,
    c.is_portal_enabled AS default_carrier_is_portal_enabled,
    c.created_at AS default_carrier_created_at,
    c.updated_at AS default_carrier_updated_at,
    co.id AS default_carrier_option_id,
    co.name AS default_carrier_option_name,
    co.service_level_token AS default_carrier_option_service_level_token,
    co.is_portal_enabled AS default_carrier_option_is_portal_enabled,
    co.created_at AS default_carrier_option_created_at,
    co.updated_at AS default_carrier_option_updated_at,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    pt.created_at AS payment_term_created_at,
    pt.updated_at AS payment_term_updated_at,
    st.id AS shipping_term_id,
    st.name AS shipping_term_name,
    st.is_freight_exempt AS shipping_term_is_freight_exempt,
    st.is_carrier_rate AS shipping_term_is_carrier_rate,
    st.created_at AS shipping_term_created_at,
    st.updated_at AS shipping_term_updated_at,
    p.id AS priority_id,
    p.code AS priority_code,
    p.name AS priority_name,
    sr.id AS default_sales_rep_id,
    sru.name AS default_sales_rep_name,
    sr.status_code AS default_sales_rep_status_code,
    sr.created_at AS default_sales_rep_created_at,
    sr.updated_at AS default_sales_rep_updated_at,
    tg.id AS type_group_id,
    tg.name AS type_group_name,
    tg.commission_status_code AS type_group_commission_status_code,
    tg.freight_status_code AS type_group_freight_status_code,
    tg.account_group_type_code AS type_group_type_code,
    tg.created_at AS type_group_created_at,
    tg.updated_at AS type_group_updated_at,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    par.created_at AS parent_account_created_at,
    par.updated_at AS parent_account_updated_at,
    clq.id AS credit_limit_id,
    clq.value AS credit_limit_value,
    clu.id AS credit_limit_unit_id,
    clu.abbreviation AS credit_limit_unit_abbreviation,
    clu.name AS credit_limit_unit_name,
    clu.unit_dimension_code AS credit_limit_unit_type,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
LEFT JOIN account pa ON pa.id = par.counterparty_account_id
LEFT JOIN carrier c ON c.id = ar.default_carrier_id
LEFT JOIN carrier_option co ON co.id = ar.default_carrier_option_id
LEFT JOIN payment_term pt ON pt.id = ar.payment_term_id
LEFT JOIN shipping_term st ON st.id = ar.shipping_term_id
LEFT JOIN priority p ON p.code = ar.priority_code
LEFT JOIN account_user sr ON sr.id = ar.default_sales_rep_id
LEFT JOIN `user` sru ON sru.id = sr.user_id
LEFT JOIN account_group tg ON tg.id = ar.account_group_id AND tg.account_group_type_code = 'type_group'
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
LEFT JOIN quantity clq ON clq.id = ar.credit_limit_id
LEFT JOIN unit clu ON clu.id = clq.unit_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'customer'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.alias LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
    OR ar.notes LIKE sqlc.narg('search_query')
    OR ab.support_email LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
  )
  AND (
    sqlc.arg('include_pricing_group_filter') = false
    OR EXISTS (
      SELECT 1 FROM account_relation_price_group arpg
      WHERE arpg.account_relation_id = ar.id
      AND arpg.account_group_id IN (sqlc.slice('pricing_group_ids'))
    )
  )
  AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
  )
  AND (
    sqlc.arg('include_status_filter') = false
    OR ar.account_status_code IN (sqlc.slice('status_codes'))
  )
  AND (
    sqlc.arg('include_shipping_term_filter') = false
    OR ar.shipping_term_id IN (sqlc.slice('shipping_term_ids'))
  )
  AND (
    sqlc.arg('include_payment_term_filter') = false
    OR ar.payment_term_id IN (sqlc.slice('payment_term_ids'))
  )
  AND (
    sqlc.arg('include_commission_status_filter') = false
    OR ar.commission_status_code IN (sqlc.slice('commission_status_codes'))
  )
  AND (
    sqlc.arg('include_freight_status_filter') = false
    OR ar.freight_status_code IN (sqlc.slice('freight_status_codes'))
  )
  AND (
    sqlc.arg('include_carrier_filter') = false
    OR ar.default_carrier_id IN (sqlc.slice('carrier_ids'))
  )
  AND (
    sqlc.arg('include_carrier_option_filter') = false
    OR ar.default_carrier_option_id IN (sqlc.slice('carrier_option_ids'))
  )
  AND (
    sqlc.arg('include_parent_account_filter') = false
    OR (
      sqlc.arg('parent_account_filter_value') = true
      AND EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
    OR (
      sqlc.arg('parent_account_filter_value') = false
      AND NOT EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
  )
  AND (
    (sqlc.narg('city') IS NULL AND sqlc.narg('state') IS NULL AND sqlc.narg('postal_code') IS NULL)
    OR EXISTS (
      SELECT 1 FROM account_address aa
      JOIN address addr ON addr.id = aa.address_id
      JOIN geolocation g ON g.id = addr.geolocation_id
      WHERE aa.account_id = ar.counterparty_account_id
      AND (sqlc.narg('city') IS NULL OR g.locality = sqlc.narg('city'))
      AND (sqlc.narg('state') IS NULL OR g.state = sqlc.narg('state'))
      AND (sqlc.narg('postal_code') IS NULL OR g.postal_code = sqlc.narg('postal_code'))
    )
  )
  AND (sqlc.narg('start_date') IS NULL OR ar.created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date') IS NULL OR ar.created_at <= sqlc.narg('end_date'))
  AND (
    ar.created_at > sqlc.arg('cursor_created_at')
    OR (ar.created_at = sqlc.arg('cursor_created_at') AND ar.counterparty_account_id > sqlc.arg('cursor_id'))
  )
ORDER BY ar.created_at ASC, ar.counterparty_account_id ASC
LIMIT ?;

-- name: ListCustomersPriceGroups :many
SELECT
    arpg.account_relation_id,
    ag.id,
    ag.name,
    ag.commission_status_code,
    ag.freight_status_code,
    ag.account_group_type_code,
    ag.created_at,
    ag.updated_at
FROM account_relation_price_group arpg
INNER JOIN account_group ag ON ag.id = arpg.account_group_id
WHERE arpg.account_relation_id IN (sqlc.slice('relation_ids'));

-- name: ListCustomersNotificationPreferences :many
SELECT
    arnp.account_relation_id,
    arnp.notification_type_code
FROM account_relation_notification_preference arnp
WHERE arnp.account_relation_id IN (sqlc.slice('relation_ids'));

-- name: CountCustomers :one
SELECT COUNT(*) AS total
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.account_relation_role_code = 'customer'
  AND (
    sqlc.narg('search_query') IS NULL
    OR a.name LIKE sqlc.narg('search_query')
    OR ar.alias LIKE sqlc.narg('search_query')
    OR ar.external_number LIKE sqlc.narg('search_query')
    OR ar.notes LIKE sqlc.narg('search_query')
    OR ab.support_email LIKE sqlc.narg('search_query')
  )
  AND (
    sqlc.arg('include_customer_group_filter') = false
    OR ar.account_group_id IN (sqlc.slice('customer_group_ids'))
  )
  AND (
    sqlc.arg('include_pricing_group_filter') = false
    OR EXISTS (
      SELECT 1 FROM account_relation_price_group arpg
      WHERE arpg.account_relation_id = ar.id
      AND arpg.account_group_id IN (sqlc.slice('pricing_group_ids'))
    )
  )
  AND (
    sqlc.arg('include_sales_rep_filter') = false
    OR ar.default_sales_rep_id IN (sqlc.slice('sales_rep_ids'))
  )
  AND (
    sqlc.arg('include_status_filter') = false
    OR ar.account_status_code IN (sqlc.slice('status_codes'))
  )
  AND (
    sqlc.arg('include_shipping_term_filter') = false
    OR ar.shipping_term_id IN (sqlc.slice('shipping_term_ids'))
  )
  AND (
    sqlc.arg('include_payment_term_filter') = false
    OR ar.payment_term_id IN (sqlc.slice('payment_term_ids'))
  )
  AND (
    sqlc.arg('include_commission_status_filter') = false
    OR ar.commission_status_code IN (sqlc.slice('commission_status_codes'))
  )
  AND (
    sqlc.arg('include_freight_status_filter') = false
    OR ar.freight_status_code IN (sqlc.slice('freight_status_codes'))
  )
  AND (
    sqlc.arg('include_carrier_filter') = false
    OR ar.default_carrier_id IN (sqlc.slice('carrier_ids'))
  )
  AND (
    sqlc.arg('include_carrier_option_filter') = false
    OR ar.default_carrier_option_id IN (sqlc.slice('carrier_option_ids'))
  )
  AND (
    sqlc.arg('include_parent_account_filter') = false
    OR (
      sqlc.arg('parent_account_filter_value') = true
      AND EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
    OR (
      sqlc.arg('parent_account_filter_value') = false
      AND NOT EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id)
    )
  )
  AND (
    (sqlc.narg('city') IS NULL AND sqlc.narg('state') IS NULL AND sqlc.narg('postal_code') IS NULL)
    OR EXISTS (
      SELECT 1 FROM account_address aa
      JOIN address addr ON addr.id = aa.address_id
      JOIN geolocation g ON g.id = addr.geolocation_id
      WHERE aa.account_id = ar.counterparty_account_id
      AND (sqlc.narg('city') IS NULL OR g.locality = sqlc.narg('city'))
      AND (sqlc.narg('state') IS NULL OR g.state = sqlc.narg('state'))
      AND (sqlc.narg('postal_code') IS NULL OR g.postal_code = sqlc.narg('postal_code'))
    )
  )
  AND (sqlc.narg('start_date') IS NULL OR ar.created_at >= sqlc.narg('start_date'))
  AND (sqlc.narg('end_date') IS NULL OR ar.created_at <= sqlc.narg('end_date'));

-- name: GetCustomer :one
SELECT
    ar.id AS relation_id,
    ar.counterparty_account_id AS account_id,
    COALESCE(NULLIF(ar.alias, ''), a.name) AS account_name,
    ar.external_number,
    ar.is_edi_enabled,
    ar.notes,
    ar.account_status_code AS status,
    ar.commission_status_code,
    ar.freight_status_code,
    ar.carrier_billing_type,
    ar.carrier_billing_account,
    ar.stripe_customer_id,
    ar.stripe_email,
    ab.support_email AS email,
    ab.phone_number,
    ab.website_url,
    EXISTS (SELECT 1 FROM account_relation car WHERE car.parent_account_relation_id = ar.id) AS is_parent_account,
    ar.parent_account_relation_id,
    pa.name AS parent_account_name,
    par.counterparty_account_id AS parent_account_id,
    c.id AS default_carrier_id,
    c.name AS default_carrier_name,
    c.is_portal_enabled AS default_carrier_is_portal_enabled,
    c.created_at AS default_carrier_created_at,
    c.updated_at AS default_carrier_updated_at,
    co.id AS default_carrier_option_id,
    co.name AS default_carrier_option_name,
    co.service_level_token AS default_carrier_option_service_level_token,
    co.is_portal_enabled AS default_carrier_option_is_portal_enabled,
    co.created_at AS default_carrier_option_created_at,
    co.updated_at AS default_carrier_option_updated_at,
    pt.id AS payment_term_id,
    pt.name AS payment_term_name,
    pt.is_active AS payment_term_is_active,
    pt.created_at AS payment_term_created_at,
    pt.updated_at AS payment_term_updated_at,
    st.id AS shipping_term_id,
    st.name AS shipping_term_name,
    st.is_freight_exempt AS shipping_term_is_freight_exempt,
    st.is_carrier_rate AS shipping_term_is_carrier_rate,
    st.created_at AS shipping_term_created_at,
    st.updated_at AS shipping_term_updated_at,
    p.id AS priority_id,
    p.code AS priority_code,
    p.name AS priority_name,
    sr.id AS default_sales_rep_id,
    sru.name AS default_sales_rep_name,
    sr.status_code AS default_sales_rep_status_code,
    sr.created_at AS default_sales_rep_created_at,
    sr.updated_at AS default_sales_rep_updated_at,
    tg.id AS type_group_id,
    tg.name AS type_group_name,
    tg.commission_status_code AS type_group_commission_status_code,
    tg.freight_status_code AS type_group_freight_status_code,
    tg.account_group_type_code AS type_group_type_code,
    tg.created_at AS type_group_created_at,
    tg.updated_at AS type_group_updated_at,
    par.external_number AS parent_account_number,
    par.created_at AS parent_account_created_at,
    par.updated_at AS parent_account_updated_at,
    ba.id AS default_billing_address_id,
    ba.name AS default_billing_address_name,
    ba.phone AS default_billing_address_phone,
    ba.email AS default_billing_address_email,
    ba.is_drop_ship AS default_billing_is_drop_ship,
    bg.id AS default_billing_geolocation_id,
    bg.street_line_1 AS default_billing_street_line_1,
    bg.street_line_2 AS default_billing_street_line_2,
    bg.locality AS default_billing_locality,
    bg.state AS default_billing_state,
    bg.postal_code AS default_billing_postal_code,
    bg.country AS default_billing_country,
    ba.created_at AS default_billing_address_created_at,
    ba.updated_at AS default_billing_address_updated_at,
    sa.id AS default_shipping_address_id,
    sa.name AS default_shipping_address_name,
    sa.phone AS default_shipping_address_phone,
    sa.email AS default_shipping_address_email,
    sa.is_drop_ship AS default_shipping_is_drop_ship,
    sg.id AS default_shipping_geolocation_id,
    sg.street_line_1 AS default_shipping_street_line_1,
    sg.street_line_2 AS default_shipping_street_line_2,
    sg.locality AS default_shipping_locality,
    sg.state AS default_shipping_state,
    sg.postal_code AS default_shipping_postal_code,
    sg.country AS default_shipping_country,
    sa.created_at AS default_shipping_address_created_at,
    sa.updated_at AS default_shipping_address_updated_at,
    clq.id AS credit_limit_id,
    clq.value AS credit_limit_value,
    clu.id AS credit_limit_unit_id,
    clu.abbreviation AS credit_limit_unit_abbreviation,
    clu.name AS credit_limit_unit_name,
    clu.unit_dimension_code AS credit_limit_unit_type,
    ar.created_at,
    ar.updated_at
FROM account_relation ar
INNER JOIN account a ON a.id = ar.counterparty_account_id
LEFT JOIN account_branding ab ON ab.owner_account_id = ar.counterparty_account_id
LEFT JOIN account_relation par ON par.id = ar.parent_account_relation_id
LEFT JOIN account pa ON pa.id = par.counterparty_account_id
LEFT JOIN carrier c ON c.id = ar.default_carrier_id
LEFT JOIN carrier_option co ON co.id = ar.default_carrier_option_id
LEFT JOIN payment_term pt ON pt.id = ar.payment_term_id
LEFT JOIN shipping_term st ON st.id = ar.shipping_term_id
LEFT JOIN priority p ON p.code = ar.priority_code
LEFT JOIN account_user sr ON sr.id = ar.default_sales_rep_id
LEFT JOIN `user` sru ON sru.id = sr.user_id
LEFT JOIN account_group tg ON tg.id = ar.account_group_id AND tg.account_group_type_code = 'type_group'
LEFT JOIN address ba ON ba.id = ar.default_billing_address_id
LEFT JOIN geolocation bg ON bg.id = ba.geolocation_id
LEFT JOIN address sa ON sa.id = ar.default_shipping_address_id
LEFT JOIN geolocation sg ON sg.id = sa.geolocation_id
LEFT JOIN quantity clq ON clq.id = ar.credit_limit_id
LEFT JOIN unit clu ON clu.id = clq.unit_id
WHERE ar.owner_account_id = sqlc.arg('owner_account_id')
  AND ar.counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND ar.account_relation_role_code = 'customer';

-- name: GetCustomerPriceGroups :many
SELECT
    ag.id,
    ag.name,
    ag.commission_status_code,
    ag.freight_status_code,
    ag.account_group_type_code,
    ag.created_at,
    ag.updated_at
FROM account_relation_price_group arpg
INNER JOIN account_group ag ON ag.id = arpg.account_group_id
WHERE arpg.account_relation_id = sqlc.arg('account_relation_id');

-- name: GetCustomerNotificationPreferences :many
SELECT
    arnp.id,
    arnp.notification_type_code
FROM account_relation_notification_preference arnp
WHERE arnp.account_relation_id = sqlc.arg('account_relation_id');

-- name: InsertAccountRelation :exec
INSERT INTO account_relation (
    id, owner_account_id, counterparty_account_id, account_relation_role_code,
    external_number, is_edi_enabled, notes, parent_account_relation_id,
    commission_status_code, freight_status_code,
    default_carrier_id, default_carrier_option_id, default_sales_rep_id,
    account_status_code, payment_term_id, account_group_id, priority_code,
    shipping_term_id, carrier_billing_type, carrier_billing_account,
    default_billing_address_id, default_shipping_address_id,
    stripe_customer_id, stripe_email,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('counterparty_account_id'), 'customer',
    sqlc.narg('external_number'), sqlc.arg('is_edi_enabled'), sqlc.narg('notes'), sqlc.narg('parent_account_relation_id'),
    sqlc.narg('commission_status_code'), sqlc.narg('freight_status_code'),
    sqlc.narg('default_carrier_id'), sqlc.narg('default_carrier_option_id'), sqlc.narg('default_sales_rep_id'),
    sqlc.arg('account_status_code'), sqlc.narg('payment_term_id'), sqlc.narg('account_group_id'), sqlc.narg('priority_code'),
    sqlc.narg('shipping_term_id'), sqlc.narg('carrier_billing_type'), sqlc.narg('carrier_billing_account'),
    sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'),
    sqlc.narg('stripe_customer_id'), sqlc.narg('stripe_email'),
    NOW(3), NOW(3)
);

-- name: UpdateCustomer :exec
UPDATE account_relation SET
    alias = COALESCE(sqlc.narg('alias'), alias),
    external_number = COALESCE(sqlc.narg('external_number'), external_number),
    is_edi_enabled = COALESCE(sqlc.narg('is_edi_enabled'), is_edi_enabled),
    notes = sqlc.narg('notes'),
    parent_account_relation_id = COALESCE(sqlc.narg('parent_account_relation_id'), parent_account_relation_id),
    commission_status_code = COALESCE(sqlc.narg('commission_status_code'), commission_status_code),
    freight_status_code = COALESCE(sqlc.narg('freight_status_code'), freight_status_code),
    default_carrier_id = sqlc.narg('default_carrier_id'),
    default_carrier_option_id = sqlc.narg('default_carrier_option_id'),
    default_sales_rep_id = sqlc.narg('default_sales_rep_id'),
    account_status_code = COALESCE(sqlc.narg('account_status_code'), account_status_code),
    payment_term_id = sqlc.narg('payment_term_id'),
    account_group_id = sqlc.narg('account_group_id'),
    priority_code = COALESCE(sqlc.narg('priority_code'), priority_code),
    shipping_term_id = sqlc.narg('shipping_term_id'),
    carrier_billing_type = COALESCE(sqlc.narg('carrier_billing_type'), carrier_billing_type),
    carrier_billing_account = sqlc.narg('carrier_billing_account'),
    default_billing_address_id = sqlc.narg('default_billing_address_id'),
    default_shipping_address_id = sqlc.narg('default_shipping_address_id'),
    credit_limit_id = sqlc.narg('credit_limit_id'),
    stripe_customer_id = COALESCE(sqlc.narg('stripe_customer_id'), stripe_customer_id),
    stripe_email = COALESCE(sqlc.narg('stripe_email'), stripe_email),
    updated_at = NOW(3)
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id')
  AND account_relation_role_code = 'customer';

-- name: DeleteCustomer :exec
DELETE FROM account_relation
WHERE id = sqlc.arg('id')
  AND owner_account_id = sqlc.arg('owner_account_id')
  AND account_relation_role_code = 'customer';

-- name: CustomerExistsByExternalNumber :one
SELECT COUNT(*) > 0 AS customer_exists FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
AND external_number = sqlc.arg('external_number')
AND account_relation_role_code = 'customer'
AND (sqlc.narg('exclude_counterparty_id') IS NULL OR counterparty_account_id != sqlc.narg('exclude_counterparty_id'));

-- name: InsertCustomerAccount :exec
INSERT INTO account (id, name, account_type_code, onboarding_status_code,
    default_billing_address_id, default_shipping_address_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('name'), 'company', 'unclaimed',
    sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'), NOW(3), NOW(3));

-- name: InsertCustomerAccountBranding :exec
INSERT INTO account_branding (id, owner_account_id, support_email, phone_number, website_url, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.narg('support_email'), sqlc.narg('phone_number'), sqlc.narg('website_url'), NOW(3), NOW(3));

-- name: InsertCustomerRelation :exec
INSERT INTO account_relation (
    id, owner_account_id, counterparty_account_id, account_relation_role_code,
    alias, external_number, notes, is_edi_enabled,
    commission_status_code, freight_status_code,
    default_carrier_id, default_carrier_option_id, default_sales_rep_id,
    account_status_code, payment_term_id, account_group_id, priority_code,
    shipping_term_id, carrier_billing_type, carrier_billing_account,
    default_billing_address_id, default_shipping_address_id,
    credit_limit_id,
    created_at, updated_at
) VALUES (
    sqlc.arg('id'), sqlc.arg('owner_account_id'), sqlc.arg('counterparty_account_id'), 'customer',
    sqlc.arg('alias'), sqlc.arg('external_number'), sqlc.narg('notes'), sqlc.arg('is_edi_enabled'),
    sqlc.arg('commission_status_code'), sqlc.arg('freight_status_code'),
    sqlc.narg('default_carrier_id'), sqlc.narg('default_carrier_option_id'), sqlc.narg('default_sales_rep_id'),
    sqlc.arg('account_status_code'), sqlc.narg('payment_term_id'), sqlc.narg('account_group_id'), sqlc.narg('priority_code'),
    sqlc.narg('shipping_term_id'), sqlc.narg('carrier_billing_type'), sqlc.narg('carrier_billing_account'),
    sqlc.narg('default_billing_address_id'), sqlc.narg('default_shipping_address_id'),
    sqlc.narg('credit_limit_id'),
    NOW(3), NOW(3)
);

-- name: DeleteCustomerByAccountID :exec
DELETE FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'customer';

-- name: DeleteCustomerAccountUsers :exec
DELETE FROM account_user
WHERE account_id = sqlc.arg('account_id');

-- name: DeleteCustomerAccountAddresses :exec
DELETE FROM account_address
WHERE account_id = sqlc.arg('account_id');

-- name: BulkDeleteCustomerRelations :exec
DELETE FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id IN (sqlc.slice('counterparty_account_ids'))
  AND account_relation_role_code = 'customer';

-- name: GetCustomerRelationID :one
SELECT id FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'customer';

-- name: GetFrequentlyOrderedProducts :many
WITH ranked AS (
    SELECT
        fg.item_id AS item_id,
        it.description AS product_name,
        u.id AS unit_id,
        u.abbreviation AS unit_abbreviation,
        COUNT(*) AS order_count,
        ROW_NUMBER() OVER (PARTITION BY fg.item_id ORDER BY COUNT(*) DESC) AS rn
    FROM sales_order_line sol
    JOIN sales_order so ON so.id = sol.sales_order_id
    JOIN product fg ON fg.id = sol.product_id
    JOIN item it ON it.id = fg.item_id
    JOIN quantity q ON q.id = sol.quantity_id
    JOIN unit u ON u.id = q.unit_id
    WHERE so.owner_account_id = sqlc.arg('owner_account_id')
      AND so.buyer_account_id = sqlc.arg('buyer_account_id')
      AND fg.product_type_code = 'sale'
      AND fg.is_portal_ready = 1
    GROUP BY fg.item_id, it.description, u.id, u.abbreviation
)
SELECT item_id, product_name, unit_id, unit_abbreviation, order_count
FROM ranked
WHERE rn = 1
ORDER BY order_count DESC
LIMIT 12;

-- name: MergeCustomerOrders :exec
UPDATE sales_order SET buyer_account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND buyer_account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerInvoices :exec
UPDATE invoice i
JOIN sales_order so ON so.id = i.sales_order_id
SET i.account_id = sqlc.arg('target_account_id'),
    i.updated_at = NOW(3)
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND i.account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerShipments :exec
UPDATE shipment s
JOIN sales_order so ON so.id = s.sales_order_id
SET s.account_id = sqlc.arg('target_account_id'),
    s.updated_at = NOW(3)
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND s.account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerDeliveries :exec
UPDATE delivery d
JOIN sales_order so ON so.id = d.sales_order_id
SET d.account_id = sqlc.arg('target_account_id'),
    d.updated_at = NOW(3)
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND d.account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerTransactions :exec
UPDATE transaction SET customer_account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE account_id = sqlc.arg('owner_account_id')
  AND customer_account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerAccountPrices :exec
UPDATE account_price SET recipient_account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND recipient_account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerInventoryReceipts :exec
UPDATE inventory_receipt SET holder_account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND holder_account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerReceivingOrders :exec
UPDATE receiving_order ro
JOIN sales_order so ON so.id = ro.order_id
SET ro.account_id = sqlc.arg('target_account_id'),
    ro.updated_at = NOW(3)
WHERE so.owner_account_id = sqlc.arg('owner_account_id')
  AND ro.account_id IN (sqlc.slice('source_account_ids'));

-- name: MergeCustomerInventoryIssues :exec
UPDATE inventory_issue SET account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE account_id IN (sqlc.slice('source_account_ids'));

-- name: DeleteCustomerNotificationPreferences :exec
DELETE FROM account_relation_notification_preference
WHERE account_relation_id IN (sqlc.slice('relation_ids'));

-- name: DeleteCustomerProductLineAccess :exec
DELETE FROM account_relation_product_line
WHERE account_relation_id IN (sqlc.slice('relation_ids'));

-- name: GetCustomerStripeCustomerID :one
SELECT stripe_customer_id, stripe_email
FROM account_relation
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'customer';

-- name: SetCustomerStripeCustomerID :exec
UPDATE account_relation SET
    stripe_customer_id = sqlc.arg('stripe_customer_id'),
    stripe_email = sqlc.arg('stripe_email'),
    updated_at = NOW(3)
WHERE owner_account_id = sqlc.arg('owner_account_id')
  AND counterparty_account_id = sqlc.arg('counterparty_account_id')
  AND account_relation_role_code = 'customer';

-- name: GetCustomerEmail :one
SELECT ab.support_email
FROM account_branding ab
WHERE ab.owner_account_id = sqlc.arg('account_id');

-- name: UpsertCustomerBranding :exec
INSERT INTO account_branding (id, owner_account_id, support_email, phone_number, website_url, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.narg('support_email'), sqlc.narg('phone_number'), sqlc.narg('website_url'), NOW(3), NOW(3))
ON DUPLICATE KEY UPDATE
    support_email = VALUES(support_email),
    phone_number = VALUES(phone_number),
    website_url = VALUES(website_url),
    updated_at = NOW(3);

-- name: InsertAccountRelationPriceGroup :exec
INSERT INTO account_relation_price_group (id, account_relation_id, account_group_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_relation_id'), sqlc.arg('account_group_id'), NOW(3), NOW(3));

-- name: DeleteAccountRelationPriceGroupsByRelationID :exec
DELETE FROM account_relation_price_group
WHERE account_relation_id = sqlc.arg('account_relation_id');

-- name: GetRelationPriceGroupIDs :many
SELECT account_group_id FROM account_relation_price_group
WHERE account_relation_id = sqlc.arg('account_relation_id');

-- name: GetRelationsPriceGroups :many
SELECT id, account_group_id, account_relation_id FROM account_relation_price_group
WHERE account_relation_id IN (sqlc.slice('relation_ids'));

-- name: MoveAccountRelationPriceGroups :exec
UPDATE account_relation_price_group
SET account_relation_id = sqlc.arg('target_relation_id'), updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteAccountRelationPriceGroupsByIDs :exec
DELETE FROM account_relation_price_group
WHERE id IN (sqlc.slice('ids'));

-- name: GetRelationProductLineIDs :many
SELECT product_line_id FROM account_relation_product_line
WHERE account_relation_id = sqlc.arg('account_relation_id');

-- name: GetRelationsProductLines :many
SELECT id, product_line_id, account_relation_id FROM account_relation_product_line
WHERE account_relation_id IN (sqlc.slice('relation_ids'));

-- name: MoveAccountRelationProductLines :exec
UPDATE account_relation_product_line
SET account_relation_id = sqlc.arg('target_relation_id'), updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteAccountRelationProductLinesByIDs :exec
DELETE FROM account_relation_product_line
WHERE id IN (sqlc.slice('ids'));

-- name: ReparentChildRelations :exec
UPDATE account_relation
SET parent_account_relation_id = sqlc.arg('target_relation_id'), updated_at = NOW(3)
WHERE parent_account_relation_id IN (sqlc.slice('source_relation_ids'));

-- name: GetAccountAddressIDs :many
SELECT address_id FROM account_address
WHERE account_id = sqlc.arg('account_id');

-- name: InsertMergeAccountAddress :exec
INSERT INTO account_address (id, account_id, address_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('account_id'), sqlc.arg('address_id'), NOW(3), NOW(3));

-- name: DeleteAccountAddressesByAccountID :exec
DELETE FROM account_address
WHERE account_id = sqlc.arg('account_id');

-- name: GetAccountUsersByAccountID :many
SELECT id, user_id FROM account_user
WHERE account_id = sqlc.arg('account_id');

-- name: MoveAccountUsers :exec
UPDATE account_user
SET account_id = sqlc.arg('target_account_id'), updated_at = NOW(3)
WHERE id IN (sqlc.slice('ids'));

-- name: DeleteAccountUsersByAccountID :exec
DELETE FROM account_user
WHERE account_id = sqlc.arg('account_id');

-- name: InsertCustomerCreditLimitQuantity :exec
INSERT INTO quantity (id, value, unit_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('value'), sqlc.arg('unit_id'), NOW(3), NOW(3));

-- name: UpdateCustomerCreditLimitQuantity :exec
UPDATE quantity SET value = sqlc.arg('value'), unit_id = sqlc.arg('unit_id'), updated_at = NOW(3)
WHERE id = sqlc.arg('id');

-- name: DeleteCustomerCreditLimitQuantity :exec
DELETE FROM quantity WHERE id = sqlc.arg('id');
