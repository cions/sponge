// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

//go:build !windows

package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type FileReplacer struct {
	file    *os.File
	name    string
	tmpname string
}

func NewFileReplacer(name string, appendMode bool) (io.WriteCloser, error) {
	if f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666); err == nil {
		return f, err
	}

	var mode fs.FileMode = 0o666
	if fi, err := os.Stat(name); err == nil {
		mode = fi.Mode().Perm()
	}
	dir, _ := filepath.Split(name)
	try := 0
	for {
		tmpname := dir + randHex(8) + ".tmp"
		f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if try++; errors.Is(err, fs.ErrExist) && try < 10000 {
			continue
		} else if err != nil {
			return nil, err
		}

		if appendMode {
			orig, err := os.Open(name)
			if err != nil {
				err2 := f.Close()
				err3 := os.Remove(tmpname)
				return nil, errors.Join(err, err2, err3)
			}
			if _, err := io.Copy(f, orig); err != nil {
				err2 := f.Close()
				err3 := orig.Close()
				err4 := os.Remove(tmpname)
				return nil, errors.Join(err, err2, err3, err4)
			}
			if err := orig.Close(); err != nil {
				err2 := f.Close()
				err3 := os.Remove(tmpname)
				return nil, errors.Join(err, err2, err3)
			}
		}

		return &FileReplacer{f, name, tmpname}, nil
	}
}

func (f *FileReplacer) Write(p []byte) (int, error) {
	return f.file.Write(p)
}

func (f *FileReplacer) Close() error {
	defer os.Remove(f.tmpname)
	if err := f.file.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.tmpname, f.name); err != nil {
		return err
	}
	return nil
}
