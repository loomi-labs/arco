-- Add column "macfuse_warning_dismissed" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `macfuse_warning_dismissed` bool NOT NULL DEFAULT false;
