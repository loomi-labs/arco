-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_settings" table
CREATE TABLE `new_settings` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `expert_mode` bool NOT NULL DEFAULT false,
  `theme` text NOT NULL DEFAULT 'system',
  `disable_transitions` bool NOT NULL DEFAULT false,
  `disable_shadows` bool NOT NULL DEFAULT false,
  `macfuse_warning_dismissed` bool NOT NULL DEFAULT false,
  `full_disk_access_warning_dismissed` bool NOT NULL DEFAULT false,
  `feedback_last_prompted_at` datetime NULL,
  `usage_logging_enabled` bool NULL,
  `installation_id` uuid NOT NULL,
  `font_scale` integer NOT NULL DEFAULT 100,
  `high_contrast` bool NOT NULL DEFAULT false
);
-- Copy rows from old table "settings" to new temporary table "new_settings"
INSERT INTO `new_settings` (`id`, `created_at`, `updated_at`, `expert_mode`, `theme`, `disable_transitions`, `disable_shadows`, `macfuse_warning_dismissed`, `full_disk_access_warning_dismissed`, `feedback_last_prompted_at`, `usage_logging_enabled`, `installation_id`) SELECT `id`, `created_at`, `updated_at`, `expert_mode`, `theme`, `disable_transitions`, `disable_shadows`, `macfuse_warning_dismissed`, `full_disk_access_warning_dismissed`, `feedback_last_prompted_at`, `usage_logging_enabled`, `installation_id` FROM `settings`;
-- Drop "settings" table after copying rows
DROP TABLE `settings`;
-- Rename temporary table "new_settings" to "settings"
ALTER TABLE `new_settings` RENAME TO `settings`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
