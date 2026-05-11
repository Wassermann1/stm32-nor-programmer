package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func makeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {

	// a quit item will be appended to our first (File) menu
	openFile := fyne.NewMenuItem("Open", func() { OpenFile(w) })
	saveAsFile := fyne.NewMenuItem("Save As", func() { SaveAs(w, tableData) })
	file := fyne.NewMenu("File", openFile, saveAsFile)

	// Device menu
	refreshDeviceList := fyne.NewMenuItem("Refresh Device List", func() { GetPorts() })
	openDevice := fyne.NewMenuItem("Open Device", func() { OpenPort(w) })
	deviceInfo := fyne.NewMenuItem("Device Info", func() { DeviceInfo(w) })
	closeDevice := fyne.NewMenuItem("Close Device", func() { CloseDevice(w) })
	device := fyne.NewMenu("Device", refreshDeviceList, openDevice, deviceInfo, closeDevice)

	// Main menu container
	main := fyne.NewMainMenu(
		file,
		device,
	)
	return main
}

func GetPorts() {
	var err error
	PortList, err = enumerator.GetDetailedPortsList()
	if err != nil {
		slog.Error("ERROR getting ports:", err)
	}
	if len(PortList) == 0 {
		slog.Error("ERROR: No serial ports found!")
	}
	Ports.Set([]string{})
	for _, port := range PortList {
		if port.IsUSB {
			slog.Info("found port %s\n", port.Product)
			info := port.Product
			Ports.Append(info)
			PortsMap[port.Product] = port
		}
	}
}

func OpenPort(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		slog.Error("ERROR getting selected port:", err)
	}
	mode := &serial.Mode{BaudRate: 115200}
	Port, err = serial.Open(PortsMap[name].Name, mode)
	if err != nil {
		slog.Error("ERROR opening port:", err)
		return
	}
	Port.SetReadTimeout(100 * time.Millisecond)
	info := fmt.Sprintf("Connected to port %s", PortsMap[name].Product)
	slog.Info(info)
	resp, err := ReadResponse(Port)
	if err != nil {
		slog.Error("ERROR reading response:", err)
	}
	slog.Info(resp)
	fyne.Do(func() {
		ReadIDButton.Enable()
	})
	dialog.ShowInformation("Connected", resp, w)
	ConnectProgrammerButton.Disable()
}

func DeviceInfo(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		slog.Error("ERROR getting selected port:", err)
	}
	port := PortsMap[name]
	slog.Info("Port name:     %s", port.Name)
	slog.Info("Product:       %s", port.Product)
	slog.Info("Serial number: %s", port.SerialNumber)
	slog.Info("VID | PID:     %s | %s", port.VID, port.PID)

	// Формируем текст информации
	form := widget.NewForm(
		widget.NewFormItem("Port name", widget.NewLabel(port.Name)),
		widget.NewFormItem("Product", widget.NewLabel(port.Product)),
		widget.NewFormItem("Serial number", widget.NewLabel(port.SerialNumber)),
		widget.NewFormItem("VID | PID", widget.NewLabel(
			fmt.Sprintf("%04X | %04X", port.VID, port.PID),
		)),
	)

	// Показываем диалог
	content := container.NewPadded(form)
	d := dialog.NewCustom("Device Info", "Close", content, w)
	d.Resize(fyne.NewSize(350, 200))
	d.Show()
}

func ReadResponse(port serial.Port) (string, error) {
	buf := make([]byte, 256)
	var out strings.Builder
	for {
		n, err := port.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			break
		}
		out.Write(buf[:n])
	}

	return out.String(), nil
}

func SaveAs(w fyne.Window, data binding.Item[[]byte]) {
	fileSave := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				slog.Error("ERROR saving file:", err)
				return
			}
			if writer == nil {
				return
			}

			defer func() {
				if cerr := writer.Close(); cerr != nil {
					slog.Error("ERROR closing file:", cerr)
					dialog.ShowError(cerr, w)
				}
			}()

			data, err := data.Get()
			if err != nil {
				slog.Error("ERROR getting data:", err)
				dialog.ShowError(err, w)
				return
			}

			_, err = writer.Write(data)
			if err != nil {
				slog.Error("ERROR writing to file:", err)
				dialog.ShowError(err, w)
				return
			}

			slog.Info("Dump saved: %s (%d bytes)", writer.URI().Path(), len(data))
		}, w,
	)

	fileSave.SetFileName(".bin")
	fileSave.Resize(fyne.NewSize(800, 600))
	fileSave.Show()
}

func OpenFile(w fyne.Window) {
	fileOpen := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			slog.Error("ERROR opening file:", err)
			return
		}
		if reader == nil {
			return
		}
		defer func() {
			if cerr := reader.Close(); cerr != nil {
				slog.Error("ERROR closing file:", cerr)
				dialog.ShowError(cerr, w)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			slog.Error("ERROR reading from file:", err)
			dialog.ShowError(err, w)
			return
		}
		if err = tableData.Set(data); err != nil {
			slog.Error("ERROR setting dump data:", err)
			dialog.ShowError(err, w)
			return
		}
		flash, err := CurrentFlash.Get()
		if err == nil && flash != nil {
			WriteDumpButton.Enable()
		}
	}, w)
	fileOpen.Resize(fyne.NewSize(800, 600))
	fileOpen.Show()
}

func CloseDevice(w fyne.Window) {
	if Port != nil {
		Port.Close()
		Port = nil
		PortList = []*enumerator.PortDetails{}
		if err := SelectedPortID.Set(""); err != nil {
			slog.Error("ERROR setting dump data:", err)
			dialog.ShowError(err, w)
		}
		if err := CurrentFlash.Set(nil); err != nil {
			slog.Error("ERROR setting dump data:", err)
			dialog.ShowError(err, w)
		}
		ConnectProgrammerButton.Disable()
		ReadIDButton.Disable()
		ReadDumpButton.Disable()
		WriteDumpButton.Disable()
		VerifyDumpButton.Disable()
		EraseChipButton.Disable()
	}
	return
}
