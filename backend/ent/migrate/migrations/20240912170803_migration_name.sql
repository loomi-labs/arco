-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_profiles" table
CREATE TABLE `new_backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `prefix` text NOT NULL, `backup_paths` json NOT NULL, `exclude_paths` json NULL, `is_setup_complete` bool NOT NULL DEFAULT (false), `icon` text NOT NULL DEFAULT ('home'));
-- Copy rows from old table "backup_profiles" to new temporary table "new_backup_profiles"
INSERT INTO `new_backup_profiles` (`id`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `is_setup_complete`) SELECT `id`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `is_setup_complete` FROM `backup_profiles`;
-- Drop "backup_profiles" table after copying rows
DROP TABLE `backup_profiles`;
-- Rename temporary table "new_backup_profiles" to "backup_profiles"
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;