package main

import (
	"fmt"
	"io"
	"log/slog"
	"strconv"
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
	openFile := fyne.NewMenuItem("Open", func() { OpenFile(a) })
	saveAsFile := fyne.NewMenuItem("Save As", func() { SaveAs(a, tableData) })
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
		slog.Error("ERROR getting ports:", "err", err)
	}
	if len(PortList) == 0 {
		slog.Error("ERROR: No serial ports found!")
	}
	Ports.Set([]string{})
	for _, port := range PortList {
		if port.IsUSB {
			info := port.Product
			if info == "" {
				info = port.Name
			}
			slog.Info("found port", "port", info)
			Ports.Append(info)
			PortsMap[port.Product] = port
		}
	}
}

func OpenPort(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		slog.Error("ERROR getting selected port:", "err", err)
	}
	mode := &serial.Mode{BaudRate: 115200}
	p := PortsMap[name].Name
	if p == "" {
		p = PortsMap[name].Product
	}
	Port, err = serial.Open(p, mode)
	if err != nil {
		slog.Error("ERROR opening port:", "err", err)
		return
	}
	Port.SetReadTimeout(100 * time.Millisecond)
	info := fmt.Sprintf("Connected to port %s", PortsMap[name].Product)
	slog.Info(info)
	resp, err := ReadResponse(Port)
	if err != nil {
		slog.Error("ERROR reading response:", "err", err)
	}
	slog.Info(resp)
	fyne.Do(func() {
		ReadIDButton.Enable()
	})
	dialog.ShowInformation("Connected", resp, w)
	ConnectProgrammerButton.Disable()
}

func DeviceInfo(w fyne.Window) {
	flash, err := CurrentFlash.Get()
	if err != nil {
		slog.Error("ERROR getting current flash:", "err", err)
		dialog.ShowError(err, w)
		return
	}

	form := widget.NewForm(
		widget.NewFormItem("Flash name", widget.NewLabel(flash.Name)),
		widget.NewFormItem("Capacity", widget.NewLabel(strconv.Itoa(flash.Capacity)+" bytes")),
		widget.NewFormItem("DeviceID 1", widget.NewLabel(flash.DeviceID)),
		widget.NewFormItem("DeviceID 2", widget.NewLabel(flash.DeviceID2)),
		widget.NewFormItem("DeviceID 3", widget.NewLabel(flash.DeviceID3)),
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

func SaveAs(a fyne.App, data binding.Item[[]byte]) {
	w := a.NewWindow("Save File")
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(800, 600))
	w.Show()
	fileSave := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			defer w.Close()
			if err != nil {
				slog.Error("ERROR saving file:", "err", err)
				return
			}
			if writer == nil {
				return
			}

			defer func() {
				if cerr := writer.Close(); cerr != nil {
					slog.Error("ERROR closing file:", "err", cerr)
					dialog.ShowError(cerr, w)
				}
			}()

			data, err := data.Get()
			if err != nil {
				slog.Error("ERROR getting data:", "err", err)
				dialog.ShowError(err, w)
				return
			}

			_, err = writer.Write(data)
			if err != nil {
				slog.Error("ERROR writing to file:", "err", err)
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

func OpenFile(a fyne.App) {
	w := a.NewWindow("Open File")
	w.CenterOnScreen()
	w.Resize(fyne.NewSize(800, 600))
	w.Show()
	fileOpen := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		defer w.Close()
		if err != nil {
			slog.Error("ERROR opening file:", "err", err)
			return
		}
		if reader == nil {
			return
		}
		defer func() {
			if cerr := reader.Close(); cerr != nil {
				slog.Error("ERROR closing file:", "err", cerr)
				dialog.ShowError(cerr, w)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			slog.Error("ERROR reading from file:", "err", err)
			dialog.ShowError(err, w)
			return
		}
		if err = tableData.Set(data); err != nil {
			slog.Error("ERROR setting dump data:", "err", err)
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
			slog.Error("ERROR setting dump data:", "err", err)
			dialog.ShowError(err, w)
		}
		if err := CurrentFlash.Set(nil); err != nil {
			slog.Error("ERROR setting dump data:", "err", err)
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
