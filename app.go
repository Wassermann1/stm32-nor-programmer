package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

func createFirmwareTable() fyne.CanvasObject {
	t := widget.NewTable(
		func() (int, int) {
			data, _ := tableData.Get()
			return (len(data) + 15) / 16, 16
		},
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Alignment = fyne.TextAlignCenter
			return l
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			data, _ := tableData.Get()
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

	tableData.AddListener(binding.NewDataListener(func() {
		t.Refresh()
		t.Resize(t.Size())
	}))

	return container.NewVScroll(t)
}

func main() {

	a := app.New()

	w := a.NewWindow("NOR Flash Programmer")

	w.SetMainMenu(makeMenu(a, w))

	w.Resize(fyne.NewSize(1240, 768))

	//Buttons
	ConnectProgrammerButton = widget.NewButton("Connect Programmer", func() { OpenPort(w) })
	ConnectProgrammerButton.Disable()
	ReadIDButton = widget.NewButton("Read ID", func() { ReadID(w) })
	ReadIDButton.Disable()
	ReadDumpButton = widget.NewButton("Read Dump", func() { go ReadDumpBinary(w) })
	ReadDumpButton.Disable()
	WriteDumpButton = widget.NewButton("Write Dump", func() { WriteFirmware(w) })
	WriteDumpButton.Disable()
	VerifyDumpButton = widget.NewButton("Verify Dump", func() { VerifyDump(w) })
	VerifyDumpButton.Disable()
	EraseChipButton = widget.NewButton("Erase Chip", func() { ChipErase(w) })
	EraseChipButton.Disable()

	flashLabel, flashImage := createFlashDisplay()

	Actions := container.NewVBox(ConnectProgrammerButton, ReadIDButton, ReadDumpButton, WriteDumpButton, VerifyDumpButton, EraseChipButton, flashLabel)

	right := container.NewVSplit(Actions, flashImage)
	right.SetOffset(0.0)

	left := createFirmwareTable()

	split := container.NewHSplit(left, right)
	split.SetOffset(0.7)

	mainContent := container.NewBorder(addToolbar(), nil, nil, nil, split)

	w.SetContent(mainContent)

	w.ShowAndRun()
}

func createFlashDisplay() (*widget.Label, *fyne.Container) {
	// 1️⃣ Создаём виджеты
	flashImage = canvas.NewImageFromResource(nil)
	flashImage.FillMode = canvas.ImageFillContain

	flashLabel = widget.NewLabel("No Flash Detected")
	flashLabel.TextStyle.Bold = true
	flashLabel.Alignment = fyne.TextAlignCenter

	// 2️⃣ Подписываемся на изменения CurrentFlash
	CurrentFlash.AddListener(binding.NewDataListener(func() {
		// 🔥 Любое обновление UI из binding-слушателя должно идти через fyne.Do
		fyne.Do(updateFlashDisplay)
	}))

	// 3️⃣ Инициализируем текущее состояние (на случай, если флешка уже выбрана)
	updateFlashDisplay()

	imageContainer := container.NewStack(flashImage)
	return flashLabel, imageContainer
}

// updateFlashDisplay вызывается при смене CurrentFlash
func updateFlashDisplay() {
	flash, _ := CurrentFlash.Get()

	if flash == nil || flash.Name == "" {
		flashLabel.SetText("No Flash Detected")
		flashLabel.Importance = widget.WarningImportance
		flashImage.File = ""
		flashImage.Resource = nil
	} else {
		flashLabel.SetText(flash.Name)
		flashLabel.Importance = widget.SuccessImportance

		// Подставляем картинку по модели
		if path, ok := flashImageMap[flash.Name]; ok {
			flashImage.File = path
			flashImage.Resource = nil
		} else {
			// Если модели нет в маппинге — показываем заглушку или скрываем
			flashImage.File = ""
			flashImage.Resource = nil
		}
	}

	flashLabel.Refresh()
	flashImage.Refresh()
}
