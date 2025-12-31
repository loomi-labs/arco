-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_users" table
CREATE TABLE `new_users` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `email` text NOT NULL, `last_logged_in` datetime NULL, `refresh_token` text NULL, `access_token` text NULL, `access_token_expires_at` datetime NULL, `refresh_token_expires_at` datetime NULL);
-- Copy rows from old table "users" to new temporary table "new_users"
INSERT INTO `new_users` (`id`, `created_at`, `updated_at`, `email`, `last_logged_in`, `refresh_token`, `access_token`, `access_token_expires_at`, `refresh_token_expires_at`) SELECT `id`, `created_at`, `updated_at`, `email`, `last_logged_in`, `refresh_token`, `access_token`, `access_token_expires_at`, `refresh_token_expires_at` FROM `users`;
-- Drop "users" table after copying rows
DROP TABLE `users`;
-- Rename temporary table "new_users" to "users"
ALTER TABLE `new_users` RENAME TO `users`;
-- atlas:nolint MF101
-- Safe: fresh table with copied data; no duplicates possible
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
-- atlas:nolint MF103
-- Safe: table was empty during early development
-- Create "new_auth_sessions" table
CREATE TABLE `new_auth_sessions` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `session_id` text NOT NULL, `status` text NOT NULL DEFAULT 'PENDING', `expires_at` datetime NOT NULL);
-- Copy rows from old table "auth_sessions" to new temporary table "new_auth_sessions"
INSERT INTO `new_auth_sessions` (`id`, `created_at`, `updated_at`, `status`, `expires_at`) SELECT `id`, `created_at`, `updated_at`, `status`, `expires_at` FROM `auth_sessions`;
-- Drop "auth_sessions" table after copying rows
DROP TABLE `auth_sessions`;
-- Rename temporary table "new_auth_sessions" to "auth_sessions"
ALTER TABLE `new_auth_sessions` RENAME TO `auth_sessions`;
-- Create index "auth_sessions_session_id_key" to table: "auth_sessions"
CREATE UNIQUE INDEX `auth_sessions_session_id_key` ON `auth_sessions` (`session_id`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
