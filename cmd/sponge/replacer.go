// Copyright (c) 2024 cions
// Licensed under the MIT License. See LICENSE for details.

package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func randHex(n int) string {
	if n <= 0 || n%2 != 0 {
		panic("randHex: n must be a positive even number")
	}
	buf := make([]byte, n/2)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return hex.EncodeToString(buf)[:n]
}
