package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// CreateFirmwareTable - set up dump preview table
func CreateFirmwareTable() fyne.CanvasObject {
	t := widget.NewTable(
		func() (int, int) {
			data, _ := TableData.Get()
			return (len(data) + 15) / 16, 16
		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Alignment = fyne.TextAlignCenter
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			data, _ := TableData.Get()
			label := co.(*widget.Label)
			index := tci.Row*16 + tci.Col
			if index < len(data) {
				label.SetText(fmt.Sprintf("%02X", data[index]))
			} else {
				label.SetText("")
			}
		},
	)

	// Включаем встроенные заголовки
	t.ShowHeaderRow = true
	t.ShowHeaderColumn = true
	t.HideSeparators = false
	t.StickyColumnCount = 0

	t.CreateHeader = func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.Alignment = fyne.TextAlignCenter
		l.TextStyle = fyne.TextStyle{Bold: true}
		return l
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		label := co.(*widget.Label)
		switch {
		case tci.Row == -1:
			// Заголовок колонки
			label.SetText(fmt.Sprintf("%02X", tci.Col))
		case tci.Col == -1:
			// Заголовок строки (адрес)
			label.SetText(fmt.Sprintf("%08X", tci.Row*16))
		}
	}

	// Ширина колонки адресов и данных
	t.SetColumnWidth(-1, 110) // колонка заголовков строк
	for i := 0; i < 16; i++ {
		t.SetColumnWidth(i, 40)
	}

	TableData.AddListener(binding.NewDataListener(func() {
		t.Refresh()
		t.Resize(t.Size())
	}))

	return container.NewVScroll(t)
}
