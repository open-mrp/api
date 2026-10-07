-- +goose Up

-- The EDI permissions guarded the EDI runs and DC locations pages and endpoints, which are gone. Roles
-- that held them keep every other permission.
DELETE FROM `role_permission` WHERE `permission_code` IN ('edi_locations', 'edi_runs');
DELETE FROM `permission` WHERE `code` IN ('edi_locations', 'edi_runs');
DELETE FROM `permission_group` WHERE `code` = 'edi'
  AND NOT EXISTS (SELECT 1 FROM `permission` p WHERE p.`permission_group_code` = 'edi');

-- +goose Down

-- Not reversible: the rows named permissions that no longer exist.
SELECT 1;
