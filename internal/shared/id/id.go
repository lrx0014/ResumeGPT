package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func New(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(fmt.Sprintf("generate identifier: %v", err))
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}
