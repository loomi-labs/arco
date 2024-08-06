-- Rename a column from "backup_dirs" to "backup_paths"
ALTER TABLE `backup_profiles` RENAME COLUMN `backup_dirs` TO `backup_paths`;
-- Rename a column from "exclude_dirs" to "exclude_paths"
ALTER TABLE `backup_profiles` RENAME COLUMN `exclude_dirs` TO `exclude_paths`;
