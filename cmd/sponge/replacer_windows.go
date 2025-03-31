// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

//go:build windows

package main

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const PointerSize = 32 << (^uintptr(0) >> 63)

type FileReplacer struct {
	file    *os.File
	handle  windows.Handle
	name    string
	tmpname string
}

func createFile(name string, templatefile uintptr) (windows.Handle, error) {
	wname, err := windows.UTF16FromString(name)
	if err != nil {
		panic("UTF16FromString: name contains a NUL byte")
	}
	return windows.CreateFile(
		&wname[0],
		windows.GENERIC_READ|windows.GENERIC_WRITE|windows.DELETE,
		0,
		nil,
		windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_NORMAL,
		windows.Handle(templatefile),
	)
}

func NewFileReplacer(name string, appendMode bool) (*FileReplacer, error) {
	if handle, err := createFile(name, 0); err == nil {
		file := os.NewFile(uintptr(handle), name)
		return &FileReplacer{file, handle, name, name}, err
	}

	original, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	dir, _ := filepath.Split(name)
	tries := 0
	for {
		tmpname := dir + randHex(8) + ".tmp"
		handle, err := createFile(tmpname, original.Fd())
		if tries++; err == windows.ERROR_FILE_EXISTS && tries < 10000 {
			continue
		} else if err != nil {
			err = &os.PathError{Op: "CreateFile", Path: tmpname, Err: err}
			err2 := original.Close()
			return nil, errors.Join(err, err2)
		}
		tmpfile := os.NewFile(uintptr(handle), tmpname)
		replacer := &FileReplacer{tmpfile, handle, name, tmpname}

		if appendMode {
			if _, err := io.Copy(tmpfile, original); err != nil {
				err2 := original.Close()
				err3 := replacer.Remove()
				return nil, errors.Join(err, err2, err3)
			}
		}

		if err := original.Close(); err != nil {
			err2 := replacer.Remove()
			return nil, errors.Join(err, err2)
		}

		return replacer, nil
	}
}

func (f *FileReplacer) File() *os.File {
	return f.file
}

func (f *FileReplacer) Read(p []byte) (int, error) {
	return f.file.Read(p)
}

func (f *FileReplacer) Write(p []byte) (int, error) {
	return f.file.Write(p)
}

func (f *FileReplacer) CloseHandle() error {
	return f.file.Close()
}

func (f *FileReplacer) Rename(newname string) error {
	wname, err := windows.UTF16FromString(newname)
	if err != nil {
		panic("UTF16FromString: newname contains a NUL byte")
	}

	flags := windows.FILE_RENAME_REPLACE_IF_EXISTS
	flags |= windows.FILE_RENAME_POSIX_SEMANTICS
	flags |= windows.FILE_RENAME_IGNORE_READONLY_ATTRIBUTE
	fri := binary.LittleEndian.AppendUint32(nil, uint32(flags))

	if PointerSize == 32 {
		fri = binary.LittleEndian.AppendUint32(fri, 0) // RootDirectory
	} else {
		fri = binary.LittleEndian.AppendUint32(fri, 0) // padding
		fri = binary.LittleEndian.AppendUint64(fri, 0) // RootDirectory
	}

	fri = binary.LittleEndian.AppendUint32(fri, 2*uint32(len(wname)-1))
	fri, err = binary.Append(fri, binary.LittleEndian, wname)
	if err != nil {
		panic(err.Error())
	}

	err = windows.SetFileInformationByHandle(
		f.handle, windows.FileRenameInfoEx, &fri[0], uint32(len(fri)))
	if err != nil {
		return &os.SyscallError{
			Syscall: "SetFileInformationByHandle(FileRenameInfoEx)",
			Err:     err,
		}
	}

	return nil
}

func (f *FileReplacer) Dispose() error {
	fdi := []byte{1}
	err := windows.SetFileInformationByHandle(
		f.handle, windows.FileDispositionInfo, &fdi[0], uint32(len(fdi)))
	if err != nil {
		return &os.SyscallError{
			Syscall: "SetFileInformationByHandle(FileDispositionInfo)",
			Err:     err,
		}
	}
	return nil
}

func (f *FileReplacer) Close() error {
	if err := f.Rename(f.name); err != nil {
		err2 := f.Dispose()
		err3 := f.CloseHandle()
		return errors.Join(err, err2, err3)
	}
	if err := f.CloseHandle(); err != nil {
		return err
	}
	return nil
}

func (f *FileReplacer) Remove() error {
	err1 := f.Dispose()
	err2 := f.CloseHandle()
	return errors.Join(err1, err2)
}

var _ WriteCloseRemover = &FileReplacer{}
