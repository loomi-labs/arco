-- Rename any duplicate backup profile names before adding unique constraint
-- For duplicates, keep the first occurrence (lowest ID) unchanged,
-- and append " (N)" to subsequent duplicates where N is the count of earlier duplicates
UPDATE backup_profiles SET name = name || ' (' || (
    SELECT COUNT(*) FROM backup_profiles bp2
    WHERE bp2.name = backup_profiles.name AND bp2.id < backup_profiles.id
) || ')'
WHERE id NOT IN (
    SELECT MIN(id) FROM backup_profiles GROUP BY name
);

-- Create index "backup_profiles_name_key" to table: "backup_profiles"
-- atlas:nolint MF101
-- Safe: duplicates are renamed above before creating the unique index
CREATE UNIQUE INDEX `backup_profiles_name_key` ON `backup_profiles` (`name`);
