package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"qrstream/common"
)

type EncodedTransfer struct {
	SessionID   string
	FileName    string
	FileSize    int64
	ChunkSize   int
	ChunkCount  int
	ChunkFrames []string
}

func BuildTransfer(filePath string, chunkSize int) (*EncodedTransfer, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("invalid chunk size: %d", chunkSize)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	total := (len(content) + chunkSize - 1) / chunkSize
	if total == 0 {
		total = 1
	}

	sessionID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(content))
	payloads := make([]string, 0, total)
	for i := 0; i < total; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		part := content[start:end]
		chunk := common.Chunk{
			ID:        i,
			Total:     total,
			Data:      common.EncodeBase64(part),
			CRC32:     fmt.Sprintf("%08x", common.CRC32(part)),
			Timestamp: common.NowUnixMilli(),
		}

		packet := common.Packet{
			Type:      common.PacketTypeChunk,
			SessionID: sessionID,
			FileName:  filepath.Base(filePath),
			Chunk:     &chunk,
			Meta: &common.PacketMeta{
				TotalChunks: total,
				FileSize:    int64(len(content)),
				Timestamp:   common.NowUnixMilli(),
			},
		}

		raw, err := json.Marshal(packet)
		if err != nil {
			return nil, fmt.Errorf("marshal chunk %d failed: %w", i, err)
		}
		payloads = append(payloads, string(raw))
	}

	return &EncodedTransfer{
		SessionID:   sessionID,
		FileName:    filepath.Base(filePath),
		FileSize:    int64(len(content)),
		ChunkSize:   chunkSize,
		ChunkCount:  total,
		ChunkFrames: payloads,
	}, nil
}
