// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cions/go-options"
)

var NAME = "sponge"
var VERSION = "(devel)"
var USAGE = `Usage: $NAME [-ar] [FILE]

$NAME reads the standard input and writes it to the specified file.
Unlike shell redirects, $NAME reads all input before writing output.
This allows building a pipeline that reads from and writes to the same file.

If FILE is omitted, $NAME outputs to the standard out.

Options:
  -a, --append          Append to FILE instead of overwriting it
  -r, --replace         Replace FILE atomically instead of overwriting it
  -h, --help            Show this help message and exit
      --version         Show version information and exit
`

type Options struct {
	Append  bool
	Replace bool
	Output  string
}

func (opts *Options) Kind(name string) options.Kind {
	switch name {
	case "-a", "--append":
		return options.Boolean
	case "-r", "--replace":
		return options.Boolean
	case "-h", "--help":
		return options.Boolean
	case "--version":
		return options.Boolean
	default:
		return options.Unknown
	}
}

func (opts *Options) Option(name string, value string, hasValue bool) error {
	switch name {
	case "-a", "--append":
		opts.Append = true
	case "-r", "--replace":
		opts.Replace = true
	case "-h", "--help":
		return options.ErrHelp
	case "--version":
		return options.ErrVersion
	default:
		return options.ErrUnknown
	}
	return nil
}

func (opts *Options) Arg(index int, value string, afterDDash bool) error {
	switch index {
	case 0:
		opts.Output = value
	default:
		return fmt.Errorf("too many arguments")
	}
	return nil
}

func run(args []string) error {
	opts := &Options{
		Output: "-",
	}

	_, err := options.Parse(opts, args)
	if errors.Is(err, options.ErrHelp) {
		usage := strings.ReplaceAll(USAGE, "$NAME", NAME)
		fmt.Print(usage)
		return nil
	} else if errors.Is(err, options.ErrVersion) {
		version := VERSION
		if bi, ok := debug.ReadBuildInfo(); ok {
			version = bi.Main.Version
		}
		fmt.Printf("%v %v\n", NAME, version)
		return nil
	} else if err != nil {
		return err
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var w io.WriteCloser
	if opts.Output == "-" {
		w = os.Stdout
	} else if opts.Replace {
		f, err := NewFileReplacer(opts.Output, opts.Append)
		if err != nil {
			return err
		}
		w = f
	} else {
		flags := os.O_WRONLY | os.O_CREATE
		if opts.Append {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		f, err := os.OpenFile(opts.Output, flags, 0o666)
		if err != nil {
			return err
		}
		w = f
	}

	if _, err := w.Write(data); err != nil {
		err2 := w.Close()
		return fmt.Errorf("write error: %w", errors.Join(err, err2))
	}
	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v: error: %v\n", NAME, err)
		os.Exit(1)
	}
}
