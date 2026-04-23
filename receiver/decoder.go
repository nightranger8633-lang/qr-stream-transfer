package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"qrstream/common"

	"github.com/liyue201/goqr"
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

const duplicatePayloadWindow = 300 * time.Millisecond

var (
	decodeMu            sync.Mutex
	recentPayloadSeenAt = map[string]time.Time{}
)

func decodePackets(img image.Image) ([]common.Packet, error) {
	fmt.Println("---- decode frame start ----")

	symbols, err := goqr.Recognize(img)
	if err != nil {
		fmt.Println("[QR ERROR]", err)
		return nil, err
	}

	fmt.Println("[QR SYMBOLS FOUND]", len(symbols))

	out := make([]common.Packet, 0, len(symbols))

	for i, symbol := range symbols {

		fmt.Println("---- SYMBOL", i, "----")
		fmt.Println("[RAW PAYLOAD LENGTH]", len(symbol.Payload))
		fmt.Println("[RAW PAYLOAD STRING]")
		fmt.Println(string(symbol.Payload))

		if len(symbol.Payload) == 0 {
			fmt.Println("[SKIP] empty payload")
			continue
		}

		if shouldSkipDuplicatePayload(symbol.Payload) {
			fmt.Println("[SKIP] duplicate payload in debounce window")
			continue
		}

		var p common.Packet

		err := json.Unmarshal(symbol.Payload, &p)
		if err != nil {
			fmt.Println("[JSON ERROR]", err)
			fmt.Println("[BAD PAYLOAD]")
			fmt.Println(string(symbol.Payload))
			continue
		}

		fmt.Println("[JSON OK] packet parsed")

		// ===== packet 基础信息 =====
		fmt.Println("[TYPE]", p.Type)
		fmt.Println("[SESSION]", p.SessionID)
		fmt.Println("[FILE]", p.FileName)

		// ===== chunk debug =====
		if p.Chunk != nil {
			fmt.Println("[CHUNK ID]", p.Chunk.ID)
			fmt.Println("[CHUNK TOTAL]", p.Chunk.Total)
			fmt.Println("[CHUNK DATA LEN]", len(p.Chunk.Data))
			fmt.Println("[CHUNK CRC32]", p.Chunk.CRC32)
		} else {
			fmt.Println("[WARNING] chunk is nil")
		}

		out = append(out, p)
	}

	fmt.Println("---- decode frame end ----")
	return out, nil
}

func shouldSkipDuplicatePayload(payload []byte) bool {
	key := string(payload)
	now := time.Now()

	decodeMu.Lock()
	defer decodeMu.Unlock()

	if ts, ok := recentPayloadSeenAt[key]; ok && now.Sub(ts) < duplicatePayloadWindow {
		return true
	}
	recentPayloadSeenAt[key] = now

	// periodic cleanup to bound map size
	if len(recentPayloadSeenAt) > 2048 {
		cutoff := now.Add(-3 * time.Second)
		for k, v := range recentPayloadSeenAt {
			if v.Before(cutoff) {
				delete(recentPayloadSeenAt, k)
			}
		}
	}
	return false
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
