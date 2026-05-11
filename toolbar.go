package main

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func addToolbar(a fyne.App) *fyne.Container {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			// Действие для кнопки "Refresh"
			GetPorts()
		}),
		widget.NewToolbarAction(theme.DocumentIcon(), func() {
			// Действие для кнопки "Open"
			OpenFile(a.NewWindow("Open File"))
		}),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			// Действие для кнопки "Save As"
			SaveAs(a.NewWindow("Save File"), tableData)
		}),
		widget.NewToolbarAction(theme.FileTextIcon(), func() { showLog(a) }),
	)

	p, err := Ports.Get()
	if err != nil {
		slog.Error("ERROR getting ports:", err)
	}
	portSelector := widget.NewSelect(p, func(value string) {
		SelectedPortID.Set(value)
		ConnectProgrammerButton.Enable()
	})
	portSelector.PlaceHolder = "Select Port"
	Ports.AddListener(binding.NewDataListener(func() {
		newPorts, err := Ports.Get()
		if err != nil {
			slog.Error("ERROR getting ports:", err)
			return
		}
		portSelector.Options = newPorts // ← Обновляем опции
		if portSelector.Selected != "" {
			found := false
			for _, opt := range newPorts {
				if opt == portSelector.Selected {
					found = true
					break
				}
			}
			if !found {
				portSelector.SetSelected("")
			}
		}
		portSelector.Refresh()
		portSelector.Resize(portSelector.Size())
		slog.Info("Ports updated")
	}))

	return container.NewAdaptiveGrid(2, portSelector, toolbar)
}

func showLog(a fyne.App) {
	logWindow := a.NewWindow("Log")
	logText := widget.NewMultiLineEntry()
	logText.Bind(LogBuffer)
	logText.Wrapping = fyne.TextWrapBreak
	saveBtn := widget.NewButton("Save Log", func() { saveLog(logWindow, LogBuffer) })
	logWindow.SetContent(container.NewBorder(nil, saveBtn, nil, nil, logText))
	logWindow.CenterOnScreen()
	logWindow.Resize(fyne.NewSize(800, 600))
	logWindow.Show()
}

func saveLog(w fyne.Window, data binding.Item[string]) {
	fileSave := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				slog.Error("file save cancelled or error:", err)
				return
			}
			if writer == nil {
				return
			}

			defer func() {
				if cerr := writer.Close(); cerr != nil {
					slog.Error("error closing file:", cerr)
					dialog.ShowError(cerr, w)
				}
			}()

			data, err := data.Get()
			if err != nil {
				slog.Error("error getting data:", err)
				dialog.ShowError(err, w)
				return
			}

			_, err = writer.Write([]byte(data))
			if err != nil {
				slog.Error("error writing to file:", err)
				dialog.ShowError(err, w)
				return
			}

			slog.Info("Log saved: %s (%d bytes)", writer.URI().Path(), len(data))
		}, w,
	)

	fileSave.SetFileName("log.txt")
	fileSave.Resize(fyne.NewSize(800, 600))
	fileSave.Show()
}
