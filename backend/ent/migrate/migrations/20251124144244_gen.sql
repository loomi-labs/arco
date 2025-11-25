-- Remove duplicate archive_backup_profile foreign key column
-- Consolidate to single backup_profile_archives column as per proper bidirectional edge definition
-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- atlas:nolint DS103
-- Safe: Data from archive_backup_profile is preserved via COALESCE in the INSERT below
-- Create "new_archives" table
CREATE TABLE `new_archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `duration` real NOT NULL, `borg_id` text NOT NULL, `will_be_pruned` bool NOT NULL DEFAULT false, `archive_repository` integer NOT NULL, `backup_profile_archives` integer NULL, CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE);
-- Copy rows from old table "archives" to new temporary table "new_archives"
-- Use COALESCE to prefer archive_backup_profile if it exists, otherwise use backup_profile_archives
INSERT INTO `new_archives` (`id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `backup_profile_archives`)
SELECT `id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, COALESCE(`archive_backup_profile`, `backup_profile_archives`) FROM `archives`;
-- Drop "archives" table after copying rows
DROP TABLE `archives`;
-- Rename temporary table "new_archives" to "archives"
ALTER TABLE `new_archives` RENAME TO `archives`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
