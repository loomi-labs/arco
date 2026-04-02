-- Add column "usage_logging_enabled" to table: "settings"
ALTER TABLE `settings` ADD COLUMN `usage_logging_enabled` bool NULL;
-- Add column "installation_id" to table: "settings"
-- atlas:nolint LT101
-- Safe because we immediately set a default UUID for the existing settings row
ALTER TABLE `settings` ADD COLUMN `installation_id` uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
-- Set a random UUID for the existing settings row (will be overridden by app on next startup via Ent default)
-- Create "analytics_events" table
CREATE TABLE `analytics_events` (
  `id` integer NOT NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `event_name` text NOT NULL,
  `event_properties` json NULL,
  `app_version` text NOT NULL,
  `os_info` text NOT NULL,
  `locale` text NOT NULL DEFAULT '',
  `event_time` datetime NOT NULL,
  `sent` bool NOT NULL DEFAULT false
);
-- Create index "analyticsevent_sent" to table: "analytics_events"
CREATE INDEX `analyticsevent_sent` ON `analytics_events` (`sent`);
