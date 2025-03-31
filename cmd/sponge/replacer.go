// Copyright (c) 2024-2025 cions
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

type WriteCloseRemover interface {
	io.WriteCloser
	Remove() error
}

func randHex(n int) string {
	if n <= 0 || n%2 != 0 {
		panic("randHex: n must be a positive even number")
	}
	b := make([]byte, n/2)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(b)[:n]
}
