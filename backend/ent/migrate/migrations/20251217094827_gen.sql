-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- atlas:nolint DS103 MF101
-- Tokens are migrated to system keyring before this migration runs (see migrate_credentials.go)
-- Unique indexes already existed, recreating them is safe
-- Create "new_users" table
CREATE TABLE `new_users` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `email` text NOT NULL, `last_logged_in` datetime NULL, `access_token_expires_at` datetime NULL, `refresh_token_expires_at` datetime NULL);
-- Copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`id`, `created_at`, `updated_at`, `email`, `last_logged_in`, `access_token_expires_at`, `refresh_token_expires_at`) SELECT `id`, `created_at`, `updated_at`, `email`, `last_logged_in`, `access_token_expires_at`, `refresh_token_expires_at` FROM `users`;
-- Drop "users" table after copying rows
DROP TABLE `users`;
-- Rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
-- atlas:nolint MF101
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
-- atlas:nolint DS103 MF101
-- Password is migrated to system keyring before this migration runs (see migrate_credentials.go)
-- Unique indexes already existed, recreating them is safe
-- Backup tables with FK references to repositories before table recreation
CREATE TEMPORARY TABLE `temp_backup_profile_repositories` AS
SELECT * FROM `backup_profile_repositories`;
CREATE TEMPORARY TABLE `temp_archives` AS
SELECT * FROM `archives`;
CREATE TEMPORARY TABLE `temp_notifications` AS
SELECT * FROM `notifications`;
-- Create "new_repositories" table (without password column)
CREATE TABLE `new_repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `url` text NOT NULL, `has_password` bool NOT NULL DEFAULT false, `last_quick_check_at` datetime NULL, `quick_check_error` json NULL, `last_full_check_at` datetime NULL, `full_check_error` json NULL, `stats_total_chunks` integer NOT NULL DEFAULT 0, `stats_total_size` integer NOT NULL DEFAULT 0, `stats_total_csize` integer NOT NULL DEFAULT 0, `stats_total_unique_chunks` integer NOT NULL DEFAULT 0, `stats_unique_size` integer NOT NULL DEFAULT 0, `stats_unique_csize` integer NOT NULL DEFAULT 0, `cloud_repository_repository` integer NULL, CONSTRAINT `repositories_cloud_repositories_repository` FOREIGN KEY (`cloud_repository_repository`) REFERENCES `cloud_repositories` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL);
-- Copy rows from old table "repositories" to new temporary table "new_repositories"
INSERT INTO `new_repositories` (`id`, `created_at`, `updated_at`, `name`, `url`, `has_password`, `last_quick_check_at`, `quick_check_error`, `last_full_check_at`, `full_check_error`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`, `cloud_repository_repository`) SELECT `id`, `created_at`, `updated_at`, `name`, `url`, `has_password`, `last_quick_check_at`, `quick_check_error`, `last_full_check_at`, `full_check_error`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`, `cloud_repository_repository` FROM `repositories`;
-- Drop "repositories" table after copying rows
DROP TABLE `repositories`;
-- Rename temporary table "new_repositories" to "repositories"
ALTER TABLE `new_repositories` RENAME TO `repositories`;
-- atlas:nolint MF101
-- Create index "repositories_name_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_name_key` ON `repositories` (`name`);
-- atlas:nolint MF101
-- Create index "repositories_url_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
-- atlas:nolint MF101
-- Create index "repositories_cloud_repository_repository_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_cloud_repository_repository_key` ON `repositories` (`cloud_repository_repository`);
-- Restore tables with FK references to repositories
DELETE FROM `backup_profile_repositories`;
INSERT INTO `backup_profile_repositories` SELECT * FROM `temp_backup_profile_repositories`;
DROP TABLE `temp_backup_profile_repositories`;
DELETE FROM `archives`;
INSERT INTO `archives` SELECT * FROM `temp_archives`;
DROP TABLE `temp_archives`;
DELETE FROM `notifications`;
INSERT INTO `notifications` SELECT * FROM `temp_notifications`;
DROP TABLE `temp_notifications`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
