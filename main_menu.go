package main

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

func makeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {

	// a quit item will be appended to our first (File) menu
	openFile := fyne.NewMenuItem("Open", func() { OpenFile(w) })
	saveAsFile := fyne.NewMenuItem("Save As", func() { SaveAs(w) })
	file := fyne.NewMenu("File", openFile, saveAsFile)

	// Device menu
	refreshDeviceList := fyne.NewMenuItem("Refresh Device List", func() { GetPorts() })
	openDevice := fyne.NewMenuItem("Open Device", func() { OpenPort(w) })
	deviceInfo := fyne.NewMenuItem("Device Info", func() { DeviceInfo(w) })
	closeDevice := fyne.NewMenuItem("Close Device", nil)
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
		log.Println(err)
	}
	if len(PortList) == 0 {
		log.Print("ERROR: No serial ports found!")
	}
	Ports.Set([]string{})
	for _, port := range PortList {
		if port.IsUSB {
			log.Printf("found port %s\n", port.Product)
			info := port.Product
			Ports.Append(info)
			PortsMap[port.Product] = port
		}
	}
}

func OpenPort(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		log.Println(err)
	}
	mode := &serial.Mode{BaudRate: 115200}
	Port, err = serial.Open(PortsMap[name].Name, mode)
	if err != nil {
		log.Print(err)
		return
	}
	Port.SetReadTimeout(100 * time.Millisecond)
	info := fmt.Sprintf("Connected to port %s", PortsMap[name].Product)
	log.Println(info)
	resp, err := ReadResponse(Port)
	if err != nil {
		log.Println(err)
	}
	log.Print(resp)
	fyne.Do(func() {
		ReadIDButton.Enable()
	})
	dialog.ShowInformation("Connected", resp, w)
	ConnectProgrammerButton.Disable()
}

func DeviceInfo(w fyne.Window) {
	name, err := SelectedPortID.Get()
	if err != nil {
		log.Println(err)
	}
	port := PortsMap[name]
	log.Printf("Port name:     %s\n", port.Name)
	log.Printf("Product:       %s\n", port.Product)
	log.Printf("Serial number: %s\n", port.SerialNumber)
	log.Printf("VID | PID:     %s | %s\n", port.VID, port.PID)

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

func SaveAs(w fyne.Window) {
	fileSave := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				log.Println("file save cancelled or error:", err)
				return
			}
			if writer == nil {
				return
			}

			defer func() {
				if cerr := writer.Close(); cerr != nil {
					log.Println("error closing file:", cerr)
					dialog.ShowError(cerr, w)
				}
			}()

			data, err := tableData.Get()
			if err != nil {
				log.Println("error getting dump data:", err)
				dialog.ShowError(err, w)
				return
			}

			_, err = writer.Write(data)
			if err != nil {
				log.Println("error writing to file:", err)
				dialog.ShowError(err, w)
				return
			}

			log.Printf("✅ Dump saved: %s (%d bytes)", writer.URI().Path(), len(data))
		}, w,
	)

	fileSave.Resize(fyne.NewSize(800, 600))
	fileSave.Show()
}

func OpenFile(w fyne.Window) {
	fileOpen := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			log.Println("file save cancelled or error:", err)
			return
		}
		if reader == nil {
			return
		}
		defer func() {
			if cerr := reader.Close(); cerr != nil {
				log.Println("error closing file:", cerr)
				dialog.ShowError(cerr, w)
			}
		}()

		data, err := io.ReadAll(reader)
		if err != nil {
			log.Println("error reading from file:", err)
			dialog.ShowError(err, w)
			return
		}
		if err = tableData.Set(data); err != nil {
			log.Println("error setting dump data:", err)
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
			log.Println("error setting dump data:", err)
			dialog.ShowError(err, w)
		}
		if err := CurrentFlash.Set(nil); err != nil {
			log.Println("error setting dump data:", err)
			dialog.ShowError(err, w)
		}
	}
	return
}
