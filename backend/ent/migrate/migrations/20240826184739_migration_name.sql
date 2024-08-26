-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_failed_backup_runs" table
CREATE TABLE `new_failed_backup_runs` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `error` text NOT NULL, `failed_backup_run_backup_profile` integer NOT NULL, `failed_backup_run_repository` integer NOT NULL, CONSTRAINT `failed_backup_runs_backup_profiles_backup_profile` FOREIGN KEY (`failed_backup_run_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE NO ACTION, CONSTRAINT `failed_backup_runs_repositories_repository` FOREIGN KEY (`failed_backup_run_repository`) REFERENCES `repositories` (`id`) ON DELETE NO ACTION);
-- Copy rows from old table "failed_backup_runs" to new temporary table "new_failed_backup_runs"
INSERT INTO `new_failed_backup_runs` (`id`, `error`, `failed_backup_run_backup_profile`, `failed_backup_run_repository`) SELECT `id`, `error`, `failed_backup_run_backup_profile`, `failed_backup_run_repository` FROM `failed_backup_runs`;
-- Drop "failed_backup_runs" table after copying rows
DROP TABLE `failed_backup_runs`;
-- Rename temporary table "new_failed_backup_runs" to "failed_backup_runs"
ALTER TABLE `new_failed_backup_runs` RENAME TO `failed_backup_runs`;
-- Create index "failedbackuprun_failed_backup_run_backup_profile_failed_backup_run_repository" to table: "failed_backup_runs"
CREATE UNIQUE INDEX `failedbackuprun_failed_backup_run_backup_profile_failed_backup_run_repository` ON `failed_backup_runs` (`failed_backup_run_backup_profile`, `failed_backup_run_repository`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
