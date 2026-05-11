package main

import (
	"embed"
	"io"
	"log/slog"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

var LogBuffer = binding.NewString()

// io.Writer implementation for slog
type bindingWriter struct {
	b binding.String
}

func (w *bindingWriter) Write(p []byte) (n int, err error) {
	cur, _ := w.b.Get()
	_ = w.b.Set(cur + string(p))
	return len(p), nil
}

func initLogger() {
	multi := io.MultiWriter(os.Stdout, &bindingWriter{LogBuffer})
	logger := slog.New(slog.NewTextHandler(multi, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, a.Value.Time().Format("02/01 15:04:05"))
			}
			return a
		},
	}))
	slog.SetDefault(logger)
}

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

	flashImage *canvas.Image
	flashLabel *widget.Label
)

var (
	//go:embed static
	res           embed.FS
	flashImageMap = map[string]string{
		"Spansion S29AL016M":              "static/S29AL016M.png",
		"Spansion S29GL064N":              "static/S29GL064N.png",
		"Spansion S29GL128P":              "static/S29GL128P.png",
		"Macronix MX29LV400BT":            "static/MX29LV400BT.png",
		"Excel Semiconductor ES29LV800DB": "static/ES29LV800.png",
		"Toshiba TV005700002":             "static/TV005700002.png",
	}
	flashImageCache = make(map[string]fyne.Resource)
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

type progressUpdate struct {
	progress float64
	status   string
}

type readResult struct {
	data []byte
	err  error
}
