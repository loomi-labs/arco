-- Add column "duration" to table: "archives"
ALTER TABLE `archives` ADD COLUMN `duration` datetime NOT NULL;
-- Add column "borg_id" to table: "archives"
ALTER TABLE `archives` ADD COLUMN `borg_id` text NOT NULL;
