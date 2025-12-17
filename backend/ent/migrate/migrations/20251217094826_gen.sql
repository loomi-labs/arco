-- Add has_password column to repositories
-- This runs BEFORE password migration to keyring
ALTER TABLE `repositories` ADD COLUMN `has_password` bool NOT NULL DEFAULT false;

-- Set has_password = true for repos that have a password
UPDATE `repositories` SET `has_password` = true WHERE `password` IS NOT NULL AND `password` != '';
