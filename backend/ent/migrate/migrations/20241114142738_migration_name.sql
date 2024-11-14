-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_archives" table
CREATE TABLE `new_archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `created_at` datetime NOT NULL, `duration` integer NOT NULL, `borg_id` text NOT NULL, `will_be_pruned` bool NOT NULL DEFAULT (false), `archive_repository` integer NOT NULL, `archive_backup_profile` integer NULL, `backup_profile_archives` integer NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE, CONSTRAINT `archives_backup_profiles_backup_profile` FOREIGN KEY (`archive_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL, CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL);
-- Copy rows from old table "archives" to new temporary table "new_archives"
INSERT INTO `new_archives` (`id`, `name`, `created_at`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives`) SELECT `id`, `name`, `created_at`, 0 AS `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives` FROM `archives`;
-- Drop "archives" table after copying rows
DROP TABLE `archives`;
-- Rename temporary table "new_archives" to "archives"
ALTER TABLE `new_archives` RENAME TO `archives`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
