package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func LeonardoImageSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func NormalizeLeonardoImageHash(v string) (string, bool) {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) != sha256.Size*2 {
		return "", false
	}
	if _, e := hex.DecodeString(v); e != nil {
		return "", false
	}
	return v, true
}
