# Arco Backup

## About

Arco is a backup tool that aims to provide a simple and beautiful interface to manage your backups. 

It uses [Borg](https://borgbackup.readthedocs.io/en/stable/index.html) and is compatible with any Borg repository starting from version 1.2.7.

## Building

To build a redistributable, install [go](https://go.dev/doc/install), [pnpm](https://pnpm.io/installation), and [Wails](https://wails.io/docs/gettingstarted/installation). Then run `make build` in the project directory.

## Live Development

To run in live development mode, run `make dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes.