-- Add column "updated_at" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `updated_at` datetime NOT NULL;
-- Add column "next_run" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `next_run` datetime NULL;
-- Add column "last_run" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `last_run` datetime NULL;
-- Add column "last_run_status" to table: "pruning_rules"
ALTER TABLE `pruning_rules` ADD COLUMN `last_run_status` text NULL;
