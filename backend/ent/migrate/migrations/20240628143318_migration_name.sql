-- Rename a column from "weekly_day" to "weekday"
ALTER TABLE `backup_schedules` RENAME COLUMN `weekly_day` TO `weekday`;
-- Rename a column from "monthly_day" to "monthday"
ALTER TABLE `backup_schedules` RENAME COLUMN `monthly_day` TO `monthday`;
