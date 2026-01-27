-- Add full disk access warning dismissed column to settings
ALTER TABLE `settings` ADD COLUMN `full_disk_access_warning_dismissed` bool NOT NULL DEFAULT false;
