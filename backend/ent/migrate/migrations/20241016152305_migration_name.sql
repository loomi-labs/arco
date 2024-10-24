-- Add column "next_integrity_check" to table: "backup_profiles"
ALTER TABLE `backup_profiles` ADD COLUMN `next_integrity_check` datetime NULL;
-- Create "pruning_rules" table
CREATE TABLE `pruning_rules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `keep_hourly` integer NOT NULL, `keep_daily` integer NOT NULL, `keep_weekly` integer NOT NULL, `keep_monthly` integer NOT NULL, `keep_yearly` integer NOT NULL, `keep_within_days` integer NOT NULL, `backup_profile_pruning_rules` integer NOT NULL, CONSTRAINT `pruning_rules_backup_profiles_pruning_rules` FOREIGN KEY (`backup_profile_pruning_rules`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Create index "pruning_rules_backup_profile_pruning_rules_key" to table: "pruning_rules"
CREATE UNIQUE INDEX `pruning_rules_backup_profile_pruning_rules_key` ON `pruning_rules` (`backup_profile_pruning_rules`);
