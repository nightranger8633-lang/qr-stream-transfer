package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/liyue201/goqr"
	"qrstream/common"
)

type DecoderState struct {
	SessionID string         `json:"session_id"`
	FileName  string         `json:"file_name"`
	Total     int            `json:"total"`
	Chunks    map[int]string `json:"chunks"` // base64 encoded bytes
	UpdatedAt int64          `json:"updated_at"`
}

type TransferState struct {
	SessionID string
	FileName  string
	Total     int
	Chunks    map[int][]byte
	Seen      map[int]bool
	Discarded int
}

func decodePackets(img image.Image) ([]common.Packet, error) {
	symbols, err := goqr.Recognize(img)
	if err != nil {
		return nil, err
	}
	out := make([]common.Packet, 0, len(symbols))
	for _, symbol := range symbols {
		var p common.Packet
		if len(symbol.Payload) == 0 {
			continue
		}
		if err := json.Unmarshal(symbol.Payload, &p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func writeOutput(outputPath string, chunks map[int][]byte, total int) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output failed: %w", err)
	}
	defer f.Close()

	for i := 0; i < total; i++ {
		part, ok := chunks[i]
		if !ok {
			return fmt.Errorf("missing chunk id=%d", i)
		}
		if _, err := f.Write(part); err != nil {
			return fmt.Errorf("write output failed at chunk=%d: %w", i, err)
		}
	}
	return nil
}

func resumePath(saveDir, sessionID string) string {
	return filepath.Join(saveDir, ".resume-"+sessionID+".json")
}

func saveResume(saveDir string, s *TransferState) error {
	ds := DecoderState{
		SessionID: s.SessionID,
		FileName:  s.FileName,
		Total:     s.Total,
		Chunks:    make(map[int]string, len(s.Chunks)),
		UpdatedAt: time.Now().Unix(),
	}
	for id, raw := range s.Chunks {
		ds.Chunks[id] = common.EncodeBase64(raw)
	}
	raw, err := json.Marshal(ds)
	if err != nil {
		return err
	}
	return os.WriteFile(resumePath(saveDir, s.SessionID), raw, 0o644)
}

func loadResume(saveDir, sessionID string) (*TransferState, error) {
	p := resumePath(saveDir, sessionID)
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var ds DecoderState
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}
	ts := &TransferState{
		SessionID: ds.SessionID,
		FileName:  ds.FileName,
		Total:     ds.Total,
		Chunks:    make(map[int][]byte, len(ds.Chunks)),
		Seen:      make(map[int]bool, len(ds.Chunks)),
	}
	for id, b64 := range ds.Chunks {
		part, err := common.DecodeBase64(b64)
		if err != nil {
			continue
		}
		ts.Chunks[id] = part
		ts.Seen[id] = true
	}
	return ts, nil
}

func missingChunkIDs(s *TransferState) []int {
	out := make([]int, 0)
	if s == nil || s.Total <= 0 {
		return out
	}
	for i := 0; i < s.Total; i++ {
		if !s.Seen[i] {
			out = append(out, i)
		}
	}
	return out
}

func sortedMissing(s *TransferState) string {
	miss := missingChunkIDs(s)
	if len(miss) == 0 {
		return "[]"
	}
	slices.Sort(miss)
	if len(miss) > 12 {
		return fmt.Sprintf("%v...(%d)", miss[:12], len(miss))
	}
	return fmt.Sprintf("%v", miss)
}
