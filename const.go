package main

import (
	"embed"
	"io"
	"log/slog"
	"os"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"go.bug.st/serial"
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

var (
	Ports          = binding.NewStringList()
	Port           serial.Port
	SelectedPortID = binding.NewString()
	TableData      = binding.NewBytes()
)

var (
	//go:embed static
	res embed.FS
)

var (
	ConnectProgrammerButton,
	ReadIDButton,
	ReadDumpButton,
	WriteDumpButton,
	VerifyDumpButton,
	EraseChipButton *widget.Button
)

type progressUpdate struct {
	progress float64
	status   string
}

type readResult struct {
	data []byte
	err  error
}
