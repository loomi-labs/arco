-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_profiles" table
CREATE TABLE `new_backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `prefix` text NOT NULL, `backup_paths` json NOT NULL, `exclude_paths` json NULL, `icon` text NOT NULL);
-- Copy rows from old table "backup_profiles" to new temporary table "new_backup_profiles"
INSERT INTO `new_backup_profiles` (`id`, `created_at`, `updated_at`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `icon`) SELECT `id`, `created_at`, `updated_at`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `icon` FROM `backup_profiles`;
-- Drop "backup_profiles" table after copying rows
DROP TABLE `backup_profiles`;
-- Rename temporary table "new_backup_profiles" to "backup_profiles"
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;
-- Create index "backup_profiles_prefix_key" to table: "backup_profiles"
CREATE UNIQUE INDEX `backup_profiles_prefix_key` ON `backup_profiles` (`prefix`);
-- Create "new_backup_schedules" table
CREATE TABLE `new_backup_schedules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `mode` text NOT NULL DEFAULT ('disabled'), `daily_at` datetime NOT NULL, `weekday` text NOT NULL, `weekly_at` datetime NOT NULL, `monthday` integer NOT NULL, `monthly_at` datetime NOT NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_backup_schedule` integer NOT NULL, CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Copy rows from old table "backup_schedules" to new temporary table "new_backup_schedules"
INSERT INTO `new_backup_schedules` (`id`, `created_at`, `updated_at`, `mode`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`, `next_run`, `last_run`, `last_run_status`, `backup_profile_backup_schedule`) SELECT `id`, `created_at`, `updated_at`, `mode`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`, `next_run`, `last_run`, `last_run_status`, `backup_profile_backup_schedule` FROM `backup_schedules`;
-- Drop "backup_schedules" table after copying rows
DROP TABLE `backup_schedules`;
-- Rename temporary table "new_backup_schedules" to "backup_schedules"
ALTER TABLE `new_backup_schedules` RENAME TO `backup_schedules`;
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
-- Create "new_notifications" table
CREATE TABLE `new_notifications` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `message` text NOT NULL, `type` text NOT NULL, `seen` bool NOT NULL DEFAULT (false), `action` text NULL, `notification_backup_profile` integer NOT NULL, `notification_repository` integer NOT NULL, CONSTRAINT `notifications_backup_profiles_backup_profile` FOREIGN KEY (`notification_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `notifications_repositories_repository` FOREIGN KEY (`notification_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
-- Copy rows from old table "notifications" to new temporary table "new_notifications"
INSERT INTO `new_notifications` (`id`, `created_at`, `updated_at`, `message`, `type`, `seen`, `action`, `notification_backup_profile`, `notification_repository`) SELECT `id`, `created_at`, `updated_at`, `message`, `type`, `seen`, `action`, `notification_backup_profile`, `notification_repository` FROM `notifications`;
-- Drop "notifications" table after copying rows
DROP TABLE `notifications`;
-- Rename temporary table "new_notifications" to "notifications"
ALTER TABLE `new_notifications` RENAME TO `notifications`;
-- Create "new_pruning_rules" table
CREATE TABLE `new_pruning_rules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `is_enabled` bool NOT NULL, `keep_hourly` integer NOT NULL, `keep_daily` integer NOT NULL, `keep_weekly` integer NOT NULL, `keep_monthly` integer NOT NULL, `keep_yearly` integer NOT NULL, `keep_within_days` integer NOT NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_pruning_rule` integer NOT NULL, CONSTRAINT `pruning_rules_backup_profiles_pruning_rule` FOREIGN KEY (`backup_profile_pruning_rule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Copy rows from old table "pruning_rules" to new temporary table "new_pruning_rules"
INSERT INTO `new_pruning_rules` (`id`, `created_at`, `updated_at`, `is_enabled`, `keep_hourly`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly`, `keep_within_days`, `next_run`, `last_run`, `last_run_status`, `backup_profile_pruning_rule`) SELECT `id`, `created_at`, `updated_at`, `is_enabled`, `keep_hourly`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly`, `keep_within_days`, `next_run`, `last_run`, `last_run_status`, `backup_profile_pruning_rule` FROM `pruning_rules`;
-- Drop "pruning_rules" table after copying rows
DROP TABLE `pruning_rules`;
-- Rename temporary table "new_pruning_rules" to "pruning_rules"
ALTER TABLE `new_pruning_rules` RENAME TO `pruning_rules`;
-- Create index "pruning_rules_backup_profile_pruning_rule_key" to table: "pruning_rules"
CREATE UNIQUE INDEX `pruning_rules_backup_profile_pruning_rule_key` ON `pruning_rules` (`backup_profile_pruning_rule`);
-- Create "new_repositories" table
CREATE TABLE `new_repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `location` text NOT NULL, `password` text NOT NULL, `next_integrity_check` datetime NULL, `stats_total_chunks` integer NOT NULL DEFAULT (0), `stats_total_size` integer NOT NULL DEFAULT (0), `stats_total_csize` integer NOT NULL DEFAULT (0), `stats_total_unique_chunks` integer NOT NULL DEFAULT (0), `stats_unique_size` integer NOT NULL DEFAULT (0), `stats_unique_csize` integer NOT NULL DEFAULT (0));
-- Copy rows from old table "repositories" to new temporary table "new_repositories"
INSERT INTO `new_repositories` (`id`, `created_at`, `updated_at`, `name`, `location`, `password`, `next_integrity_check`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`) SELECT `id`, `created_at`, `updated_at`, `name`, `location`, `password`, `next_integrity_check`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize` FROM `repositories`;
-- Drop "repositories" table after copying rows
DROP TABLE `repositories`;
-- Rename temporary table "new_repositories" to "repositories"
ALTER TABLE `new_repositories` RENAME TO `repositories`;
-- Create index "repositories_location_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_location_key` ON `repositories` (`location`);
-- Create "new_settings" table
CREATE TABLE `new_settings` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `theme` text NOT NULL DEFAULT ('system'), `show_welcome` bool NOT NULL DEFAULT (true));
-- Copy rows from old table "settings" to new temporary table "new_settings"
INSERT INTO `new_settings` (`id`, `created_at`, `updated_at`, `theme`, `show_welcome`) SELECT `id`, `created_at`, `updated_at`, `theme`, `show_welcome` FROM `settings`;
-- Drop "settings" table after copying rows
DROP TABLE `settings`;
-- Rename temporary table "new_settings" to "settings"
ALTER TABLE `new_settings` RENAME TO `settings`;
-- Create "new_archives" table
CREATE TABLE `new_archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `duration` real NOT NULL, `borg_id` text NOT NULL, `will_be_pruned` bool NOT NULL DEFAULT (false), `archive_repository` integer NOT NULL, `archive_backup_profile` integer NULL, `backup_profile_archives` integer NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE, CONSTRAINT `archives_backup_profiles_backup_profile` FOREIGN KEY (`archive_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL, CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL);
-- Copy rows from old table "archives" to new temporary table "new_archives"
INSERT INTO `new_archives` (`id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives`) SELECT `id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives` FROM `archives`;
-- Drop "archives" table after copying rows
DROP TABLE `archives`;
-- Rename temporary table "new_archives" to "archives"
ALTER TABLE `new_archives` RENAME TO `archives`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
