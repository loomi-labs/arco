-- Add column "warning_message" to table: "archives"
ALTER TABLE `archives` ADD COLUMN `warning_message` text NULL;

-- Delete existing warning_backup_run notifications (type is now stored on Archive entity)
-- atlas:nolint DS103
DELETE FROM `notifications` WHERE `type` = 'warning_backup_run';
