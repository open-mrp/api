-- +goose Up
-- Configure the one account whose OEE is wrong without it. The shift window is nullable and unset
-- everywhere else, which leaves every other account on the unclipped behaviour until it configures
-- one deliberately.
--
-- The values are read off the account's own scan history rather than asked for: every knitting scan
-- in `batch` for the last six weeks falls between 06:00 and 22:00 local, Monday to Friday, which is
-- exactly the 2 shifts x 8 hours x 5 days the settings row already claims. The zone matches the
-- account's generation_timezone (America/Indianapolis, Eastern). Raw UTC scan hours look like
-- round-the-clock operation and are not evidence of one.
--
-- Scoped by account_id so it cannot touch anyone else, and idempotent: re-running sets the same
-- values, and the IS NULL guard means it will not overwrite a window somebody has since chosen.
UPDATE account_production_schedule_setting
   SET shift_start_time = '06:00',
       shift_timezone = 'America/Indianapolis',
       shift_days_of_week = '1111100',
       updated_at = NOW(3)
 WHERE account_id = 'ac_01gf7a8200f4r985f3r9x8mhkq'
   AND shift_start_time IS NULL;

-- +goose Down
UPDATE account_production_schedule_setting
   SET shift_start_time = NULL,
       shift_timezone = NULL,
       shift_days_of_week = '1111100',
       updated_at = NOW(3)
 WHERE account_id = 'ac_01gf7a8200f4r985f3r9x8mhkq';
