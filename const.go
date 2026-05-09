package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

const (
	chunkSize    = 512
	writeTimeout = 100 * time.Second
	readTimeout  = 3 * time.Second
)

var (
	PortList     []*enumerator.PortDetails
	SelectedPort *enumerator.PortDetails
	Ports        = binding.NewStringList()
	PortsMap     = make(map[string]*enumerator.PortDetails)
	Port         serial.Port

	SelectedPortID = binding.NewString()

	tableData = binding.NewBytes()

	flashImage    *canvas.Image
	flashLabel    *widget.Label
	flashImageMap = map[string]string{
		"S29AL016M":            "static/S29AL016M.png",
		"S29GL064N":            "static/S29GL064N.png",
		"Spansion S29GL128P":   "static/S29GL128P.png",
		"Macronix MX29LV400BT": "static/MX29LV400BT.png",
		"ES29LV800":            "static/ES29LV800.png",
	}
)

var (
	ConnectProgrammerButton,
	ReadIDButton,
	ReadDumpButton,
	WriteDumpButton,
	VerifyDumpButton,
	EraseChipButton *widget.Button
	Actions *fyne.Container
)

// Структура для передачи прогресса (прогресс + опциональный статус)
type progressUpdate struct {
	progress float64
	status   string
}

// Структура для результата чтения
type readResult struct {
	data []byte
	err  error
}
