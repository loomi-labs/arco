-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_pruning_rules" table
CREATE TABLE `new_pruning_rules` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `keep_hourly` integer NOT NULL, `keep_daily` integer NOT NULL, `keep_weekly` integer NOT NULL, `keep_monthly` integer NOT NULL, `keep_yearly` integer NOT NULL, `keep_within_days` integer NOT NULL, `backup_profile_pruning_rule` integer NOT NULL, CONSTRAINT `pruning_rules_backup_profiles_pruning_rule` FOREIGN KEY (`backup_profile_pruning_rule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE);
-- Copy rows from old table "pruning_rules" to new temporary table "new_pruning_rules"
INSERT INTO `new_pruning_rules` (`id`, `keep_hourly`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly`, `keep_within_days`, `backup_profile_pruning_rule`) SELECT `id`, `keep_hourly`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly`, `keep_within_days`, `backup_profile_pruning_rules` FROM `pruning_rules`;
-- Drop "pruning_rules" table after copying rows
DROP TABLE `pruning_rules`;
-- Rename temporary table "new_pruning_rules" to "pruning_rules"
ALTER TABLE `new_pruning_rules` RENAME TO `pruning_rules`;
-- Create index "pruning_rules_backup_profile_pruning_rule_key" to table: "pruning_rules"
CREATE UNIQUE INDEX `pruning_rules_backup_profile_pruning_rule_key` ON `pruning_rules` (`backup_profile_pruning_rule`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
