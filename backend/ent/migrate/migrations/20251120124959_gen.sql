-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Backup join table and notification data
CREATE TEMPORARY TABLE `temp_backup_profile_repositories` AS
SELECT * FROM `backup_profile_repositories`;

CREATE TEMPORARY TABLE `temp_notifications` AS
SELECT * FROM `notifications`;
-- Create "new_backup_profiles" table
CREATE TABLE `new_backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `prefix` text NOT NULL, `backup_paths` json NOT NULL, `exclude_paths` json NULL, `icon` text NOT NULL, `compression_mode` text NOT NULL DEFAULT 'lz4', `compression_level` integer NULL, `data_section_collapsed` bool NOT NULL DEFAULT false, `schedule_section_collapsed` bool NOT NULL DEFAULT false, `advanced_section_collapsed` bool NOT NULL DEFAULT true, CONSTRAINT `compression_level_valid` CHECK (
					(compression_mode IN ('none', 'lz4') AND compression_level IS NULL) OR
					(compression_mode = 'zstd' AND compression_level >= 1 AND compression_level <= 22) OR
					(compression_mode = 'zlib' AND compression_level >= 0 AND compression_level <= 9) OR
					(compression_mode = 'lzma' AND compression_level >= 0 AND compression_level <= 6)
				));
-- Copy rows from old table "backup_profiles" to new temporary table "new_backup_profiles"
INSERT INTO `new_backup_profiles` (`id`, `created_at`, `updated_at`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `icon`, `data_section_collapsed`, `schedule_section_collapsed`) SELECT `id`, `created_at`, `updated_at`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `icon`, `data_section_collapsed`, `schedule_section_collapsed` FROM `backup_profiles`;
-- Drop "backup_profiles" table after copying rows
DROP TABLE `backup_profiles`;
-- Rename temporary table "new_backup_profiles" to "backup_profiles"
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;
-- Create index "backup_profiles_prefix_key" to table: "backup_profiles"
CREATE UNIQUE INDEX `backup_profiles_prefix_key` ON `backup_profiles` (`prefix`);
-- Restore join table and notification data
DELETE FROM `backup_profile_repositories`;
INSERT INTO `backup_profile_repositories`
SELECT * FROM `temp_backup_profile_repositories`;

DELETE FROM `notifications`;
INSERT INTO `notifications`
SELECT * FROM `temp_notifications`;

-- Clean up temporary tables
DROP TABLE `temp_backup_profile_repositories`;
DROP TABLE `temp_notifications`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
