-- Disable foreign keys during table recreation
PRAGMA foreign_keys = off;

-- Create new table with compression fields and CHECK constraint
-- atlas:nolint DS103
-- Safe: All data preserved during table recreation, new fields have sensible defaults
CREATE TABLE `new_backup_profiles` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `name` text NOT NULL,
  `prefix` text NOT NULL,
  `backup_paths` json NOT NULL,
  `exclude_paths` json NOT NULL,
  `icon` text NOT NULL,
  `data_section_collapsed` bool NOT NULL DEFAULT false,
  `schedule_section_collapsed` bool NOT NULL DEFAULT false,
  `compression_mode` text NOT NULL DEFAULT 'lz4',
  `compression_level` integer NULL,
  `advanced_section_collapsed` bool NOT NULL DEFAULT true,
  CONSTRAINT `compression_level_valid` CHECK (
    (compression_mode IN ('none', 'lz4') AND compression_level IS NULL) OR
    (compression_mode = 'zstd' AND compression_level >= 1 AND compression_level <= 22) OR
    (compression_mode IN ('zlib', 'lzma') AND compression_level >= 0 AND compression_level <= 9)
  )
);

-- Copy all existing data, new columns will use defaults
INSERT INTO `new_backup_profiles` (
  id, created_at, updated_at, name, prefix, backup_paths, exclude_paths,
  icon, data_section_collapsed, schedule_section_collapsed
)
SELECT
  id, created_at, updated_at, name, prefix, backup_paths, exclude_paths,
  icon, data_section_collapsed, schedule_section_collapsed
FROM `backup_profiles`;

-- Replace old table with new table
DROP TABLE `backup_profiles`;
ALTER TABLE `new_backup_profiles` RENAME TO `backup_profiles`;

-- Recreate indexes and unique constraints
CREATE UNIQUE INDEX `backup_profiles_prefix` ON `backup_profiles` (`prefix`);

-- Re-enable foreign keys
PRAGMA foreign_keys = on;
