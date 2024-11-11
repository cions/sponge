# sponge

[![GitHub Releases](https://img.shields.io/github/v/release/cions/sponge?sort=semver)](https://github.com/cions/sponge/releases)
[![LICENSE](https://img.shields.io/github/license/cions/sponge)](https://github.com/cions/sponge/blob/master/LICENSE)
[![CI](https://github.com/cions/sponge/actions/workflows/ci.yml/badge.svg)](https://github.com/cions/sponge/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/cions/sponge.svg)](https://pkg.go.dev/github.com/cions/sponge)
[![Go Report Card](https://goreportcard.com/badge/github.com/cions/sponge)](https://goreportcard.com/report/github.com/cions/sponge)

Go implementation of [moreutils](https://joeyh.name/code/moreutils/)'s sponge(1) command.

## Usage

```
$ sponge --help
Usage: sponge [-ar] [FILE]

sponge reads the standard input and writes it to the specified file.
Unlike shell redirects, sponge reads all input before writing output.
This allows building a pipeline that reads from and writes to the same file.

If FILE is omitted, sponge outputs to the standard out.

Options:
  -a, --append          Append to FILE instead of overwriting it
  -r, --replace         Replace FILE atomically instead of overwriting it
  -h, --help            Show this help message and exit
      --version         Show version information and exit
```

## Installation

[Download from GitHub Releases](https://github.com/cions/sponge/releases)

### Build from source

```sh
$ go install github.com/cions/sponge/cmd/sponge@latest
```

## License

MIT
