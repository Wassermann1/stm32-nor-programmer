package main

import (
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

var (
	PortsMap = make(map[string]*enumerator.PortDetails)
	PortList []*enumerator.PortDetails
)

// DataProvider - wrap interface to save binding data
type DataProvider interface {
	Get() ([]byte, error)
}
type StringProvider struct {
	b binding.String
}

func (p StringProvider) Get() ([]byte, error) {
	s, err := p.b.Get()
	return []byte(s), err
}

/* File operations */

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
		if err = TableData.Set(data); err != nil {
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

func SaveAs(a fyne.App, data DataProvider) {
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

			bytes, err := data.Get()
			if err != nil {
				slog.Error("ERROR getting data:", "err", err)
				dialog.ShowError(err, w)
				return
			}

			_, err = writer.Write(bytes)
			if err != nil {
				slog.Error("ERROR writing to file:", "err", err)
				dialog.ShowError(err, w)
				return
			}

			slog.Info("File saved", "path", writer.URI().Path(), "bytes", len(bytes))
		}, w,
	)

	fileSave.Resize(fyne.NewSize(800, 600))
	fileSave.Show()
}

/* Port operation */

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
			PortsMap[info] = port
		}
	}
}

func OpenPort(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		slog.Error("ERROR getting selected port:", "err", err)
		dialog.ShowError(err, w)
	}
	mode := &serial.Mode{BaudRate: 115200}
	Port, err = serial.Open(PortsMap[name].Name, mode)
	if err != nil {
		slog.Error("ERROR opening port:", "err", err)
		if strings.Contains(err.Error(), "Permission") {
			dialog.ShowInformation(
				"Permission denied",
				"Cannot open serial port.\n\n"+
					"Add yourself to the proper group:\n"+
					"sudo usermod -a -G uucp $USER - for Arch\n"+
					"sudo usermod -a -G dialout $USER - for Ubuntu\n"+
					"Then log out and log back in.",
				w)
		}
		return
	}
	Port.SetReadTimeout(100 * time.Millisecond)
	slog.Info("Connected", "port", PortsMap[name].Name)
	resp, err := ReadResponse(Port)
	if err != nil {
		slog.Error("ERROR reading response:", "err", err)
		dialog.ShowError(err, w)
	}
	slog.Info(resp)
	fyne.Do(func() {
		ReadIDButton.Enable()
		DeviceInfoMenu.Disabled = false
		CloseDeviceMenu.Disabled = false
		OpenDeviceMenu.Disabled = true
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
		OpenDeviceMenu.Disabled = false
		DeviceInfoMenu.Disabled = true
		CloseDeviceMenu.Disabled = true

	}
	return
}
