-- Add column "interval_minutes" to table: "backup_schedules"
ALTER TABLE `backup_schedules` ADD COLUMN `interval_minutes` integer NOT NULL DEFAULT 60;

-- Update mode enum: 'hourly' -> 'minute_interval'
-- Existing hourly schedules become minute_interval with 60 minutes (default)
UPDATE `backup_schedules` SET `mode` = 'minute_interval' WHERE `mode` = 'hourly';
