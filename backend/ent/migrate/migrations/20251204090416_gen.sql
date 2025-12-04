-- Add column "exclude_caches" to table: "backup_profiles"
ALTER TABLE `backup_profiles` ADD COLUMN `exclude_caches` bool NOT NULL DEFAULT false;
