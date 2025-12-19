---
name: linear
description: Quick reference for common Linear issue operations using linctl CLI. Use this for issue management, status updates, and project tracking.
---

# Linear Issue Operations

Quick reference for managing Linear issues using the `linctl` CLI.

## Project Context

- **Team**: TECH
- **Assignee**: (your email here)
- **Statuses**: Backlog, Todo, In Progress, Code Review, Done, Canceled, Duplicate

## Common Operations

### Get Issue Details
```bash
linctl issue get TECH-123
```

### List Issues
```bash
# List your issues
linctl issue list --assignee me

# List by status
linctl issue list --state "In Progress"

# Include completed issues
linctl issue list --include-completed

# Recent issues
linctl issue list --newer-than 3_weeks_ago
```

### Search Issues
```bash
linctl issue search "keyword" --team TECH
```

### Create Issue
```bash
linctl issue create --team TECH --title "Issue title" --description "Details" --assign-me --priority 2
```

Priority values: 0=None, 1=Urgent, 2=High, 3=Normal, 4=Low

### Update Issue
```bash
linctl issue update TECH-123 --state "In Progress"
linctl issue update TECH-123 --priority 2
```

### Assign to Self
```bash
linctl issue assign TECH-123
```

## Tips

- Use `--json` flag for machine-readable output
- Use `--plaintext` for non-interactive output
- Run `linctl issue --help` for full command reference
