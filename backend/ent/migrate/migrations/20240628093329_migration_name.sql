-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_profiles" table
CREATE TABLE `new_backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `prefix` text NOT NULL, `directories` json NOT NULL, `is_setup_complete` bool NOT NULL DEFAULT (false));
-- Copy rows from old table "backup_profiles" to new temporary table "new_backup_profiles"
INSERT INTO `new_backup_profiles` (`id`, `name`, `prefix`, `directories`, `is_setup_complete`) SELECT `id`, `name`, `prefix`, `directories`, `is_setup_complete` FROM `backup_profiles`;
-- Drop "backup_profiles" table after copying rows
DROP TABLE `backup_profiles`;
-- Rename temporary table "new_backup_profiles" to "backup_profiles"
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;
-- Create "new_backup_schedules" table
CREATE TABLE `new_backup_schedules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `hourly` bool NOT NULL DEFAULT (false), `daily_at` datetime NULL, `weekly_day` text NULL, `weekly_at` datetime NULL, `monthly_day` integer NULL, `monthly_at` datetime NULL, `backup_profile_backup_schedule` integer NOT NULL, CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE NO ACTION);
-- Copy rows from old table "backup_schedules" to new temporary table "new_backup_schedules"
INSERT INTO `new_backup_schedules` (`id`, `hourly`, `daily_at`, `weekly_day`, `weekly_at`, `monthly_day`, `monthly_at`, `backup_profile_backup_schedule`) SELECT `id`, `hourly`, `daily_at`, `weekly_day`, `weekly_at`, `monthly_day`, `monthly_at`, `backup_schedule_backup_profile` FROM `backup_schedules`;
-- Drop "backup_schedules" table after copying rows
DROP TABLE `backup_schedules`;
-- Rename temporary table "new_backup_schedules" to "backup_schedules"
ALTER TABLE `new_backup_schedules` RENAME TO `backup_schedules`;
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
