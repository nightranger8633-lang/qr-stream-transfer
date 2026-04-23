package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	filePath := flag.String("file", "", "path to input file")
	chunkSize := flag.Int("chunk-size", 1000, "chunk size in bytes")
	fps := flag.Int("fps", 10, "QR refresh rate (5-15 recommended)")
	qrSize := flag.Int("qr-size", 1000, "QR image size in pixels")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("missing required flag: -file")
	}

	payloads, fileSize, err := BuildChunkPayloads(*filePath, *chunkSize)
	if err != nil {
		log.Fatalf("build payloads failed: %v", err)
	}

	fmt.Printf("sender ready: file=%s size=%d bytes chunks=%d chunk-size=%d fps=%d\n",
		*filePath, fileSize, len(payloads), *chunkSize, *fps)

	err = RunDisplay(payloads, DisplayConfig{
		FPS:    *fps,
		QRSize: *qrSize,
	})
	if err != nil {
		log.Fatalf("display failed: %v", err)
	}
}
