-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_repositories" table
CREATE TABLE `new_repositories` (`id` integer NOT NULL PRIMARY KEY AUTOINCREMENT, `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, `name` text NOT NULL, `url` text NOT NULL, `password` text NOT NULL, `next_integrity_check` datetime NULL, `stats_total_chunks` integer NOT NULL DEFAULT 0, `stats_total_size` integer NOT NULL DEFAULT 0, `stats_total_csize` integer NOT NULL DEFAULT 0, `stats_total_unique_chunks` integer NOT NULL DEFAULT 0, `stats_unique_size` integer NOT NULL DEFAULT 0, `stats_unique_csize` integer NOT NULL DEFAULT 0, `cloud_repository_repository` integer NULL, CONSTRAINT `repositories_cloud_repositories_repository` FOREIGN KEY (`cloud_repository_repository`) REFERENCES `cloud_repositories` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL);
-- Copy rows from old table "repositories" to new temporary table "new_repositories"
INSERT INTO `new_repositories` (`id`, `created_at`, `updated_at`, `name`, `url`, `password`, `next_integrity_check`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`, `cloud_repository_repository`) SELECT `id`, `created_at`, `updated_at`, `name`, `url`, `password`, `next_integrity_check`, `stats_total_chunks`, `stats_total_size`, `stats_total_csize`, `stats_total_unique_chunks`, `stats_unique_size`, `stats_unique_csize`, `cloud_repository_repository` FROM `repositories`;
-- Drop "repositories" table after copying rows
DROP TABLE `repositories`;
-- Rename temporary table "new_repositories" to "repositories"
ALTER TABLE `new_repositories` RENAME TO `repositories`;
-- Create index "repositories_name_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_name_key` ON `repositories` (`name`);
-- Create index "repositories_url_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_url_key` ON `repositories` (`url`);
-- Create index "repositories_cloud_repository_repository_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_cloud_repository_repository_key` ON `repositories` (`cloud_repository_repository`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
