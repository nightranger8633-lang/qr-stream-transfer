package main

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

type CaptureConfig struct {
	DisplayIndex int
	UseRegion    bool
	X            int
	Y            int
	Width        int
	Height       int
}

func BuildCaptureRect(cfg CaptureConfig) (image.Rectangle, error) {
	displayCount := screenshot.NumActiveDisplays()
	if displayCount <= 0 {
		return image.Rectangle{}, fmt.Errorf("no active displays detected")
	}

	if cfg.DisplayIndex < 0 || cfg.DisplayIndex >= displayCount {
		return image.Rectangle{}, fmt.Errorf("display index out of range: %d (active=%d)", cfg.DisplayIndex, displayCount)
	}

	bounds := screenshot.GetDisplayBounds(cfg.DisplayIndex)
	if !cfg.UseRegion {
		return bounds, nil
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return image.Rectangle{}, fmt.Errorf("invalid region size: %dx%d", cfg.Width, cfg.Height)
	}

	r := image.Rect(cfg.X, cfg.Y, cfg.X+cfg.Width, cfg.Y+cfg.Height)
	if !r.In(bounds) {
		return image.Rectangle{}, fmt.Errorf("region %v is outside display bounds %v", r, bounds)
	}
	return r, nil
}
