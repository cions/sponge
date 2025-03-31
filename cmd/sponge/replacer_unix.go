// Copyright (c) 2024-2025 cions
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

func NewFileReplacer(name string, appendMode bool) (*FileReplacer, error) {
	if file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666); err == nil {
		return &FileReplacer{file, name, name}, err
	}

	var mode fs.FileMode = 0o666
	if fi, err := os.Stat(name); err == nil {
		mode = fi.Mode().Perm()
	}

	dir, _ := filepath.Split(name)
	tries := 0
	for {
		tmpname := dir + randHex(8) + ".tmp"
		tmpfile, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if tries++; errors.Is(err, fs.ErrExist) && tries < 10000 {
			continue
		} else if err != nil {
			return nil, err
		}
		replacer := &FileReplacer{tmpfile, name, tmpname}

		if appendMode {
			original, err := os.Open(name)
			if err != nil {
				err2 := replacer.Remove()
				return nil, errors.Join(err, err2)
			}
			if _, err := io.Copy(replacer, original); err != nil {
				err2 := original.Close()
				err3 := replacer.Remove()
				return nil, errors.Join(err, err2, err3)
			}
			if err := original.Close(); err != nil {
				err2 := replacer.Remove()
				return nil, errors.Join(err, err2)
			}
		}

		return replacer, nil
	}
}

func (f *FileReplacer) File() *os.File {
	return f.file
}

func (f *FileReplacer) Write(p []byte) (int, error) {
	return f.file.Write(p)
}

func (f *FileReplacer) Close() error {
	if err := f.file.Close(); err != nil {
		err2 := os.Remove(f.tmpname)
		return errors.Join(err, err2)
	}
	if err := os.Rename(f.tmpname, f.name); err != nil {
		err2 := os.Remove(f.tmpname)
		return errors.Join(err, err2)
	}
	return nil
}

func (f *FileReplacer) Remove() error {
	err1 := f.file.Close()
	err2 := os.Remove(f.tmpname)
	return errors.Join(err1, err2)
}

var _ WriteCloseRemover = &FileReplacer{}
