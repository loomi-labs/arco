-- Test seed data for migration testing
-- This data represents the state after migration 20250221150025_add_collapse_state.sql
-- Schema at this point has repositories with 'location' field (before rename to 'url')

-- Insert repositories (using 'location' field, not 'url')
INSERT INTO `repositories` (`id`, `created_at`, `updated_at`, `name`, `location`, `password`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`)
VALUES
    (1, '2024-01-01 10:00:00', '2024-01-01 10:00:00', 'Test Repo 1', 'ssh://user@host1.example.com:22/~/backup', 'password123', 100, 1024000, 512000, 80, 819200, 409600),
    (2, '2024-01-02 10:00:00', '2024-01-02 10:00:00', 'Test Repo 2', 'ssh://user@host2.example.com:22/~/backup', 'password456', 200, 2048000, 1024000, 150, 1638400, 819200);

-- Insert backup profiles
INSERT INTO `backup_profiles` (`id`, `created_at`, `updated_at`, `name`, `prefix`, `backup_paths`, `exclude_paths`, `icon`, `data_section_collapsed`, `schedule_section_collapsed`)
VALUES
    (1, '2024-01-01 10:00:00', '2024-01-01 10:00:00', 'Home Backup', 'home', '["\/home\/user\/documents", "\/home\/user\/photos"]', '["*.tmp", "*.cache"]', 'home', false, false),
    (2, '2024-01-02 10:00:00', '2024-01-02 10:00:00', 'Work Backup', 'work', '["\/home\/user\/work"]', '["node_modules"]', 'briefcase', false, false);

-- Insert backup_profile_repositories relationships (this is the critical data to preserve)
INSERT INTO `backup_profile_repositories` (`backup_profile_id`, `repository_id`)
VALUES
    (1, 1),  -- Home Backup -> Test Repo 1
    (1, 2),  -- Home Backup -> Test Repo 2
    (2, 1);  -- Work Backup -> Test Repo 1

-- Insert archives
INSERT INTO `archives` (`id`, `created_at`, `updated_at`, `name`, `duration`, `borg_id`, `will_be_pruned`, `archive_repository`, `archive_backup_profile`)
VALUES
    (1, '2024-01-01 12:00:00', '2024-01-01 12:00:00', 'home-2024-01-01', 120.5, 'abc123def456', false, 1, 1),
    (2, '2024-01-02 12:00:00', '2024-01-02 12:00:00', 'work-2024-01-02', 95.3, 'ghi789jkl012', false, 1, 2);

-- Insert notifications
INSERT INTO `notifications` (`id`, `created_at`, `updated_at`, `message`, `type`, `seen`, `notification_backup_profile`, `notification_repository`)
VALUES
    (1, '2024-01-01 13:00:00', '2024-01-01 13:00:00', 'Backup completed successfully', 'success', false, 1, 1),
    (2, '2024-01-02 13:00:00', '2024-01-02 13:00:00', 'Warning: low disk space', 'warning', false, 2, 1);

-- Insert backup schedules
INSERT INTO `backup_schedules` (`id`, `created_at`, `updated_at`, `mode`, `daily_at`, `weekday`, `weekly_at`, `monthday`, `monthly_at`, `backup_profile_backup_schedule`)
VALUES
    (1, '2024-01-01 10:00:00', '2024-01-01 10:00:00', 'daily', '2024-01-01 02:00:00', 'monday', '2024-01-01 02:00:00', 1, '2024-01-01 02:00:00', 1),
    (2, '2024-01-02 10:00:00', '2024-01-02 10:00:00', 'weekly', '2024-01-02 03:00:00', 'sunday', '2024-01-02 03:00:00', 1, '2024-01-02 03:00:00', 2);

-- Insert pruning rules
INSERT INTO `pruning_rules` (`id`, `created_at`, `updated_at`, `is_enabled`, `keep_hourly`, `keep_daily`, `keep_weekly`, `keep_monthly`, `keep_yearly`, `keep_within_days`, `backup_profile_pruning_rule`)
VALUES
    (1, '2024-01-01 10:00:00', '2024-01-01 10:00:00', true, 24, 7, 4, 6, 2, 30, 1),
    (2, '2024-01-02 10:00:00', '2024-01-02 10:00:00', true, 0, 14, 8, 12, 3, 60, 2);

-- Settings row already exists from 20241202193640_default_settings.sql migration
