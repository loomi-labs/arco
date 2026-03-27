-- Add column "feedback_last_prompted_at" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `feedback_last_prompted_at` datetime NULL;
