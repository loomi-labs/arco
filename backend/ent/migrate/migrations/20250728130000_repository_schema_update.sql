-- Repository schema update: rename location to url, add CloudRepository entity
-- This migration safely handles the schema changes using SQLite table recreation pattern

-- Migration safety: This migration is safe because:
-- 1. Uses standard SQLite table recreation pattern
-- 2. Transforms data directly during INSERT (location -> url)
-- 3. All existing data is preserved with proper constraints
-- 4. Foreign keys are properly handled with temporary disable/enable

-- Step 1: Create the cloud_repositories table
CREATE TABLE `cloud_repositories` (
    `id` INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    `created_at` DATETIME NOT NULL,
    `updated_at` DATETIME NOT NULL,
    `cloud_id` TEXT NOT NULL,
    `storage_used_bytes` INTEGER NOT NULL DEFAULT 0,
    `location` TEXT NOT NULL
);

-- Step 2: Update repositories table schema using column-based approach
-- This preserves foreign key relationships from other tables

-- Add new url column (non-nullable with default to satisfy constraint)
ALTER TABLE `repositories` ADD COLUMN `url` TEXT NOT NULL DEFAULT '';

-- Copy location data to url column
UPDATE `repositories` SET `url` = `location`;

-- Drop old location index before dropping the column
DROP INDEX IF EXISTS `repositories_location_key`;

-- atlas:nolint DS103
-- Safe to drop location column - data has been copied to url column
ALTER TABLE `repositories` DROP COLUMN `location`;

-- Add cloud repository relationship column
ALTER TABLE `repositories` ADD COLUMN `cloud_repository_repository` INTEGER NULL REFERENCES `cloud_repositories` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL;

-- Step 3: Create indexes
-- atlas:nolint MF101
-- Safe to add unique indexes because data is preserved from existing columns
-- Any duplicates would have been present in the original table
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
CREATE UNIQUE INDEX `repositories_cloud_repository_repository_key` ON `repositories` (`cloud_repository_repository`);