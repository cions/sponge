// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

//go:build windows

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const (
	PointerSize = 32 << (^uintptr(0) >> 63)
	maxRW       = 1 << 30 // 1 GiB
)

type FileReplacer struct {
	handle  windows.Handle
	name    string
	tmpname string
}

func NewFileReplacer(name string, appendMode bool) (io.WriteCloser, error) {
	if f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666); err == nil {
		return f, err
	}

	orig, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	dir, _ := filepath.Split(name)

	tries := 0
	for {
		tmpname := dir + randHex(8) + ".tmp"
		wtmpname, err := windows.UTF16FromString(tmpname)
		if err != nil {
			panic(fmt.Sprintf("UTF16FromString: %v", err))
		}

		handle, err := windows.CreateFile(
			&wtmpname[0],
			windows.GENERIC_READ|windows.GENERIC_WRITE|windows.DELETE,
			0,
			nil,
			windows.CREATE_NEW,
			windows.FILE_ATTRIBUTE_NORMAL,
			windows.Handle(orig.Fd()),
		)
		if tries++; errors.Is(err, fs.ErrExist) && tries < 10000 {
			continue
		} else if err != nil {
			err = &os.PathError{Op: "CreateFile", Path: tmpname, Err: err}
			err2 := orig.Close()
			return nil, errors.Join(err, err2)
		}
		f := &FileReplacer{handle, name, tmpname}

		if appendMode {
			if _, err := io.Copy(f, orig); err != nil {
				err2 := orig.Close()
				err3 := f.Remove()
				err4 := f.CloseHandle()
				return nil, errors.Join(err, err2, err3, err4)
			}
		}

		if err := orig.Close(); err != nil {
			err2 := f.Remove()
			err3 := f.CloseHandle()
			return nil, errors.Join(err, err2, err3)
		}

		return f, nil
	}
}

func (f *FileReplacer) Write(p []byte) (int, error) {
	var ntotal int
	var n uint32
	for len(p) != 0 {
		b := p
		if len(p) > maxRW {
			b = b[:maxRW]
		}
		err := windows.WriteFile(f.handle, b, &n, nil)
		ntotal += int(n)
		if err != nil {
			return ntotal, &os.SyscallError{Syscall: "WriteFile", Err: err}
		}
		p = p[n:]
	}
	return ntotal, nil
}

func (f *FileReplacer) Rename(newname string) error {
	wname, err := windows.UTF16FromString(newname)
	if err != nil {
		panic(fmt.Sprintf("UTF16FromString: %v", err))
	}

	var (
		fri   []byte
		flags = windows.FILE_RENAME_REPLACE_IF_EXISTS |
			windows.FILE_RENAME_POSIX_SEMANTICS |
			windows.FILE_RENAME_IGNORE_READONLY_ATTRIBUTE
	)
	switch PointerSize {
	case 32:
		fri = binary.LittleEndian.AppendUint32(fri, uint32(flags))
		fri = binary.LittleEndian.AppendUint32(fri, 0)
	case 64:
		fri = binary.LittleEndian.AppendUint64(fri, uint64(flags))
		fri = binary.LittleEndian.AppendUint64(fri, 0)
	default:
		panic(fmt.Sprintf("bad PointerSize: %v", PointerSize))
	}
	fri = binary.LittleEndian.AppendUint32(fri, 2*uint32(len(wname)-1))
	fri, err = binary.Append(fri, binary.LittleEndian, wname)
	if err != nil {
		panic(fmt.Sprintf("encoding/binary.Append: %v", err))
	}

	err = windows.SetFileInformationByHandle(f.handle, windows.FileRenameInfoEx, &fri[0], uint32(len(fri)))
	if err != nil {
		return &os.SyscallError{Syscall: "SetFileInformationByHandle(FileRenameInfoEx)", Err: err}
	}
	return nil
}

func (f *FileReplacer) Remove() error {
	fdi := []byte{1}
	err := windows.SetFileInformationByHandle(f.handle, windows.FileDispositionInfo, &fdi[0], uint32(len(fdi)))
	if err != nil {
		return &os.SyscallError{Syscall: "SetFileInformationByHandle(FileDispositionInfo)", Err: err}
	}
	return nil
}

func (f *FileReplacer) CloseHandle() error {
	err := windows.CloseHandle(f.handle)
	if err != nil {
		return &os.SyscallError{Syscall: "CloseHandle", Err: err}
	}
	return nil
}

func (f *FileReplacer) Close() error {
	if err := f.Rename(f.name); err != nil {
		err2 := f.Remove()
		err3 := f.CloseHandle()
		return errors.Join(err, err2, err3)
	}
	if err := f.CloseHandle(); err != nil {
		return err
	}
	return nil
}

func (f *FileReplacer) Dispose() error {
	if err := f.Remove(); err != nil {
		err2 := f.CloseHandle()
		return errors.Join(err, err2)
	}
	if err := f.CloseHandle(); err != nil {
		return err
	}
	return nil
}
