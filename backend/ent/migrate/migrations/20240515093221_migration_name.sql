-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_profiles" table
CREATE TABLE `new_backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `prefix` text NOT NULL, `directories` json NOT NULL, `has_periodic_backups` bool NOT NULL DEFAULT (false), `periodic_backup_time` datetime NULL, `is_setup_complete` bool NOT NULL DEFAULT (false));
-- Copy rows from old table "backup_profiles" to new temporary table "new_backup_profiles"
INSERT INTO `new_backup_profiles` (`id`, `name`, `prefix`, `directories`, `has_periodic_backups`, `periodic_backup_time`, `is_setup_complete`) SELECT `id`, `name`, `prefix`, `directories`, `has_periodic_backups`, `periodic_backup_time`, `is_setup_complete` FROM `backup_profiles`;
-- Drop "backup_profiles" table after copying rows
DROP TABLE `backup_profiles`;
-- Rename temporary table "new_backup_profiles" to "backup_profiles"
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;