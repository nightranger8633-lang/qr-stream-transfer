package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/skip2/go-qrcode"
)

type DisplayConfig struct {
	FPS    int
	QRSize int
}

func RunDisplay(payloads []string, cfg DisplayConfig) error {
	if len(payloads) == 0 {
		return fmt.Errorf("no payloads to display")
	}
	if cfg.FPS < 1 {
		cfg.FPS = 5
	}
	if cfg.QRSize < 300 {
		cfg.QRSize = 900
	}

	game := NewSenderGame(payloads, cfg)
	ebiten.SetWindowTitle("QR Sender")
	ebiten.SetFullscreen(true)
	ebiten.SetTPS(cfg.FPS)
	return ebiten.RunGame(game)
}

type SenderGame struct {
	payloads   []string
	cfg        DisplayConfig
	currentIdx int
	shownID    int
	currentQR  *ebiten.Image
	frameCount int
	start      time.Time
	lastError  string
}

func NewSenderGame(payloads []string, cfg DisplayConfig) *SenderGame {
	return &SenderGame{
		payloads: payloads,
		cfg:      cfg,
		start:    time.Now(),
	}
}

func (g *SenderGame) Update() error {
	if len(g.payloads) == 0 {
		return nil
	}

	payload := g.payloads[g.currentIdx]
	qr, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		g.lastError = fmt.Sprintf("qr encode failed: %v", err)
		return nil
	}

	img := qr.Image(g.cfg.QRSize)
	g.currentQR = ebiten.NewImageFromImage(img)
	g.shownID = g.currentIdx
	g.frameCount++
	g.currentIdx++
	if g.currentIdx >= len(g.payloads) {
		g.currentIdx = 0
	}
	return nil
}

func (g *SenderGame) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	if g.currentQR != nil {
		side := int(math.Min(float64(w), float64(h)) * 0.82)
		sw := float64(side) / float64(g.currentQR.Bounds().Dx())
		sh := float64(side) / float64(g.currentQR.Bounds().Dy())

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(sw, sh)
		opts.GeoM.Translate(float64((w-side)/2), float64((h-side)/2))
		screen.DrawImage(g.currentQR, opts)
	}

	elapsed := time.Since(g.start).Seconds()
	realFPS := 0.0
	if elapsed > 0 {
		realFPS = float64(g.frameCount) / elapsed
	}

	status := fmt.Sprintf(
		"chunk: %d/%d | fps(target/real): %d/%.2f",
		g.shownID+1,
		len(g.payloads),
		g.cfg.FPS,
		realFPS,
	)
	ebitenutil.DebugPrintAt(screen, status, 20, 20)
	if g.lastError != "" {
		ebitenutil.DebugPrintAt(screen, g.lastError, 20, 44)
	}
}

func (g *SenderGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
