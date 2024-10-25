-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Drop "failed_backup_runs" table
DROP TABLE `failed_backup_runs`;
-- Create "notifications" table
CREATE TABLE `notifications` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `message` text NOT NULL, `type` text NOT NULL, `seen` bool NOT NULL DEFAULT (false), `action` text NULL, `notification_backup_profile` integer NOT NULL, `notification_repository` integer NOT NULL, CONSTRAINT `notifications_backup_profiles_backup_profile` FOREIGN KEY (`notification_backup_profile`) REFERENCES `backup_profiles` (`id`) ON DELETE CASCADE, CONSTRAINT `notifications_repositories_repository` FOREIGN KEY (`notification_repository`) REFERENCES `repositories` (`id`) ON DELETE CASCADE);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
