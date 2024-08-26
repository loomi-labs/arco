-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_archives" table
CREATE TABLE `new_archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `created_at` datetime NOT NULL, `duration` datetime NOT NULL, `borg_id` text NOT NULL, `archive_repository` integer NOT NULL, `archive_backup_profile` integer NULL, `backup_profile_archives` integer NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE NO ACTION, CONSTRAINT `archives_backup_profiles_backup_profile` FOREIGN KEY (`archive_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL, CONSTRAINT `archives_backup_profiles_archives` FOREIGN KEY (`backup_profile_archives`) REFERENCES `backup_profiles` (`id`) ON DELETE SET NULL);
-- Copy rows from old table "archives" to new temporary table "new_archives"
INSERT INTO `new_archives` (`id`, `name`, `created_at`, `duration`, `borg_id`, `archive_repository`) SELECT `id`, `name`, `created_at`, `duration`, `borg_id`, `archive_repository` FROM `archives`;
-- Drop "archives" table after copying rows
DROP TABLE `archives`;
-- Rename temporary table "new_archives" to "archives"
ALTER TABLE `new_archives` RENAME TO `archives`;
-- Create "failed_backup_runs" table
CREATE TABLE `failed_backup_runs` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `error` text NOT NULL, `failed_backup_run_backup_profile` integer NOT NULL, `failed_backup_run_repository` integer NOT NULL, CONSTRAINT `failed_backup_runs_backup_profiles_backup_profile` FOREIGN KEY (`failed_backup_run_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE NO ACTION, CONSTRAINT `failed_backup_runs_repositories_repository` FOREIGN KEY (`failed_backup_run_repository`) REFERENCES `repositories` (`id`) ON DELETE NO ACTION);
-- Create index "failedbackuprun_failed_backup_run_backup_profile_failed_backup_run_repository" to table: "failed_backup_runs"
CREATE INDEX `failedbackuprun_failed_backup_run_backup_profile_failed_backup_run_repository` ON `failed_backup_runs` (`failed_backup_run_backup_profile`, `failed_backup_run_repository`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
