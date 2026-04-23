package main

import (
	"fmt"
	"image"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	defaultChunkSize = 120
	defaultFPS       = 3
	defaultQRSize    = 1100
	perChunkRepeat   = 12
)

type SenderUI struct {
	app fyne.App
	win fyne.Window

	filePath string
	transfer *EncodedTransfer

	fileLabel     *widget.Label
	progressLabel *widget.Label
	fpsLabel      *widget.Label
	statusLabel   *widget.Label
	qrImage       *canvas.Image

	startBtn *widget.Button
	stopBtn  *widget.Button

	mu           sync.Mutex
	running      bool
	stopCh       chan struct{}
	currentChunk int
	totalFrames  int
	startedAt    time.Time
	lastQR       image.Image
}

func RunSenderGUI() {
	ui := &SenderUI{app: app.NewWithID("qrstream.sender")}
	ui.win = ui.app.NewWindow("QR Sender")
	ui.win.Resize(fyne.NewSize(980, 840))
	ui.build()
	ui.win.ShowAndRun()
}

func (s *SenderUI) build() {
	s.fileLabel = widget.NewLabel("File: (未选择)")
	s.progressLabel = widget.NewLabel("Progress: 0/0")
	s.fpsLabel = widget.NewLabel("FPS: 0.00")
	s.statusLabel = widget.NewLabel("Status: Idle")
	s.qrImage = canvas.NewImageFromImage(nil)
	s.qrImage.FillMode = canvas.ImageFillContain
	s.qrImage.SetMinSize(fyne.NewSize(640, 520))

	selectBtn := widget.NewButton("选择文件", func() {
		dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, s.win)
				return
			}
			if r == nil {
				return
			}
			s.filePath = r.URI().Path()
			_ = r.Close()
			s.fileLabel.SetText("File: " + s.filePath)
		}, s.win).Show()
	})

	s.startBtn = widget.NewButton("开始发送", func() {
		s.start()
	})
	s.stopBtn = widget.NewButton("停止", func() {
		s.stop()
	})
	s.stopBtn.Disable()

	top := container.NewVBox(
		container.NewHBox(selectBtn, s.startBtn, s.stopBtn),
		s.fileLabel,
		s.progressLabel,
		s.fpsLabel,
		s.statusLabel,
	)

	s.win.SetContent(container.NewBorder(top, nil, nil, nil, container.NewPadded(container.NewMax(s.qrImage))))
}

func (s *SenderUI) start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	if s.filePath == "" {
		dialog.ShowInformation("提示", "请先选择要发送的文件", s.win)
		return
	}

	transfer, err := BuildTransfer(s.filePath, defaultChunkSize)
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.transfer = transfer
	s.running = true
	s.stopCh = make(chan struct{})
	s.currentChunk = 0
	s.totalFrames = 0
	s.startedAt = time.Now()
	s.statusLabel.SetText("Status: Sending (loop)")
	s.progressLabel.SetText(fmt.Sprintf("Progress: 0/%d", s.transfer.ChunkCount))
	s.startBtn.Disable()
	s.stopBtn.Enable()

	go s.runLoop(s.stopCh)
}

func (s *SenderUI) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
	s.statusLabel.SetText("Status: Stopped")
	s.startBtn.Enable()
	s.stopBtn.Disable()
}

func (s *SenderUI) runLoop(stopCh chan struct{}) {
	ticker := time.NewTicker(time.Second / defaultFPS)
	defer ticker.Stop()

	idx := 0
	repeat := 0
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			payload := s.transfer.ChunkFrames[idx]
			s.renderPayload(payload, idx, s.transfer.ChunkCount)

			repeat++
			if repeat >= perChunkRepeat {
				repeat = 0
				idx++
				if idx >= s.transfer.ChunkCount {
					idx = 0
				}
			}
		}
	}
}

func (s *SenderUI) renderPayload(payload string, idx, total int) {
	img, err := QRImage(payload, defaultQRSize)
	if err != nil {
		return
	}
	s.lastQR = img
	s.currentChunk = idx + 1
	s.totalFrames++
	elapsed := time.Since(s.startedAt).Seconds()
	realFPS := 0.0
	if elapsed > 0 {
		realFPS = float64(s.totalFrames) / elapsed
	}

	s.qrImage.Image = s.lastQR
	s.qrImage.Refresh()
	s.progressLabel.SetText(fmt.Sprintf("Progress: %d/%d", s.currentChunk, total))
	s.fpsLabel.SetText(fmt.Sprintf("FPS: %.2f", realFPS))
}

func (s *SenderUI) finish(status string) {
	s.statusLabel.SetText(status)
	s.startBtn.Enable()
	s.stopBtn.Disable()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}
