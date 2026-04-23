package main

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
)

func NumDisplays() int {
	return screenshot.NumActiveDisplays()
}

func BuildCaptureRect(displayIndex int) (image.Rectangle, error) {
	displayCount := screenshot.NumActiveDisplays()
	if displayCount <= 0 {
		return image.Rectangle{}, fmt.Errorf("no active displays detected")
	}

	if displayIndex < 0 || displayIndex >= displayCount {
		return image.Rectangle{}, fmt.Errorf("display index out of range: %d (active=%d)", displayIndex, displayCount)
	}

	return screenshot.GetDisplayBounds(displayIndex), nil
}

func CaptureFrame(rect image.Rectangle) (image.Image, error) {
	return screenshot.CaptureRect(rect)
}

func CaptureFrameByDisplay(displayIndex int) (image.Image, error) {
	rect, err := BuildCaptureRect(displayIndex)
	if err != nil {
		return nil, err
	}
	return screenshot.CaptureRect(rect)
}

func CaptureFramesAllDisplays() ([]image.Image, error) {
	displayCount := screenshot.NumActiveDisplays()
	if displayCount <= 0 {
		return nil, fmt.Errorf("no active displays detected")
	}
	frames := make([]image.Image, 0, displayCount)
	for i := 0; i < displayCount; i++ {
		rect := screenshot.GetDisplayBounds(i)
		frame, err := screenshot.CaptureRect(rect)
		if err != nil {
			continue
		}
		frames = append(frames, frame)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("capture failed on all displays")
	}
	return frames, nil
}
