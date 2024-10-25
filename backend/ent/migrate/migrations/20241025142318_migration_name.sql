-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_schedules" table
CREATE TABLE `new_backup_schedules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `updated_at` datetime NOT NULL, `mode` text NOT NULL DEFAULT ('disabled'), `hourly` bool NOT NULL DEFAULT (false), `daily_at` datetime NOT NULL, `weekday` text NOT NULL, `weekly_at` datetime NOT NULL, `monthday` integer NOT NULL, `monthly_at` datetime NOT NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_backup_schedule` integer NOT NULL, CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Copy rows from old table "backup_schedules" to new temporary table "new_backup_schedules"
INSERT INTO `new_backup_schedules` (`id`, `updated_at`, `mode`, `hourly`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`, `next_run`, `last_run`, `last_run_status`, `backup_profile_backup_schedule`) SELECT `id`, `updated_at`, `mode`, `hourly`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`, `next_run`, `last_run`, `last_run_status`, `backup_profile_backup_schedule` FROM `backup_schedules`;
-- Drop "backup_schedules" table after copying rows
DROP TABLE `backup_schedules`;
-- Rename temporary table "new_backup_schedules" to "backup_schedules"
ALTER TABLE `new_backup_schedules` RENAME TO `backup_schedules`;
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
