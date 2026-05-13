package main

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// AddToolbar - main window toolbar setup
func AddToolbar(a fyne.App) *fyne.Container {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			GetPorts()
		}),
		widget.NewToolbarAction(theme.DocumentIcon(), func() {
			OpenFile(a)
		}),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			SaveAs(a, TableData)
		}),
		widget.NewToolbarAction(theme.FileTextIcon(), func() { showLog(a) }),
	)

	p, err := Ports.Get()
	if err != nil {
		slog.Error("ERROR getting ports:", "err", err)
	}
	portSelector := widget.NewSelect(p, func(value string) {
		SelectedPortID.Set(value)
		ConnectProgrammerButton.Enable()
		OpenDeviceMenu.Disabled = false
	})
	portSelector.PlaceHolder = "Select Port"
	Ports.AddListener(binding.NewDataListener(func() {
		newPorts, err := Ports.Get()
		if err != nil {
			slog.Error("ERROR getting ports:", "err", err)
			return
		}
		portSelector.Options = newPorts
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
	}))

	return container.NewAdaptiveGrid(2, portSelector, toolbar)
}

func showLog(a fyne.App) {
	logWindow := a.NewWindow("Log")
	logText := widget.NewMultiLineEntry()
	logText.Bind(LogBuffer)
	logText.Wrapping = fyne.TextWrapBreak
	saveBtn := widget.NewButton("Save Log", func() { SaveAs(a, StringProvider{LogBuffer}) })
	logWindow.SetContent(container.NewBorder(nil, saveBtn, nil, nil, logText))
	logWindow.CenterOnScreen()
	logWindow.Resize(fyne.NewSize(800, 600))
	logWindow.Show()
}
