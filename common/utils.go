package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
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

func CheckCRC32(data []byte, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return errors.New("empty crc32")
	}
	want, err := strconv.ParseUint(expected, 16, 32)
	if err != nil {
		return fmt.Errorf("invalid crc32 hex: %w", err)
	}
	if CRC32(data) != uint32(want) {
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
