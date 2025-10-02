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

-- Backup junction table data
CREATE TEMPORARY TABLE `backup_profile_repositories_backup` AS
SELECT * FROM `backup_profile_repositories`;

-- Drop old tables
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

-- Restore junction table data
INSERT INTO `backup_profile_repositories` SELECT * FROM `backup_profile_repositories_backup`;

-- Clean up temporary table
DROP TABLE `backup_profile_repositories_backup`;

-- Recreate indexes on repositories
CREATE UNIQUE INDEX `repositories_name_key` ON `repositories` (`name`);
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
CREATE UNIQUE INDEX `repositories_cloud_repository_repository_key` ON `repositories` (`cloud_repository_repository`);

-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
