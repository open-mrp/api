-- An inventory movement carrying every reference a movement can: a location, a lot and the shipment behind it, so the
-- generic include tests can populate `location`, `lot` and `related` on one row. Dated long ago so no list that reads
-- the newest movements first meets it.
INSERT IGNORE INTO quantity (id, value, unit_id, created_at, updated_at) VALUES
    ('qu_01seedinvmvfull00000', '1', 'un_01seedpound00000000', '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000');

INSERT IGNORE INTO inventory_change_log (
    id, item_id, quantity_id, action_type_code, account_id, type_code, actor_type, actor_id,
    location_id, lot_id, reason_code, related_type, related_id, created_at, updated_at
) VALUES (
    'inmv_01seedmvfull0000', 'it_01seedyrn1item00000', 'qu_01seedinvmvfull00000', 'user_correction',
    'ac_01k0a5smf9ekb8rqg12555zjqa', 'adjustment', 'system', NULL,
    'sglc_01seedbuilding0000', 'lot_01seeddlvlot1_000', 'found', 'shipment', 'sh_01k0a87w33emw8pmkz1mf86cg1',
    '2020-01-01 00:00:00.000', '2020-01-01 00:00:00.000'
);
