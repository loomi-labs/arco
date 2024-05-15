-- Create "backup_profiles" table
CREATE TABLE `backup_profiles` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `prefix` text NOT NULL, `directories` json NOT NULL, `directory_suggestions` json NOT NULL, `has_periodic_backups` bool NOT NULL DEFAULT (false), `periodic_backup_time` datetime NULL, `is_setup_complete` bool NOT NULL DEFAULT (false));
-- Create "repositories" table
CREATE TABLE `repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `url` text NOT NULL);
-- Create "backup_profile_repositories" table
CREATE TABLE `backup_profile_repositories` (`backup_profile_id` integer NOT NULL, `repository_id` integer NOT NULL, PRIMARY KEY (`backup_profile_id`, `repository_id`), CONSTRAINT `backup_profile_repositories_backup_profile_id` FOREIGN KEY (`backup_profile_id`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `backup_profile_repositories_repository_id` FOREIGN KEY (`repository_id`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
