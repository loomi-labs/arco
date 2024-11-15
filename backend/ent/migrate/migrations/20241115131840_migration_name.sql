-- Add column "created_at" to table: "backup_profiles"
ALTER TABLE `backup_profiles` ADD COLUMN `created_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "updated_at" to table: "backup_profiles"
ALTER TABLE `backup_profiles` ADD COLUMN `updated_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "created_at" to table: "backup_schedules"
ALTER TABLE `backup_schedules` ADD COLUMN `created_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "updated_at" to table: "notifications"
ALTER TABLE `notifications` ADD COLUMN `updated_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "created_at" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `created_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "created_at" to table: "repositories"
ALTER TABLE `repositories` ADD COLUMN `created_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "updated_at" to table: "repositories"
ALTER TABLE `repositories` ADD COLUMN `updated_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "created_at" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `created_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "updated_at" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `updated_at` datetime DEFAULT '2023-01-01 00:00:00';
-- Add column "updated_at" to table: "archives"
ALTER TABLE `archives` ADD COLUMN `updated_at` datetime DEFAULT '2023-01-01 00:00:00';

-- Update existing rows to set the default value
UPDATE `backup_profiles` SET `created_at` = '2023-01-01 00:00:00' WHERE `created_at` IS NULL;
UPDATE `backup_profiles` SET `updated_at` = '2023-01-01 00:00:00' WHERE `updated_at` IS NULL;
UPDATE `backup_schedules` SET `created_at` = '2023-01-01 00:00:00' WHERE `created_at` IS NULL;
UPDATE `notifications` SET `updated_at` = '2023-01-01 00:00:00' WHERE `updated_at` IS NULL;
UPDATE `pruning_rules` SET `created_at` = '2023-01-01 00:00:00' WHERE `created_at` IS NULL;
UPDATE `repositories` SET `created_at` = '2023-01-01 00:00:00' WHERE `created_at` IS NULL;
UPDATE `repositories` SET `updated_at` = '2023-01-01 00:00:00' WHERE `updated_at` IS NULL;
UPDATE `settings` SET `created_at` = '2023-01-01 00:00:00' WHERE `created_at` IS NULL;
UPDATE `settings` SET `updated_at` = '2023-01-01 00:00:00' WHERE `updated_at` IS NULL;
UPDATE `archives` SET `updated_at` = '2023-01-01 00:00:00' WHERE `updated_at` IS NULL;
