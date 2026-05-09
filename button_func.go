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

func ReadID(w fyne.Window) {
	if Port == nil {
		err := fmt.Errorf("Port not open")
		log.Println(err)
		dialog.ShowError(err, w)
		return
	}
	// 1. Отправить команду чтения ID
	var err error
	if _, err = Port.Write([]byte("read_id_fsmc\r\n")); err != nil {
		log.Println(err)
		dialog.ShowError(err, w)
		return
	}

	reader := bufio.NewReader(Port)
	line := ""
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			log.Println(err)
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
		log.Println(err)
		dialog.ShowError(err, w)
		return
	}

	// 2. Парсим "MFR DEV" (например "01 227E")
	parts := strings.Fields(line)
	if len(parts) < 2 {
		err = fmt.Errorf("unexpected format: %s", line)
		log.Println(err)
		dialog.ShowError(err, w)
		return
	}
	manufID := parts[0]
	devID := parts[1]

	for _, flash := range flashSpecs {
		if flash.ManufacturerID == manufID && flash.DeviceID == devID {
			if err = CurrentFlash.Set(&flash); err != nil {
				log.Println(err)
				dialog.ShowError(err, w)
				return
			}
			break
		}
	}

	flash, err := CurrentFlash.Get()
	if err != nil {
		log.Println(err)
		dialog.ShowError(err, w)
		return
	}
	fyne.Do(func() {
		ReadDumpButton.Enable()
		EraseChipButton.Enable()
	})
	data, err := tableData.Get()
	if err == nil && data != nil {
		WriteDumpButton.Enable()
	}

	info := fmt.Sprintf("✔ Detected %s\nTotal size: %d Kbytes", flash.Name, (flash.Capacity / 1024))
	dialog.ShowInformation("Info", info, w)
}

func WriteFirmware(w fyne.Window) {
	// === Предварительные проверки (в главном потоке) ===
	if Port == nil {
		dialog.ShowError(fmt.Errorf("ERROR: Port not open"), w)
		return
	}

	data, err := tableData.Get()
	if err != nil {
		dialog.ShowError(fmt.Errorf("error getting dump data: %w", err), w)
		return
	}

	flash, err := CurrentFlash.Get()
	if err != nil || flash == nil {
		dialog.ShowError(fmt.Errorf("ERROR: Connect supported NOR first"), w)
		return
	}

	if len(data) > flash.Capacity {
		dialog.ShowError(fmt.Errorf("firmware size (%d) exceeds NOR capacity (%d)", len(data), flash.Capacity), w)
		return
	}

	// === Показываем блокирующее окно с прогрессом ===
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
	progressDialog.SetOnClosed(func() {
		// Опционально: обработка закрытия окна пользователем
		// Можно добавить контекст для отмены операции
	})
	progressDialog.Resize(fyne.NewSize(400, 150))
	progressDialog.Show()

	// === Запускаем запись в фоне ===
	// Канал для обновления прогресса из горутины
	progChan := make(chan float64, 10) // буферизированный, чтобы не блокировать запись
	// Канал для результата
	resultChan := make(chan error, 1)

	go func() {
		err := doWriteFirmware(data, flash, progChan)
		resultChan <- err
		close(progChan)
	}()

	// Запускаем цикл обновления прогресс-бара
	go func() {
		for {
			select {
			case val, ok := <-progChan:
				if !ok {
					return // канал закрыт
				}
				// Обновляем UI только через fyne.Do
				fyne.Do(func() {
					progBar.SetValue(val)
				})
			case <-time.After(5 * time.Minute): // таймаут на всю операцию
				// На случай если запись зависла
				fyne.Do(func() {
					progressDialog.Dismiss()
					dialog.ShowError(fmt.Errorf("operation timeout"), w)
				})
				return
			}
		}
	}()

	// === Ожидание завершения операции ===
	// (можно вынести в отдельную горутину, если нужно, чтобы окно оставалось отзывчивым)
	go func() {
		err := <-resultChan

		fyne.Do(func() {
			progressDialog.Dismiss()

			if err != nil {
				log.Println("Write error:", err)
				dialog.ShowError(err, w)
			} else {
				log.Println("✅ Firmware write completed!")
				dialog.ShowInformation("Success", "✅ Firmware written successfully!", w)
				VerifyDumpButton.Enable()
			}
		})
	}()
}

// doWriteFirmware выполняет запись и отправляет прогресс в канал
// Можно вызывать из любой горутины
func doWriteFirmware(data []byte, flash *NOR_Flash, progChan chan<- float64) error {
	if Port == nil {
		return fmt.Errorf("port not open")
	}

	totalChunks := (len(data) + chunkSize - 1) / chunkSize
	log.Printf("📦 Writing %d bytes in %d chunk(s)...", len(data), totalChunks)

	reader := bufio.NewReader(Port)

	for addr := 0; addr < len(data); addr += chunkSize {
		end := addr + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[addr:end]
		chunkLen := len(chunk)
		chunkNum := addr/chunkSize + 1

		// 2.1 Команда WRITE_BIN
		cmd := fmt.Sprintf("WRITE_BIN %x %x\r\n", addr, chunkLen)
		if _, err := Port.Write([]byte(cmd)); err != nil {
			return fmt.Errorf("write command failed at 0x%x: %w", addr, err)
		}

		// 2.2 Ждём "READY"
		if err := waitForString(reader, "READY", readTimeout); err != nil {
			return fmt.Errorf("no READY at 0x%x: %w", addr, err)
		}

		// 2.3 Отправляем бинарные данные
		Port.ResetOutputBuffer()
		Port.ResetInputBuffer()
		if _, err := Port.Write(chunk); err != nil {
			return fmt.Errorf("write chunk at 0x%x failed: %w", addr, err)
		}

		// 2.4 Ждём "WRITE_OK" или ошибку
		response, err := waitForAnyString(reader, []string{"WRITE_OK", "Error"}, writeTimeout)
		if err != nil {
			return fmt.Errorf("timeout/error at 0x%x: %w", addr, err)
		}
		if strings.Contains(response, "Error") {
			return fmt.Errorf("device reported error at 0x%x", addr)
		}

		// 🔥 Отправляем прогресс в канал (неблокирующе, благодаря буферу)
		progress := float64(chunkNum) / float64(totalChunks)
		select {
		case progChan <- progress:
			// отправлено
		default:
			// канал переполнен — пропускаем, чтобы не блокировать запись
		}
	}

	// 3. Сброс в режим чтения
	log.Print("🔄 Resetting flash to read mode... ")
	if _, err := Port.Write([]byte("RESET_FLASH\r\n")); err != nil {
		return fmt.Errorf("reset command failed: %w", err)
	}
	if err := waitForString(reader, "OK", readTimeout); err != nil {
		return fmt.Errorf("no OK after reset: %w", err)
	}
	log.Println("done")

	// Финальное обновление прогресса
	progChan <- 1.0

	return nil
}

// waitForString читает строки из reader, пока не встретит заданную подстроку (таймаут в мс).
func waitForString(reader *bufio.Reader, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Игнорируем временные ошибки (нет данных) – даём ещё время
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

// waitForAnyString аналогична, но ищет любую из нескольких подстрок.
// Возвращает найденную строку (или ошибку).
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

func ChipErase(w fyne.Window) {
	log.Println("🔄 Erasing chip...")
	progress := dialog.NewCustomWithoutButtons("In progress",
		widget.NewLabel("🔄 Erasing chip...\nPlease wait, this may take up to 5 minutes."), w)
	progress.Resize(fyne.NewSize(400, 150))
	progress.Show()

	go func() {
		err := doChipErase()
		fyne.Do(func() {
			progress.Dismiss()
			if err != nil {
				log.Println(err)
				dialog.ShowError(err, w)
			} else {
				info := "✅ Chip erase completed!"
				log.Println(info)
				dialog.ShowInformation("Success", info, w)
			}
		})
	}()
}

func doChipErase() error {
	if Port == nil {
		return fmt.Errorf("port not open")
	}

	// Отправляем команду
	if _, err := Port.Write([]byte("chip_erase\r\n")); err != nil {
		return fmt.Errorf("send command: %w", err)
	}

	reader := bufio.NewReader(Port)
	deadline := time.Now().Add(300 * time.Second)

	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Игнорируем временные ошибки отсутствия данных
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
