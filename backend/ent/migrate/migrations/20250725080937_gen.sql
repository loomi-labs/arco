-- +goose Up
-- Add column "arco_cloud_id" to table: "repositories"
ALTER TABLE `repositories` ADD COLUMN `arco_cloud_id` text;

-- +goose Down
-- Remove column "arco_cloud_id" from table: "repositories"
ALTER TABLE `repositories` DROP COLUMN `arco_cloud_id`;
