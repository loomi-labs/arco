# Ent Migrations and Atlas Guide

Best practices for database migrations using Ent ORM and Atlas migration tools.

## Migration Best Practices

### When to Use Table Recreation vs ALTER TABLE

**Use `ALTER TABLE` for simple additions:**
```sql
-- Adding a new column with a default value is fine with ALTER TABLE
ALTER TABLE `settings` ADD COLUMN `new_column` bool NOT NULL DEFAULT false;
```

**Use table recreation for renaming or restructuring:**
SQLite doesn't support `ALTER TABLE RENAME COLUMN`. When you need to rename columns, change column types, or restructure tables, use the table recreation pattern:

```sql
PRAGMA foreign_keys = off;
-- Create new table with desired schema
CREATE TABLE `new_table` (...);

-- Transform/rename data during INSERT
INSERT INTO `new_table` (new_field, ...)
SELECT old_field, ... FROM `old_table`;

-- Replace old table
DROP TABLE `old_table`;
ALTER TABLE `new_table` RENAME TO `table`;
PRAGMA foreign_keys = on;
```

**If other tables have foreign keys pointing to the recreated table**, you must also recreate those tables to restore the FK relationships:

```sql
PRAGMA foreign_keys = off;
-- 1. Recreate the main table
CREATE TABLE `new_parent` (...);
INSERT INTO `new_parent` (...) SELECT ... FROM `parent`;
DROP TABLE `parent`;
ALTER TABLE `new_parent` RENAME TO `parent`;

-- 2. Recreate child tables that reference the parent to restore FKs
CREATE TABLE `new_child` (
  ...,
  CONSTRAINT `child_parent_id` FOREIGN KEY (`parent_id`) REFERENCES `parent` (`id`) ON DELETE CASCADE
);
INSERT INTO `new_child` (...) SELECT ... FROM `child`;
DROP TABLE `child`;
ALTER TABLE `new_child` RENAME TO `child`;
PRAGMA foreign_keys = on;
```

**Key Principles:**
- Use `ALTER TABLE ADD COLUMN` for simple additions with defaults
- Use table recreation for renaming columns, changing types, or dropping columns
- Transform data in one place (during INSERT)
- Always disable FK constraints during table recreation
- Recreate dependent tables to restore foreign key relationships

**Testing table recreation migrations:**
When using table recreation, update the migration tests in `backend/ent/migrate/migrate_test.go`:
1. Add seed data for the affected tables in `testdata/seed_data.sql`
2. Update `captureState()` to capture pre-migration state for affected fields
3. Update `validateMigration()` to verify data was preserved correctly
4. Run `task test` to verify migrations preserve all data and relationships

### Foreign Key Handling

```sql
-- Always disable FK constraints during table recreation
PRAGMA foreign_keys = off;
-- Perform table operations...
PRAGMA foreign_keys = on;
```

## Atlas Linting Guide

Atlas provides strict linting to catch potentially dangerous migration patterns.

### Nolint Directive Rules

1. **Placement**: Must be immediately before the statement it applies to
2. **Multiple Codes**: Can suppress multiple warnings: `-- atlas:nolint DS103 LT101 MF101`
3. **Documentation**: Always explain WHY the suppression is safe

**Example:**
```sql
-- atlas:nolint DS103 LT101
-- Safe because data is copied from 'location' to 'url' during INSERT
CREATE TABLE `new_repositories` (
    id INTEGER PRIMARY KEY,
    url TEXT NOT NULL  -- Renamed from 'location'
);
```

### Common Warning Codes

- **DS103**: Destructive changes (dropping columns)
- **LT101**: Non-nullable column without default value
- **MF101**: Unique constraints on potentially duplicate data

## Workflow Commands

```bash
# Update schema files, then:
task db:ent:generate    # Generate Ent models
task db:migrate:new     # Create migration
task db:ent:hash        # Hash migration
task db:ent:lint        # Lint migration
task db:migrate         # Apply migration

# Entity creation:
task db:ent:new -- EntityName
```

## Key Takeaways

1. **Simplicity**: Pure table recreation is cleaner than mixed patterns
2. **Document Safety**: Always explain why nolint suppressions are safe
3. **Single Transformation**: Transform data once, in the right place
4. **Test Thoroughly**: Lint, build, and verify after each change