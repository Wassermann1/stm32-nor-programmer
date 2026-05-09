package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func addToolbar() *fyne.Container {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			// Действие для кнопки "Add"
		}),
		widget.NewToolbarAction(theme.ContentRemoveIcon(), func() {
			// Действие для кнопки "Remove"
		}),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			// Действие для кнопки "Refresh"
			GetPorts()
		}),
	)

	p, err := Ports.Get()
	if err != nil {
		log.Fatal(err)
	}
	portSelector := widget.NewSelect(p, func(value string) {
		SelectedPortID.Set(value)
		ConnectProgrammerButton.Enable()
	})
	portSelector.PlaceHolder = "Select Port"
	Ports.AddListener(binding.NewDataListener(func() {
		newPorts, err := Ports.Get()
		if err != nil {
			log.Println("ERROR getting ports:", err)
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
		log.Println("Ports updated")
	}))

	return container.NewAdaptiveGrid(2, portSelector, toolbar)
}
