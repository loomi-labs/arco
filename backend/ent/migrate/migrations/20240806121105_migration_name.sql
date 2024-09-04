-- Create "archives" table
CREATE TABLE `archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `created_at` datetime NOT NULL, `duration` datetime NOT NULL, `borg_id` text NOT NULL, `archive_repository` integer NOT NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE NO ACTION);
-- Create "backup_profiles" table
CREATE TABLE `backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `prefix` text NOT NULL, `backup_dirs` json NOT NULL, `exclude_dirs` json NULL, `is_setup_complete` bool NOT NULL DEFAULT (false));
-- Create "backup_schedules" table
CREATE TABLE `backup_schedules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `hourly` bool NOT NULL DEFAULT (false), `daily_at` datetime NULL, `weekday` text NULL, `weekly_at` datetime NULL, `monthday` integer NULL, `monthly_at` datetime NULL, `next_run` datetime NULL, `last_run` datetime NULL, `last_run_status` text NULL, `backup_profile_backup_schedule` integer NOT NULL, CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
-- Create "repositories" table
CREATE TABLE `repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `url` text NOT NULL, `password` text NOT NULL);
-- Create index "repositories_url_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
-- Create "backup_profile_repositories" table
CREATE TABLE `backup_profile_repositories` (`backup_profile_id` integer NOT NULL, `repository_id` integer NOT NULL, PRIMARY KEY (`backup_profile_id`, `repository_id`), CONSTRAINT `backup_profile_repositories_backup_profile_id` FOREIGN KEY (`backup_profile_id`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `backup_profile_repositories_repository_id` FOREIGN KEY (`repository_id`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
