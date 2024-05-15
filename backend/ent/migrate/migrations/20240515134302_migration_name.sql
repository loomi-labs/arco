-- Add column "password" to table: "repositories"
ALTER TABLE `repositories` ADD COLUMN `password` text NOT NULL;
