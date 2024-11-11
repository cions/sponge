// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

//go:build windows

package main

import (
	"errors"
	"io"
)

func NewFileReplacer(name string, appendMode bool) (io.WriteCloser, error) {
	return nil, errors.New("-r/--replace is not supported on Windows")
}
