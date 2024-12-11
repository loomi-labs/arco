

<div style="text-align: center;">
    <img src="icon.png" alt="Icon" style="max-width: 200px;">
</div>

# Arco Backup

[![CI][s0]][l0] 
[![Go Report Card][s1]][l1] 

[s0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml/badge.svg
[l0]: https://github.com/loomi-labs/arco/actions/workflows/on_push_go_changes.yml
[s1]: https://goreportcard.com/badge/github.com/loomi-labs/arco
[l1]: https://goreportcard.com/report/github.com/loomi-labs/arco

## About

Arco is a backup tool that aims to provide a simple and beautiful interface to manage your backups. 

It uses [Borg](https://borgbackup.readthedocs.io/en/stable/index.html) and is compatible with any Borg repository starting from version 1.2.7.

## Motivation

I was looking for an easy-to-use, open-source backup tool that allows me to save all my data encrypted in the cloud.<br>
I found Borg, which is a great tool, but it does not have a graphical interface. I tried some of the available GUIs, but none of them satisfied me fully. So I decided to create my own.

## Building

To build a redistributable, install [go](https://go.dev/doc/install), [pnpm](https://pnpm.io/installation), and [Wails](https://wails.io/docs/gettingstarted/installation). Then run `make build` in the project directory.

## Live Development

To run in live development mode, run `make dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes.