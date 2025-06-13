-- Create "auth_sessions" table
CREATE TABLE `auth_sessions` (`id` text NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `user_email` text NOT NULL, `status` text NOT NULL DEFAULT 'PENDING', `expires_at` datetime NOT NULL, PRIMARY KEY (`id`));
-- Create "refresh_tokens" table
CREATE TABLE `refresh_tokens` (`id` uuid NOT NULL, `token_hash` text NOT NULL, `expires_at` datetime NOT NULL, `created_at` datetime NOT NULL, `last_used_at` datetime NULL, `user_id` uuid NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `refresh_tokens_users_refresh_tokens` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create "users" table
CREATE TABLE `users` (`id` uuid NOT NULL, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `email` text NOT NULL, `last_logged_in` datetime NULL, PRIMARY KEY (`id`));
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX `users_email_key` ON `users` (`email`);
