package common

import (
	"encoding/base32"
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
	// Use unpadded Base32 (A-Z2-7) for better OCR robustness.
	// Compared with base64 variants, this avoids '-', '_', '+' and '/'.
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data)
}

func DecodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty base64 payload")
	}
	if out, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(s)); err == nil {
		return out, nil
	}
	if out, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return out, nil
	}
	// Backward compatibility for old sessions/resume files.
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
