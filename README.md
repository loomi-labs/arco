

![Icon](docs/icon-small.png)

# Arco Backup

[![CI][s0]][l0] 
[![Go Report Card][s1]][l1] 

[s0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml/badge.svg
[l0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml
[s1]: https://goreportcard.com/badge/github.com/loomi-labs/arco
[l1]: https://goreportcard.com/report/github.com/loomi-labs/arco

![Demo](docs/demo.gif)

## About

Arco is a backup tool that provides a simple and beautiful GUI for managing backups.

It uses [Borg](https://borgbackup.readthedocs.io/en/stable/index.html) and is compatible with any Borg repository starting from version 1.2.7.

Checkout the [website](https://arco-backup.com) for more information.

## Installation

### MacOS
```bash
sh -c "$(curl -sSL 'https://arco-backup.com/macos/install.sh')"
```

### Linux
```bash
sh -c "$(curl -sSL 'https://arco-backup.com/linux/install.sh')"
```

## Features
- Step-by-step process to create a backup profile
- Automatic backups based on schedules
- Backup with encryption, compression, and deduplication
- Backup to local or remote repositories
- Restore backups

## Development

### Prerequisites

Before building or developing Arco, you need to install the following:

1. [Go](https://go.dev/doc/install) - Programming language
2. [Wails v3](https://v3alpha.wails.io/) - Framework for building desktop applications with Go and web technologies
   ```bash
   # You can install Wails v3 system-wide (or you just use go tool wails3)
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   ```
3. [pnpm](https://pnpm.io/installation) - Package manager for the frontend
4. [Task](https://taskfile.dev/installation/) - Task runner used to build and develop Arco
   ```bash
   # macOS
   brew install go-task/tap/go-task
   
   # Linux
   sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
   ```

### Building

To build a redistributable, run the following command in the project directory:
```bash
task build
```

This will build both the frontend and backend, and package the application for your platform.

### Live Development

To run in live development mode, run:
```bash
task dev
```

This will run a Vite development server that provides fast hot reload of your frontend changes. The backend will also automatically rebuild when you make changes to the Go code.

For frontend-only development, you can run:
```bash
task dev:frontend
```

### Additional Commands

For more development commands, see the [CLAUDE.md](CLAUDE.md) file, which contains a comprehensive list of all available commands for building, testing, and developing Arco.
