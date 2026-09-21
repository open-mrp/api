-- +goose NO TRANSACTION
-- +goose Up
-- OEE capacity is shifts_per_day x hours_per_shift x work_days_per_week, which says how MUCH time a
-- machine has but never WHEN. Downtime is stored as UTC instants, so the two could not be intersected
-- and a stop was charged its whole wall-clock span: a 16-hour breakdown logged 15:00 Thursday to 07:00
-- Friday ate 16 hours of a day that only holds 16, though only 8 of them were inside the shift. That
-- deflates run time, which is the Performance denominator, so P climbed above 100% on exactly the weeks
-- with the most overnight downtime.
--
-- These three columns give the capacity a calendar so an event can be clipped to the time the plant was
-- actually open. Naming and semantics mirror operating_calendar, which already models an open-days mask
-- and an IANA zone for shipping and receiving: days are seven characters of '0'/'1' Monday first, and
-- the time is a local "HH:MM" string, as cutoff_at is.
--
-- start and zone are nullable on purpose and are the gate: an account that has not told us when its day
-- begins gets today's unclipped behaviour rather than a guessed window, because defaulting them would
-- silently re-scale the OEE of every account that never configured a shift.
ALTER TABLE `account_production_schedule_setting`
  ADD COLUMN `shift_start_time` varchar(8) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `shift_timezone` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN `shift_days_of_week` varchar(7) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '1111100';

-- +goose Down
ALTER TABLE `account_production_schedule_setting`
  DROP COLUMN `shift_start_time`,
  DROP COLUMN `shift_timezone`,
  DROP COLUMN `shift_days_of_week`;
