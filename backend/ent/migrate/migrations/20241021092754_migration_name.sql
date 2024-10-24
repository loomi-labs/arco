-- Add column "is_enabled" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `is_enabled` bool NOT NULL;
