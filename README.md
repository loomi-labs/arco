

![Icon](icon-small.png)

# Arco Backup

[![CI][s0]][l0] 
[![Go Report Card][s1]][l1] 

[s0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml/badge.svg
[l0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml
[s1]: https://goreportcard.com/badge/github.com/loomi-labs/arco
[l1]: https://goreportcard.com/report/github.com/loomi-labs/arco

![Demo](docs/demo.gif)

## About

Arco is a backup tool that aims to provide a simple and beautiful interface to manage your backups. 

It uses [Borg](https://borgbackup.readthedocs.io/en/stable/index.html) and is compatible with any Borg repository starting from version 1.2.7.

## Motivation

I was looking for an easy-to-use, open-source backup tool that allows me to save all my data encrypted in the cloud.<br>
I found Borg, which is a great tool, but it does not have a graphical interface. I tried some of the available GUIs, but none of them satisfied me fully. So I decided to create my own.

## Prerequisites

Before building or developing Arco, you need to install the following:

1. [Go](https://go.dev/doc/install) - Programming language
2. [Wails v3](https://v3alpha.wails.io/) - Framework for building desktop applications with Go and web technologies
   ```bash
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

## Building

To build a redistributable, run the following command in the project directory:
```bash
task build
```

This will build both the frontend and backend, and package the application for your platform.

## Live Development

To run in live development mode, run:
```bash
task dev
```

This will run a Vite development server that provides fast hot reload of your frontend changes. The backend will also automatically rebuild when you make changes to the Go code.

For frontend-only development, you can run:
```bash
task dev:frontend
```

## Additional Commands

For more development commands, see the [CLAUDE.md](CLAUDE.md) file, which contains a comprehensive list of all available commands for building, testing, and developing Arco.
