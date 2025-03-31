// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/cions/go-options"
)

var NAME = "sponge"
var VERSION = "(devel)"
var USAGE = `Usage: $NAME [-ar] [[-o] FILE]
       $NAME [-ar] [[-o] FILE] -- COMMAND [ARGS...]

$NAME reads the standard input and writes it to the specified file.
Unlike shell redirects, $NAME reads all input before writing output.
This allows to build a pipeline that reads from and writes to the same file.

If FILE is omitted, $NAME will write to the standard output.

If COMMAND is specified, the command is executed, and only if it terminates
successfully, its output is written to FILE.

Options:
  -a, --append          Append to FILE instead of overwriting it
  -r, --replace         Replace FILE atomically instead of overwriting it
  -o, --output=FILE     Write output to FILE
  -h, --help            Show this help message and exit
      --version         Show version information and exit
`

type Options struct {
	Append  bool
	Replace bool
	Output  string
	Command []string

	outputFlagUsed bool
}

func (opts *Options) Kind(name string) options.Kind {
	switch name {
	case "-a", "--append":
		return options.Boolean
	case "-r", "--replace":
		return options.Boolean
	case "-o", "--output":
		return options.Required
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
	case "-o", "--output":
		opts.Output = value
		opts.outputFlagUsed = true
	case "-h", "--help":
		return options.ErrHelp
	case "--version":
		return options.ErrVersion
	default:
		return options.ErrUnknown
	}
	return nil
}

func (opts *Options) Args(before, after []string) error {
	if opts.outputFlagUsed && len(before) != 0 || len(before) >= 2 {
		return options.Errorf("too many arguments")
	}
	if len(before) == 1 {
		opts.Output = before[0]
	}
	opts.Command = after
	return nil
}

func (opts *Options) readAll(w io.Writer) error {
	if len(opts.Command) == 0 {
		if _, err := io.Copy(w, os.Stdin); err != nil {
			return err
		}
		return nil
	}

	cmd := exec.Command(opts.Command[0], opts.Command[1:]...)
	cmd.Stdin = os.Stdin
	if f, ok := w.(interface{ File() *os.File }); ok {
		cmd.Stdout = f.File()
	} else {
		cmd.Stdout = w
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("%q: %w", opts.Command[0], err)
		}
		return err
	}
	return nil
}

func run(args []string) error {
	opts := &Options{
		Output: "-",
	}

	_, err := options.Parse(opts, args)
	switch {
	case errors.Is(err, options.ErrHelp):
		usage := strings.ReplaceAll(USAGE, "$NAME", NAME)
		fmt.Print(usage)
		return nil
	case errors.Is(err, options.ErrVersion):
		version := VERSION
		if bi, ok := debug.ReadBuildInfo(); ok {
			version = bi.Main.Version
		}
		fmt.Printf("%v %v\n", NAME, version)
		return nil
	case err != nil:
		return err
	}

	if opts.Replace && opts.Output != "-" {
		dst, err := NewFileReplacer(opts.Output, opts.Append)
		if err != nil {
			return err
		}

		if err := opts.readAll(dst); err != nil {
			err2 := dst.Remove()
			return errors.Join(err, err2)
		}

		if err := dst.Close(); err != nil {
			return err
		}

		return nil
	} else {
		buffer := new(bytes.Buffer)

		if err := opts.readAll(buffer); err != nil {
			return err
		}

		var dst io.WriteCloser = os.Stdout
		if opts.Output != "-" {
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
			dst = f
		}

		_, err1 := io.Copy(dst, buffer)
		err2 := dst.Close()
		return errors.Join(err1, err2)
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v: error: %v\n", NAME, err)

		var ee *exec.ExitError
		switch {
		case errors.As(err, &ee):
			os.Exit(ee.ExitCode())
		case errors.Is(err, options.ErrCmdline):
			os.Exit(2)
		default:
			os.Exit(1)
		}
	}
}
