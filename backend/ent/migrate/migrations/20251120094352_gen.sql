-- Add column "expert_mode" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `expert_mode` bool NOT NULL DEFAULT false;

-- Add column "theme" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `theme` text NOT NULL DEFAULT 'system';
