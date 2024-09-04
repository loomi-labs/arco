-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_repositories" table
CREATE TABLE `new_repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `name` text NOT NULL, `url` text NOT NULL, `password` text NOT NULL, `stats_total_chunks` integer NOT NULL DEFAULT (0), `stats_total_size` integer NOT NULL DEFAULT (0), `stats_total_csize` integer NOT NULL DEFAULT (0), `stats_total_unique_chunks` integer NOT NULL DEFAULT (0), `stats_unique_size` integer NOT NULL DEFAULT (0), `stats_unique_csize` integer NOT NULL DEFAULT (0));
-- Copy rows from old table "repositories" to new temporary table "new_repositories"
INSERT INTO `new_repositories` (`id`, `name`, `url`, `password`) SELECT `id`, `name`, `url`, `password` FROM `repositories`;
-- Drop "repositories" table after copying rows
DROP TABLE `repositories`;
-- Rename temporary table "new_repositories" to "repositories"
ALTER TABLE `new_repositories` RENAME TO `repositories`;
-- Create index "repositories_url_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
