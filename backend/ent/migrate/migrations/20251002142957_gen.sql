-- Normalize repositories table structure (column order and default syntax)
-- This migration also preserves the backup_profile_repositories foreign key relationship

-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;

-- Create "new_repositories" table with normalized structure
CREATE TABLE `new_repositories` (
    `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    `name` text NOT NULL,
    `url` text NOT NULL,
    `password` text NOT NULL,
    `next_integrity_check` datetime NULL,
    `stats_total_chunks` integer NOT NULL DEFAULT 0,
    `stats_total_size` integer NOT NULL DEFAULT 0,
    `stats_total_csize` integer NOT NULL DEFAULT 0,
    `stats_total_unique_chunks` integer NOT NULL DEFAULT 0,
    `stats_unique_size` integer NOT NULL DEFAULT 0,
    `stats_unique_csize` integer NOT NULL DEFAULT 0,
    `cloud_repository_repository` integer NULL,
    CONSTRAINT `repositories_cloud_repositories_repository` FOREIGN KEY (`cloud_repository_repository`) REFERENCES `cloud_repositories` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL
);

-- Copy all data from old table
INSERT INTO `new_repositories` (
    `id`, `created_at`, `updated_at`, `name`, `url`, `password`,
    `next_integrity_check`, `stats_total_chunks`, `stats_total_size`,
    `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`,
    `stats_unique_csize`, `cloud_repository_repository`
)
SELECT
    `id`, `created_at`, `updated_at`, `name`, `url`, `password`,
    `next_integrity_check`, `stats_total_chunks`, `stats_total_size`,
    `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`,
    `stats_unique_csize`, `cloud_repository_repository`
FROM `repositories`;

-- Backup data from tables with foreign keys to repositories
CREATE TEMPORARY TABLE `backup_profile_repositories_backup` AS
SELECT * FROM `backup_profile_repositories`;

CREATE TEMPORARY TABLE `archives_backup` AS
SELECT * FROM `archives`;

CREATE TEMPORARY TABLE `notifications_backup` AS
SELECT * FROM `notifications`;

-- Drop old tables (order matters due to foreign keys)
DROP TABLE `notifications`;
DROP TABLE `archives`;
DROP TABLE `backup_profile_repositories`;
DROP TABLE `repositories`;

-- Rename new repositories table
ALTER TABLE `new_repositories` RENAME TO `repositories`;

-- Recreate junction table with proper foreign keys
CREATE TABLE `backup_profile_repositories` (
    `backup_profile_id` integer NOT NULL,
    `repository_id` integer NOT NULL,
    PRIMARY KEY (`backup_profile_id`, `repository_id`),
    CONSTRAINT `backup_profile_repositories_backup_profile_id` FOREIGN KEY (`backup_profile_id`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE,
    CONSTRAINT `backup_profile_repositories_repository_id` FOREIGN KEY (`repository_id`) REFERENCES `repositories` (`id`) ON DELETE CASCADE
);

-- Recreate archives table
CREATE TABLE `archives` (
    `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    `name` text NOT NULL,
    `duration` real NOT NULL,
    `borg_id` text NOT NULL,
    `will_be_pruned` bool NOT NULL DEFAULT false,
    `archive_repository` integer NOT NULL,
    `archive_backup_profile` integer NULL,
    `backup_profile_archives` integer NULL,
    CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE,
    CONSTRAINT `archives_backup_profiles_backup_profile` FOREIGN KEY (`archive_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL,
    CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL
);

-- Recreate notifications table
CREATE TABLE `notifications` (
    `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
    `created_at` datetime NOT NULL,
    `updated_at` datetime NOT NULL,
    `message` text NOT NULL,
    `type` text NOT NULL,
    `seen` bool NOT NULL DEFAULT false,
    `action` text NULL,
    `notification_backup_profile` integer NOT NULL,
    `notification_repository` integer NOT NULL,
    CONSTRAINT `notifications_backup_profiles_backup_profile` FOREIGN KEY (`notification_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE,
    CONSTRAINT `notifications_repositories_repository` FOREIGN KEY (`notification_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE
);

-- Restore all data
INSERT INTO `backup_profile_repositories` SELECT * FROM `backup_profile_repositories_backup`;
INSERT INTO `archives` (`id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives`)
SELECT `id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`, `backup_profile_archives` FROM `archives_backup`;
INSERT INTO `notifications` (`id`, `created_at`, `updated_at`, `message`, `type`, `seen`, `action`, `notification_backup_profile`, `notification_repository`)
SELECT `id`, `created_at`, `updated_at`, `message`, `type`, `seen`, `action`, `notification_backup_profile`, `notification_repository` FROM `notifications_backup`;

-- Clean up temporary tables
DROP TABLE `backup_profile_repositories_backup`;
DROP TABLE `archives_backup`;
DROP TABLE `notifications_backup`;

-- Recreate indexes on repositories
CREATE UNIQUE INDEX `repositories_name_key` ON `repositories` (`name`);
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
CREATE UNIQUE INDEX `repositories_cloud_repository_repository_key` ON `repositories` (`cloud_repository_repository`);

-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
