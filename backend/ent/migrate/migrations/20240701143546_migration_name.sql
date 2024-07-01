-- Add column "last_run_status" to table: "backup_schedules"
ALTER TABLE `backup_schedules` ADD COLUMN `last_run_status` text NULL;
