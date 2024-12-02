-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_settings" table
CREATE TABLE `new_settings` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `show_welcome` bool NOT NULL DEFAULT (true));
-- Copy rows from old table "settings" to new temporary table "new_settings"
INSERT INTO `new_settings` (`id`, `created_at`, `updated_at`, `show_welcome`) SELECT `id`, `created_at`, `updated_at`, `show_welcome` FROM `settings`;
-- Drop "settings" table after copying rows
DROP TABLE `settings`;
-- Rename temporary table "new_settings" to "settings"
ALTER TABLE `new_settings` RENAME TO `settings`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
