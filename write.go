package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	chunkSize    = 512
	writeTimeout = 100 * time.Second
	readTimeout  = 3 * time.Second
)

// WriteFirmware - write currently load binary to NOR
func WriteFirmware(w fyne.Window) {
	if Port == nil {
		dialog.ShowError(fmt.Errorf("port not open"), w)
		return
	}

	data, err := TableData.Get()
	if err != nil {
		dialog.ShowError(fmt.Errorf("error getting dump data: %w", err), w)
		return
	}

	flash, err := CurrentFlash.Get()
	if err != nil || flash == nil {
		dialog.ShowError(fmt.Errorf("no supported NOR connected"), w)
		return
	}

	if len(data) > flash.Capacity {
		dialog.ShowError(fmt.Errorf("firmware size (%d) exceeds NOR capacity (%d)", len(data), flash.Capacity), w)
		return
	}

	progBar := widget.NewProgressBar()
	progBar.Min = 0
	progBar.Max = 1
	progBar.Value = 0

	statusLabel := widget.NewLabel("📦 Write in progress...")
	statusLabel.TextStyle.Monospace = true

	progressContent := container.NewVBox(
		statusLabel,
		progBar,
	)

	progressDialog := dialog.NewCustomWithoutButtons("Writing Firmware", progressContent, w)
	progressDialog.Resize(fyne.NewSize(400, 150))
	progressDialog.Show()

	progChan := make(chan float64, 10)
	resultChan := make(chan error, 1)

	go func() {
		err := doWriteFirmware(data, flash, progChan)
		resultChan <- err
		close(progChan)
	}()

	go func() {
		for {
			select {
			case val, ok := <-progChan:
				if !ok {
					return
				}
				fyne.Do(func() {
					progBar.SetValue(val)
				})
			case <-time.After(5 * time.Minute):
				fyne.Do(func() {
					progressDialog.Dismiss()
					dialog.ShowError(fmt.Errorf("operation timeout"), w)
				})
				return
			}
		}
	}()

	go func() {
		err := <-resultChan

		fyne.Do(func() {
			progressDialog.Dismiss()

			if err != nil {
				slog.Error("ERROR writing firmware", "err", err)
				dialog.ShowError(err, w)
			} else {
				slog.Info("Firmware write completed!")
				dialog.ShowInformation("Success", "✅ Firmware written successfully!", w)
				VerifyDumpButton.Enable()
			}
		})
	}()
}

func doWriteFirmware(data []byte, flash *NOR_Flash, progChan chan<- float64) error {
	if Port == nil {
		return fmt.Errorf("port not open")
	}

	totalChunks := (len(data) + chunkSize - 1) / chunkSize
	info := fmt.Sprintf("Starting firmware write: %d bytes in %d chunk(s)", len(data), totalChunks)
	slog.Info(info)

	reader := bufio.NewReader(Port)

	for addr := 0; addr < len(data); addr += chunkSize {
		end := addr + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[addr:end]
		chunkLen := len(chunk)
		chunkNum := addr/chunkSize + 1

		// WRITE_BIN
		cmd := fmt.Sprintf("WRITE_BIN %x %x\r\n", addr, chunkLen)
		if _, err := Port.Write([]byte(cmd)); err != nil {
			return fmt.Errorf("write command failed at 0x%x: %w", addr, err)
		}

		// "READY"
		if err := waitForString(reader, "READY", readTimeout); err != nil {
			return fmt.Errorf("no READY at 0x%x: %w", addr, err)
		}

		// Send Data
		Port.ResetOutputBuffer()
		Port.ResetInputBuffer()
		if _, err := Port.Write(chunk); err != nil {
			return fmt.Errorf("write chunk at 0x%x failed: %w", addr, err)
		}

		// "WRITE_OK" or ERR
		response, err := waitForAnyString(reader, []string{"WRITE_OK", "Error"}, writeTimeout)
		if err != nil {
			return fmt.Errorf("timeout/error at 0x%x: %w", addr, err)
		}
		if strings.Contains(response, "Error") {
			return fmt.Errorf("device reported error at 0x%x", addr)
		}

		progress := float64(chunkNum) / float64(totalChunks)
		select {
		case progChan <- progress:
		default:
		}
	}

	slog.Info("Resetting flash to read mode... ")
	if _, err := Port.Write([]byte("RESET_FLASH\r\n")); err != nil {
		return fmt.Errorf("reset command failed: %w", err)
	}
	if err := waitForString(reader, "OK", readTimeout); err != nil {
		return fmt.Errorf("no OK after reset: %w", err)
	}
	slog.Info("Write done")
	progChan <- 1.0

	return nil
}

func waitForString(reader *bufio.Reader, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			if strings.Contains(err.Error(), "no data") ||
				strings.Contains(err.Error(), "timeout") ||
				err == io.EOF {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return err
		}
		line = strings.TrimSpace(line)
		if strings.Contains(line, target) {
			return nil
		}
	}
	return fmt.Errorf("timeout waiting for '%s'", target)
}

func waitForAnyString(reader *bufio.Reader, targets []string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			if strings.Contains(err.Error(), "no data") ||
				strings.Contains(err.Error(), "timeout") ||
				err == io.EOF {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			return "", err
		}
		line = strings.TrimSpace(line)
		for _, t := range targets {
			if strings.Contains(line, t) {
				return line, nil
			}
		}
		if line != "" {
			fmt.Print(line + " ")
		}
	}
	return "", fmt.Errorf("timeout waiting for %v", targets)
}
