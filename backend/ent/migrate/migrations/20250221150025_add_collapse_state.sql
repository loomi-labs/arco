-- Add collapse state columns to backup_profiles
ALTER TABLE `backup_profiles` ADD COLUMN `data_section_collapsed` bool NOT NULL DEFAULT false;
ALTER TABLE `backup_profiles` ADD COLUMN `schedule_section_collapsed` bool NOT NULL DEFAULT false;
