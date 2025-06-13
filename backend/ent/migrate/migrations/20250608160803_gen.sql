-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_auth_sessions" table
CREATE TABLE `new_auth_sessions` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `status` text NOT NULL DEFAULT 'PENDING', `expires_at` datetime NOT NULL, PRIMARY KEY (`id`));
-- Copy rows from old table "auth_sessions" to new temporary table "new_auth_sessions"
INSERT INTO `new_auth_sessions` (`id`, `created_at`, `updated_at`, `status`, `expires_at`) SELECT `id`, `created_at`, `updated_at`, `status`, `expires_at` FROM `auth_sessions`;
-- Drop "auth_sessions" table after copying rows
DROP TABLE `auth_sessions`;
-- Rename temporary table "new_auth_sessions" to "auth_sessions"
ALTER TABLE `new_auth_sessions` RENAME TO `auth_sessions`;
-- Drop "refresh_tokens" table
DROP TABLE `refresh_tokens`;
-- Add column "refresh_token" to table: "users"
ALTER TABLE `users` ADD COLUMN `refresh_token` text NULL;
-- Add column "access_token" to table: "users"
ALTER TABLE `users` ADD COLUMN `access_token` text NULL;
-- Add column "access_token_expires_at" to table: "users"
ALTER TABLE `users` ADD COLUMN `access_token_expires_at` datetime NULL;
-- Add column "refresh_token_expires_at" to table: "users"
ALTER TABLE `users` ADD COLUMN `refresh_token_expires_at` datetime NULL;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
