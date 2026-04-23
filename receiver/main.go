package main

import (
	"flag"
	"fmt"
	"log"
	"time"
)

func main() {
	outputPath := flag.String("out", "received.bin", "output file path")
	intervalMS := flag.Int("interval-ms", 80, "screen sampling interval in milliseconds (50-100 recommended)")
	displayIndex := flag.Int("display", 0, "display index for screen capture")
	x := flag.Int("x", 0, "capture region x")
	y := flag.Int("y", 0, "capture region y")
	w := flag.Int("w", 0, "capture region width (0 means full display)")
	h := flag.Int("h", 0, "capture region height (0 means full display)")
	flag.Parse()

	useRegion := *w > 0 && *h > 0

	rect, err := BuildCaptureRect(CaptureConfig{
		DisplayIndex: *displayIndex,
		UseRegion:    useRegion,
		X:            *x,
		Y:            *y,
		Width:        *w,
		Height:       *h,
	})
	if err != nil {
		log.Fatalf("build capture region failed: %v", err)
	}

	fmt.Printf("receiver started: out=%s interval=%dms region=%v\n", *outputPath, *intervalMS, rect)
	err = RunReceiver(ReceiverConfig{
		OutputPath: *outputPath,
		Interval:   time.Duration(*intervalMS) * time.Millisecond,
		Region:     rect,
	})
	if err != nil {
		log.Fatalf("receiver failed: %v", err)
	}
}
