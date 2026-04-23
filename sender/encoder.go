package main

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"os"

	"qrstream/common"
)

func BuildChunkPayloads(filePath string, chunkSize int) ([]string, int, error) {
	if chunkSize <= 0 {
		return nil, 0, fmt.Errorf("invalid chunk size: %d", chunkSize)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("read file failed: %w", err)
	}

	total := (len(content) + chunkSize - 1) / chunkSize
	if total == 0 {
		total = 1
	}

	payloads := make([]string, 0, total)
	for i := 0; i < total; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		part := content[start:end]
		chunk := common.Chunk{
			ID:    i,
			Total: total,
			Data:  common.EncodeBase64(part),
			CRC32: crc32.ChecksumIEEE(part),
		}

		raw, err := json.Marshal(chunk)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal chunk %d failed: %w", i, err)
		}
		payloads = append(payloads, string(raw))
	}

	return payloads, len(content), nil
}
