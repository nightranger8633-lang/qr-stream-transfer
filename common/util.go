package common

import (
	"encoding/base64"
	"errors"
	"hash/crc32"
)

func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func CheckCRC32(data []byte, expected uint32) error {
	actual := crc32.ChecksumIEEE(data)
	if actual != expected {
		return errors.New("crc32 mismatch")
	}
	return nil
}
