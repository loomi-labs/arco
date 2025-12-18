-- Add column "disable_transitions" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `disable_transitions` bool NOT NULL DEFAULT false;
-- Add column "disable_shadows" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `disable_shadows` bool NOT NULL DEFAULT false;
