package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"qrstream/common"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	captureInterval = 80 * time.Millisecond
)

type ReceiverUI struct {
	app fyne.App
	win fyne.Window

	saveDir string
	resume  bool

	pathLabel     *widget.Label
	progressLabel *widget.Label
	missingLabel  *widget.Label
	statusLabel   *widget.Label
	logBox        *widget.Entry
	progressBar   *widget.ProgressBar
	startBtn      *widget.Button
	stopBtn       *widget.Button

	mu      sync.Mutex
	running bool
	stopCh  chan struct{}

	state *TransferState
}

func RunReceiverGUI() {
	ui := &ReceiverUI{app: app.NewWithID("qrstream.receiver")}
	ui.win = ui.app.NewWindow("QR Receiver")
	ui.win.Resize(fyne.NewSize(980, 800))
	ui.build()
	ui.win.ShowAndRun()
}

func (r *ReceiverUI) build() {
	r.pathLabel = widget.NewLabel("保存目录: (未选择)")
	r.progressLabel = widget.NewLabel("Received: 0/0")
	r.missingLabel = widget.NewLabel("Missing chunks: 0")
	r.statusLabel = widget.NewLabel("Status: Idle")
	r.progressBar = widget.NewProgressBar()
	r.logBox = widget.NewMultiLineEntry()
	r.logBox.Wrapping = fyne.TextWrapWord
	r.logBox.Disable()

	selectDir := widget.NewButton("选择保存目录", func() {
		dialog.NewFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, r.win)
				return
			}
			if u == nil {
				return
			}
			r.saveDir = u.Path()
			r.pathLabel.SetText("保存目录: " + r.saveDir)
		}, r.win).Show()
	})
	resumeCheck := widget.NewCheck("断点续传（从 .resume 恢复）", func(v bool) {
		r.resume = v
	})
	resumeCheck.SetChecked(false)

	r.startBtn = widget.NewButton("开始接收", func() { r.start() })
	r.stopBtn = widget.NewButton("停止", func() { r.stop() })
	r.stopBtn.Disable()

	top := container.NewVBox(
		container.NewHBox(selectDir, r.startBtn, r.stopBtn),
		resumeCheck,
		r.pathLabel,
		r.progressLabel,
		r.missingLabel,
		r.progressBar,
		r.statusLabel,
	)
	r.win.SetContent(container.NewBorder(top, nil, nil, nil, r.logBox))
}

func (r *ReceiverUI) start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running {
		return
	}
	if r.saveDir == "" {
		dialog.ShowInformation("提示", "请先选择保存目录", r.win)
		return
	}
	if err := os.MkdirAll(r.saveDir, 0o755); err != nil {
		dialog.ShowError(err, r.win)
		return
	}
	r.running = true
	r.stopCh = make(chan struct{})
	r.state = nil
	r.logBox.SetText("")
	r.progressLabel.SetText("Received: 0/0")
	r.missingLabel.SetText("Missing chunks: 0")
	r.progressBar.SetValue(0)
	r.statusLabel.SetText("Status: Scanning screen...")
	r.startBtn.Disable()
	r.stopBtn.Enable()

	go r.loop(r.stopCh)
}

func (r *ReceiverUI) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.running {
		return
	}
	close(r.stopCh)
	r.running = false
	r.startBtn.Enable()
	r.stopBtn.Disable()
	r.statusLabel.SetText("Status: Stopped")
}

func (r *ReceiverUI) loop(stopCh chan struct{}) {
	ticker := time.NewTicker(captureInterval)
	defer ticker.Stop()
	lastPersist := time.Now()
	lockedDisplay := -1

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			displayCount := NumDisplays()
			if displayCount <= 0 {
				r.addLog("capture error: no active displays detected")
				continue
			}
			for displayIdx := 0; displayIdx < displayCount; displayIdx++ {
				if lockedDisplay >= 0 && displayIdx != lockedDisplay {
					continue
				}
				frame, err := CaptureFrameByDisplay(displayIdx)
				if err != nil {
					continue
				}
				packets, err := decodePackets(frame)
				if err != nil || len(packets) == 0 {
					continue
				}

				for _, p := range packets {
					if p.Type != common.PacketTypeChunk || p.Chunk == nil {
						continue
					}
					if lockedDisplay == -1 {
						lockedDisplay = displayIdx
						r.addLog(fmt.Sprintf("locked on display %d", lockedDisplay))
					}
					if err := r.acceptChunk(p); err != nil {
						r.addLog("chunk ignored: " + err.Error())
					}
					if r.state != nil &&
						r.state.SessionID == p.SessionID &&
						r.state.Total > 0 &&
						len(r.state.Chunks) >= r.state.Total {
						if err := r.flushFile(); err != nil {
							r.addLog("save failed: " + err.Error())
						} else {
							r.setStatus("Status: Completed")
							r.addLog("transfer completed")
							r.stop()
							return
						}
					}
				}
			}

			if r.state != nil && time.Since(lastPersist) > 2*time.Second {
				_ = saveResume(r.saveDir, r.state)
				lastPersist = time.Now()
			}
		}
	}
}

func (r *ReceiverUI) acceptChunk(p common.Packet) error {
	newState := func() *TransferState {
		return &TransferState{
			SessionID: p.SessionID,
			FileName:  p.FileName,
			Total:     p.Chunk.Total,
			Chunks:    map[int][]byte{},
			Seen:      map[int]bool{},
		}
	}

	if r.state != nil && r.state.SessionID != p.SessionID {
		r.addLog("new session detected, reset state")

		r.state = nil
	}
	if p.Chunk.ID < 0 || p.Chunk.Total <= 0 || p.Chunk.ID >= p.Chunk.Total {
		return fmt.Errorf("invalid chunk id=%d total=%d", p.Chunk.ID, p.Chunk.Total)
	}
	if r.state == nil {
		if r.resume {
			s, err := loadResume(r.saveDir, p.SessionID)
			if err == nil {
				if s.Total != p.Chunk.Total {
					r.addLog(fmt.Sprintf("resume ignored due to total mismatch (resume=%d incoming=%d)", s.Total, p.Chunk.Total))
					r.state = newState()
				} else {
					r.state = s
					r.addLog("resumed session: " + p.SessionID)
					r.refreshProgress()
				}
			} else {
				r.state = newState()
			}
		} else {
			r.state = newState()
		}
	}
	fmt.Printf("ACCEPT chunk=%d total=%d chunks=%d seen=%d discarded=%d\n",
		p.Chunk.ID,
		r.state.Total,
		len(r.state.Chunks),
		len(r.state.Seen),
		r.state.Discarded)
	if r.state.SessionID != p.SessionID {
		return fmt.Errorf("session mismatch")
	}
	if r.state.Total != p.Chunk.Total {
		return fmt.Errorf("chunk total mismatch")
	}
	if r.state.Seen[p.Chunk.ID] {
		return nil
	}

	raw, err := common.DecodeBase64(p.Chunk.Data)
	if err != nil {
		r.state.Discarded++
		fmt.Printf("DROP chunk=%d reason=base64 err=%v\n", p.Chunk.ID, err)
		return fmt.Errorf("chunk=%d base64 decode failed: %w", p.Chunk.ID, err)
	}
	if err := common.CheckCRC32(raw, p.Chunk.CRC32); err != nil {
		r.state.Discarded++
		fmt.Printf("DROP chunk=%d reason=crc err=%v\n", p.Chunk.ID, err)
		return fmt.Errorf("chunk=%d crc verify failed: %w", p.Chunk.ID, err)
	}

	r.state.Seen[p.Chunk.ID] = true
	r.state.Chunks[p.Chunk.ID] = raw
	r.refreshProgress()
	return nil
}

func (r *ReceiverUI) refreshProgress() {
	if r.state == nil {
		return
	}
	received := len(r.state.Chunks)
	total := r.state.Total
	missing := total - received
	if missing < 0 {
		missing = 0
	}
	r.progressLabel.SetText(fmt.Sprintf("Received: %d/%d", received, total))
	r.missingLabel.SetText(fmt.Sprintf("Missing chunks: %d | CRC discarded: %d", missing, r.state.Discarded))
	if total > 0 {
		r.progressBar.SetValue(float64(received) / float64(total))
	} else {
		r.progressBar.SetValue(0)
	}
}

func (r *ReceiverUI) flushFile() error {
	if r.state == nil {
		return fmt.Errorf("no active state")
	}
	outputPath := filepath.Join(r.saveDir, r.state.FileName)
	if err := writeOutput(outputPath, r.state.Chunks, r.state.Total); err != nil {
		return err
	}
	_ = os.Remove(resumePath(r.saveDir, r.state.SessionID))
	r.addLog("saved file: " + outputPath)
	return nil
}

func (r *ReceiverUI) addLog(msg string) {
	line := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	cur := r.logBox.Text
	if cur == "" {
		r.logBox.SetText(line)
		return
	}
	if len(cur) > 12000 {
		parts := strings.Split(cur, "\n")
		if len(parts) > 80 {
			cur = strings.Join(parts[len(parts)-80:], "\n")
		}
	}
	r.logBox.SetText(cur + "\n" + line)
}

func (r *ReceiverUI) setStatus(s string) {
	r.statusLabel.SetText(s)
	if r.state != nil {
		r.addLog("missing ids: " + sortedMissing(r.state))
	}
}
