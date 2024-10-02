-- Rename a column from "url" to "location"
ALTER TABLE `repositories` RENAME COLUMN `url` TO `location`;
-- Drop index "repositories_url_key" from table: "repositories"
DROP INDEX `repositories_url_key`;
