package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ReadDumpBinary(w fyne.Window) {
	if Port == nil {
		dialog.ShowError(fmt.Errorf("port not open"), w)
		return
	}
	flash, err := CurrentFlash.Get()
	if err != nil || flash == nil {
		dialog.ShowError(fmt.Errorf("connect supported NOR first"), w)
		return
	}

	progBar := widget.NewProgressBar()
	progBar.Min = 0
	progBar.Max = 1
	progBar.Value = 0
	statusLabel := widget.NewLabel("📥 Preparing to read...")
	statusLabel.TextStyle.Monospace = true

	progressContent := container.NewVBox(
		statusLabel,
		progBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons("Reading Dump", progressContent, w)
	progressDialog.Resize(fyne.NewSize(400, 150))
	progressDialog.Show()

	progChan := make(chan progressUpdate, 10)
	resultChan := make(chan readResult, 1)

	go func() {
		dump, err := doReadDumpBinary(flash.Capacity, progChan)
		resultChan <- readResult{data: dump, err: err}
		close(progChan)
	}()

	go func() {
		for {
			select {
			case upd, ok := <-progChan:
				if !ok {
					return
				}
				fyne.Do(func() {
					progBar.SetValue(upd.progress)
					if upd.status != "" {
						statusLabel.SetText(upd.status)
					}
				})
			case <-time.After(10 * time.Minute):
				fyne.Do(func() {
					progressDialog.Dismiss()
					dialog.ShowError(fmt.Errorf("read operation timeout"), w)
				})
				return
			}
		}
	}()

	go func() {
		result := <-resultChan

		fyne.Do(func() {
			progressDialog.Dismiss()

			if result.err != nil {
				log.Println("Read error:", result.err)
				dialog.ShowError(result.err, w)
			} else {
				if err := tableData.Set(result.data); err != nil {
					dialog.ShowError(fmt.Errorf("error storing dump: %w", err), w)
					return
				}

				info := fmt.Sprintf("✅ Binary dump complete!\nRead %d bytes", len(result.data))
				log.Println(info)
				dialog.ShowInformation("Read finished", info, w)
			}
		})
	}()
}

func doReadDumpBinary(capacity int, progChan chan<- progressUpdate) ([]byte, error) {
	if Port == nil {
		return nil, fmt.Errorf("port not open")
	}

	log.Printf("Starting dump, expecting %d bytes of data", capacity)
	reader := bufio.NewReader(Port)
	dump := make([]byte, 0, capacity)
	cmd := fmt.Sprintf("read_full %d\r\n", capacity)
	if _, err := Port.Write([]byte(cmd)); err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("connection closed while waiting for start marker")
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if strings.Contains(line, "START_DUMP") {
			break
		}
		if strings.Contains(line, "END_DUMP") {
			Port.ResetInputBuffer()
			Port.ResetOutputBuffer()
			log.Println("Empty flash detected")
			return []byte{}, nil
		}
	}

	buf := make([]byte, 4096)
	read := 0
	startTime := time.Now()
	lastReport := startTime

	for read < capacity {
		toRead := capacity - read
		if toRead > len(buf) {
			toRead = len(buf)
		}

		n, err := reader.Read(buf[:toRead])
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read error at byte %d: %w", read, err)
		}
		if n == 0 {
			break
		}

		dump = append(dump, buf[:n]...)
		read += n
		now := time.Now()
		if now.Sub(lastReport) >= 500*time.Millisecond {
			elapsed := now.Sub(startTime).Seconds()
			speed := float64(read) / elapsed / 1024 // KB/s
			progress := float64(read) / float64(capacity)
			status := fmt.Sprintf("📥 %.1f KB/s • %d / %d bytes", speed, read, capacity)

			select {
			case progChan <- progressUpdate{progress: progress, status: status}:
			default:
			}
			lastReport = now
		}
	}

	if read != capacity {
		return nil, fmt.Errorf("read %d bytes but expected %d bytes", read, capacity)
	}
	progChan <- progressUpdate{progress: 1.0, status: "✅ Verifying..."}
	return dump, nil
}

func VerifyDump(w fyne.Window) {
	if Port == nil {
		dialog.ShowError(fmt.Errorf("port not open"), w)
		return
	}

	original, err := tableData.Get()
	if err != nil || len(original) == 0 {
		dialog.ShowError(fmt.Errorf("no firmware data in memory"), w)
		return
	}

	flash, err := CurrentFlash.Get()
	if err != nil || flash == nil {
		dialog.ShowError(fmt.Errorf("connect supported NOR first"), w)
		return
	}

	if len(original) > flash.Capacity {
		dialog.ShowError(fmt.Errorf("firmware size (%d) exceeds flash capacity (%d)",
			len(original), flash.Capacity), w)
		return
	}

	progBar := widget.NewProgressBar()
	progBar.Min, progBar.Max = 0, 1

	statusLabel := widget.NewLabel("🔍 Preparing verification...")
	statusLabel.TextStyle.Monospace = true

	progressContent := container.NewVBox(statusLabel, progBar)
	progressDialog := dialog.NewCustomWithoutButtons("Verifying Dump", progressContent, w)
	progressDialog.Resize(fyne.NewSize(400, 150))
	progressDialog.Show()

	progChan := make(chan progressUpdate, 10)
	resultChan := make(chan []byte, 1)

	go func() {
		dump, err := doReadDumpBinary(len(original), progChan)
		if err != nil {
			resultChan <- nil
		} else {
			resultChan <- dump
		}
		close(progChan)
	}()

	go func() {
		for {
			select {
			case upd, ok := <-progChan:
				if !ok {
					return
				}
				fyne.Do(func() {
					progBar.SetValue(upd.progress)
					if upd.status != "" {
						statusLabel.SetText(strings.Replace(upd.status, "📥", "🔍", 1))
					}
				})
			case <-time.After(10 * time.Minute):
				fyne.Do(func() {
					progressDialog.Dismiss()
					dialog.ShowError(fmt.Errorf("verification timeout"), w)
				})
				return
			}
		}
	}()

	go func() {
		dump := <-resultChan

		fyne.Do(func() {
			progressDialog.Dismiss()
		})

		if dump == nil {
			dialog.ShowError(fmt.Errorf("failed to read Flash for verification"), w)
			return
		}

		for i := range original {
			if original[i] != dump[i] {
				err := fmt.Errorf("verification failed at 0x%X: expected 0x%02X, got 0x%02X",
					i, original[i], dump[i])
				log.Println(err)
				fyne.Do(func() {
					dialog.ShowError(err, w)
				})
				return
			}
		}

		msg := fmt.Sprintf("✅ Verification successful!\n%d bytes matched", len(original))
		log.Println(msg)
		fyne.Do(func() {
			dialog.ShowInformation("Success", msg, w)
		})
	}()
}
