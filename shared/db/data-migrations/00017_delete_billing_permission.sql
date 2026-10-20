-- +goose Up

-- The billing permission guarded the agent spend and spending cap endpoints, which are gone. Roles that held
-- it keep every other permission.
DELETE FROM `role_permission` WHERE `permission_code` = 'billing';
DELETE FROM `permission` WHERE `code` = 'billing';

-- +goose Down

-- Not reversible: the rows named a permission that no longer exists.
SELECT 1;
