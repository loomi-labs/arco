-- Create "archives" table
CREATE TABLE `archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `duration` real NOT NULL, `borg_id` text NOT NULL, `will_be_pruned` bool NOT NULL DEFAULT (false), `archive_repository` integer NOT NULL, `archive_backup_profile` integer NULL, `backup_profile_archives` integer NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE, CONSTRAINT `archives_backup_profiles_backup_profile` FOREIGN KEY (`archive_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL, CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL);
-- Create "backup_profiles" table
CREATE TABLE `backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `prefix` text NOT NULL, `backup_paths` json NOT NULL, `exclude_paths` json NULL, `icon` text NOT NULL);
-- Create index "backup_profiles_prefix_key" to table: "backup_profiles"
CREATE UNIQUE INDEX `backup_profiles_prefix_key` ON `backup_profiles` (`prefix`);
-- Create "backup_schedules" table
CREATE TABLE `backup_schedules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `mode` text NOT NULL DEFAULT ('disabled'), `daily_at` datetime NOT NULL, `weekday` text NOT NULL, `weekly_at` datetime NOT NULL, `monthday` integer NOT NULL, `monthly_at` datetime NOT NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_backup_schedule` integer NOT NULL, CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
-- Create "notifications" table
CREATE TABLE `notifications` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `message` text NOT NULL, `type` text NOT NULL, `seen` bool NOT NULL DEFAULT (false), `action` text NULL, `notification_backup_profile` integer NOT NULL, `notification_repository` integer NOT NULL, CONSTRAINT `notifications_backup_profiles_backup_profile` FOREIGN KEY (`notification_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `notifications_repositories_repository` FOREIGN KEY (`notification_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
-- Create "pruning_rules" table
CREATE TABLE `pruning_rules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `is_enabled` bool NOT NULL, `keep_hourly` integer NOT NULL, `keep_daily` integer NOT NULL, `keep_weekly` integer NOT NULL, `keep_monthly` integer NOT NULL, `keep_yearly` integer NOT NULL, `keep_within_days` integer NOT NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_pruning_rule` integer NOT NULL, CONSTRAINT `pruning_rules_backup_profiles_pruning_rule` FOREIGN KEY (`backup_profile_pruning_rule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Create index "pruning_rules_backup_profile_pruning_rule_key" to table: "pruning_rules"
CREATE UNIQUE INDEX `pruning_rules_backup_profile_pruning_rule_key` ON `pruning_rules` (`backup_profile_pruning_rule`);
-- Create "repositories" table
CREATE TABLE `repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `location` text NOT NULL, `password` text NOT NULL, `next_integrity_check` datetime NULL, `stats_total_chunks` integer NOT NULL DEFAULT (0), `stats_total_size` integer NOT NULL DEFAULT (0), `stats_total_csize` integer NOT NULL DEFAULT (0), `stats_total_unique_chunks` integer NOT NULL DEFAULT (0), `stats_unique_size` integer NOT NULL DEFAULT (0), `stats_unique_csize` integer NOT NULL DEFAULT (0));
-- Create index "repositories_name_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_name_key` ON `repositories` (`name`);
-- Create index "repositories_location_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_location_key` ON `repositories` (`location`);
-- Create "settings" table
CREATE TABLE `settings` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `show_welcome` bool NOT NULL DEFAULT (true));
-- Create "backup_profile_repositories" table
CREATE TABLE `backup_profile_repositories` (`backup_profile_id` integer NOT NULL, `repository_id` integer NOT NULL, PRIMARY KEY (`backup_profile_id`, `repository_id`), CONSTRAINT `backup_profile_repositories_backup_profile_id` FOREIGN KEY (`backup_profile_id`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `backup_profile_repositories_repository_id` FOREIGN KEY (`repository_id`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
