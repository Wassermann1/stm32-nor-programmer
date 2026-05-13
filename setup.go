package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// ReadID - reads NOR ID, if NOR supported - calls backend to tset prper timings
func ReadID(w fyne.Window) {
	if Port == nil {
		err := fmt.Errorf("port not open")
		slog.Error("ERROR reading ID", "err", err)
		dialog.ShowError(err, w)
		return
	}

	var err error
	if _, err = Port.Write([]byte("READ_ID\r\n")); err != nil {
		slog.Error("ERROR writing to port", "err", err)
		dialog.ShowError(err, w)
		return
	}

	reader := bufio.NewReader(Port)
	line := ""
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			slog.Error("ERROR reading from port", "err", err)
			dialog.ShowError(err, w)
			return
		}
		line = strings.TrimSpace(line)
		if line != "" {
			break
		}
	}
	line = strings.TrimSpace(line)
	if line == "ERR" {
		err = fmt.Errorf("device returned error or empty")
		slog.Error("ERROR reading ID", "err", err)
		dialog.ShowError(err, w)
		return
	}

	parts := strings.Fields(line)
	if len(parts) < 4 {
		err = fmt.Errorf("unexpected format: %s", line)
		slog.Error("ERROR reading ID", "err", err)
		dialog.ShowError(err, w)
		return
	}
	manufID := parts[0]
	devID := parts[1]
	devID2 := parts[2]
	devID3 := parts[3]
	flash := LookupFlash(manufID, devID, devID2, devID3)

	if flash == nil {
		err = fmt.Errorf("unsupported NOR flash ManID: %s DevID: %s DevID2: %s DevID3: %s",
			manufID, devID, devID2, devID3)
		slog.Error("ERROR reading ID", "err", err)
		dialog.ShowError(err, w)
		return
	}

	if err = CurrentFlash.Set(flash); err != nil {
		slog.Error("ERROR setting current flash", "err", err)
		dialog.ShowError(err, w)
		return
	}

	fyne.Do(func() {
		ReadDumpButton.Enable()
		EraseChipButton.Enable()
	})
	data, err := TableData.Get()
	if err == nil && data != nil {
		WriteDumpButton.Enable()
	}

	info, err := setTimings(flash, reader)
	if err != nil {
		slog.Error("ERROR setting timings", "err", err)
		dialog.ShowError(err, w)
		return
	}
	slog.Info("FSMC timings applied successfully")
	slog.Info(info)
	info1 := fmt.Sprintf("✔ Detected %s", flash.Name)
	info2 := fmt.Sprintf("Total size: %d Kbytes", (flash.Capacity / 1024))
	slog.Info(info1)
	slog.Info(info2)
	dialog.ShowInformation("Info", info1+"\n"+info2, w)
}

func setTimings(flash *NOR_Flash, reader *bufio.Reader) (string, error) {
	regs, summary := CalcFSMCTimings(&flash.Specs)
	info := fmt.Sprintf("[timing] %s → %s\n", flash.Name, summary)

	cmd := fmt.Sprintf("CONFIG %d %d %d %d\r\n",
		regs.AddressSetupTime,
		regs.AddressHoldTime,
		regs.DataSetupTime,
		regs.BusTurnAroundDuration,
	)
	if _, err := Port.Write([]byte(cmd)); err != nil {
		return "", fmt.Errorf("send config: %w", err)
	}
	resp, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read config ack: %w", err)
	}
	if strings.Contains(strings.TrimSpace(resp), "failed") {
		return "", fmt.Errorf("device rejected timing config: %s", resp)
	}
	return info, nil
}
