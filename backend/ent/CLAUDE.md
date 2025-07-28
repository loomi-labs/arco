# Ent Migrations and Atlas Guide

Best practices for database migrations using Ent ORM and Atlas migration tools.

## Migration Best Practices

### Use Pure Table Recreation for SQLite

**✅ Recommended Pattern:**
```sql
-- Create new table with desired schema
CREATE TABLE `new_table` (...);

-- Transform data during INSERT
INSERT INTO `new_table` (new_field, ...)
SELECT old_field, ... FROM `old_table`;

-- Replace old table
DROP TABLE `old_table`;
ALTER TABLE `new_table` RENAME TO `table`;
```

**Key Principles:**
- Transform data in one place (during INSERT)
- Preserve all existing data
- Use atomic operations with transactions

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