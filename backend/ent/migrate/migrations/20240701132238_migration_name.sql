-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
