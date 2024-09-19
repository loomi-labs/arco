-- Rename a column from "url" to "location"
ALTER TABLE `repositories` RENAME COLUMN `url` TO `location`;
-- Drop index "repositories_url_key" from table: "repositories"
DROP INDEX `repositories_url_key`;
-- Create index "repositories_location_key" to table: "repositories"
CREATE UNIQUE INDEX `repositories_location_key` ON `repositories` (`location`);
