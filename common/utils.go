package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash/crc32"
	"time"
)

func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func CRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func CheckCRC32(data []byte, expected uint32) error {
	if CRC32(data) != expected {
		return errors.New("crc32 mismatch")
	}
	return nil
}

func MustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func ToJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func FromJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}
