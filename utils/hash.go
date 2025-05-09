package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
)

func MD5(s string) string {
	return hex.EncodeToString(Md5(s))
}

func Md5(s string) []byte {
	h := md5.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}

func SHA256(s string) string {
	return hex.EncodeToString(Sha256(s))
}

func Sha256(s string) []byte {
	h := sha256.New()
	h.Write([]byte(s))
	return h.Sum(nil)
}
