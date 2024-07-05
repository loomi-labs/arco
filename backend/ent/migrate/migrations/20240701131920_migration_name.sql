-- Add column "next_run" to table: "backup_schedules"
ALTER TABLE `backup_schedules` ADD COLUMN `next_run` datetime NULL;
-- Add column "last_run" to table: "backup_schedules"
ALTER TABLE `backup_schedules` ADD COLUMN `last_run` datetime NULL;
