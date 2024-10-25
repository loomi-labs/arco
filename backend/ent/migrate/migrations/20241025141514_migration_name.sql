-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_backup_schedules" table
CREATE TABLE `new_backup_schedules`
(
    `id`                             integer  NOT NULL PRIMARY KEY AUTOINCREMENT,
    `updated_at`                     datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `mode`                           text     NOT NULL DEFAULT ('disabled'),
    `hourly`                         bool     NOT NULL DEFAULT (false),
    `daily_at`                       datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `weekday`                        text     NOT NULL DEFAULT ('Monday'),
    `weekly_at`                      datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `monthday`                       integer  NOT NULL DEFAULT 1,
    `monthly_at`                     datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `next_run`                       datetime NULL,
    `last_run`                       datetime NULL,
    `last_run_status`                text     NULL,
    `backup_profile_backup_schedule` integer  NOT NULL DEFAULT 0,
    CONSTRAINT `backup_schedules_backup_profiles_backup_schedule` FOREIGN KEY (`backup_profile_backup_schedule`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE
);
-- Copy rows from old table "backup_schedules" to new temporary table "new_backup_schedules"
INSERT INTO `new_backup_schedules` (`id`, `hourly`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`,
                                    `next_run`, `last_run`, `last_run_status`, `backup_profile_backup_schedule`, `updated_at`)
SELECT
    `id`,
    COALESCE(`hourly`, false),
    COALESCE(`daily_at`, CURRENT_TIMESTAMP),
    COALESCE(`weekday`, 'Monday'),
    COALESCE(`weekly_at`, CURRENT_TIMESTAMP),
    COALESCE(`monthday`, 1),
    COALESCE(`monthly_at`, CURRENT_TIMESTAMP),
    `next_run`,
    `last_run`,
    `last_run_status`,
    COALESCE(`backup_profile_backup_schedule`, 0),
    CURRENT_TIMESTAMP
FROM `backup_schedules`;
-- Drop "backup_schedules" table after copying rows
DROP TABLE `backup_schedules`;
-- Rename temporary table "new_backup_schedules" to "backup_schedules"
ALTER TABLE `new_backup_schedules`
    RENAME TO `backup_schedules`;
-- Create index "backup_schedules_backup_profile_backup_schedule_key" to table: "backup_schedules"
CREATE UNIQUE INDEX `backup_schedules_backup_profile_backup_schedule_key` ON `backup_schedules` (`backup_profile_backup_schedule`);
-- Create index "backupschedule_next_run" to table: "backup_schedules"
CREATE INDEX `backupschedule_next_run` ON `backup_schedules` (`next_run`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;