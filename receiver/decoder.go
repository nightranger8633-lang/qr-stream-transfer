package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/liyue201/goqr"
	"qrstream/common"
)

type ReceiverConfig struct {
	OutputPath string
	Interval   time.Duration
	Region     image.Rectangle
}

func RunReceiver(cfg ReceiverConfig) error {
	if cfg.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	if cfg.Interval < 50*time.Millisecond {
		cfg.Interval = 50 * time.Millisecond
	}

	chunks := make(map[int][]byte)
	total := -1
	seenCount := 0
	start := time.Now()
	lastLog := time.Now()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for range ticker.C {
		frame, err := screenshot.CaptureRect(cfg.Region)
		if err != nil {
			if time.Since(lastLog) > 2*time.Second {
				fmt.Printf("capture error: %v\n", err)
				lastLog = time.Now()
			}
			continue
		}

		symbols, err := goqr.Recognize(frame)
		if err != nil || len(symbols) == 0 {
			continue
		}

		for _, symbol := range symbols {
			if len(symbol.Payload) == 0 {
				continue
			}

			var c common.Chunk
			if err := json.Unmarshal(symbol.Payload, &c); err != nil {
				continue
			}

			if c.ID < 0 || c.Total <= 0 || c.ID >= c.Total {
				continue
			}

			if total == -1 {
				total = c.Total
			} else if total != c.Total {
				continue
			}

			if _, exists := chunks[c.ID]; exists {
				continue
			}

			raw, err := common.DecodeBase64(c.Data)
			if err != nil {
				continue
			}
			if err := common.CheckCRC32(raw, c.CRC32); err != nil {
				continue
			}

			chunks[c.ID] = raw
			seenCount++

			fmt.Printf("received chunk: %d/%d (%.2f%%)\n", seenCount, total, float64(seenCount)*100.0/float64(total))
			if seenCount == total {
				if err := writeOutput(cfg.OutputPath, chunks, total); err != nil {
					return err
				}
				fmt.Printf("done in %s, output=%s\n", time.Since(start).Round(time.Millisecond), cfg.OutputPath)
				return nil
			}
		}
	}

	return nil
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
