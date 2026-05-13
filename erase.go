package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ChipErase(w fyne.Window) {
	slog.Info("Erasing chip...")
	progress := dialog.NewCustomWithoutButtons("In progress",
		widget.NewLabel("🔄 Erasing chip...\nPlease wait, this may take up to 5 minutes."), w)
	progress.Resize(fyne.NewSize(400, 150))
	progress.Show()

	go func() {
		err := doChipErase()
		fyne.Do(func() {
			progress.Dismiss()
			if err != nil {
				slog.Error("ERROR erasing chip", "err", err)
				dialog.ShowError(err, w)
			} else {
				info := "✅ Chip erase completed!"
				slog.Info(info)
				dialog.ShowInformation("Success", info, w)
			}
		})
	}()
}

func doChipErase() error {
	if Port == nil {
		return fmt.Errorf("port not open")
	}

	if _, err := Port.Write([]byte("CHIP_ERASE\r\n")); err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	reader := bufio.NewReader(Port)
	deadline := time.Now().Add(300 * time.Second)

	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			if strings.Contains(err.Error(), "no data") ||
				strings.Contains(err.Error(), "timeout") ||
				err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = strings.TrimSpace(line)

		if strings.Contains(line, "SUCCESS") {
			return nil
		}
		if strings.Contains(line, "ERROR") {
			return fmt.Errorf("chip erase failed: device returned ERROR")
		}
	}

	return fmt.Errorf("timeout after 300 seconds")
}
