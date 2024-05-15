-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_archives" table
CREATE TABLE `new_archives` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `created_at` datetime NOT NULL, `duration` datetime NOT NULL, `borg_id` text NOT NULL, `archive_repository` integer NOT NULL, CONSTRAINT `archives_repositories_repository` FOREIGN KEY (`archive_repository`) REFERENCES `repositories` (`id`) ON DELETE NO ACTION);
-- Copy rows from old table "archives" to new temporary table "new_archives"
INSERT INTO `new_archives` (`id`, `name`, `created_at`, `duration`, `borg_id`, `archive_repository`) SELECT `id`, `name`, `created_at`, `duration`, `borg_id`, `archive_repository` FROM `archives`;
-- Drop "archives" table after copying rows
DROP TABLE `archives`;
-- Rename temporary table "new_archives" to "archives"
ALTER TABLE `new_archives` RENAME TO `archives`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
