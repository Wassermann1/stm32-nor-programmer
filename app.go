package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {

	initLogger()

	a := app.New()
	w := a.NewWindow("NOR Flash Programmer")

	//Buttons
	ConnectProgrammerButton = widget.NewButton("Connect Programmer", func() { OpenPort(w) })
	ConnectProgrammerButton.Disable()
	ReadIDButton = widget.NewButton("Read ID", func() { ReadID(w) })
	ReadIDButton.Disable()
	ReadDumpButton = widget.NewButton("Read Dump", func() { ReadDumpBinary(w) })
	ReadDumpButton.Disable()
	WriteDumpButton = widget.NewButton("Write Dump", func() { WriteFirmware(w) })
	WriteDumpButton.Disable()
	VerifyDumpButton = widget.NewButton("Verify Dump", func() { VerifyDump(w) })
	VerifyDumpButton.Disable()
	EraseChipButton = widget.NewButton("Erase Chip", func() { ChipErase(w) })
	EraseChipButton.Disable()

	flashLabel, flashImage := CreateFlashDisplay()

	Actions := container.NewVBox(
		ConnectProgrammerButton, ReadIDButton, ReadDumpButton,
		WriteDumpButton, VerifyDumpButton, EraseChipButton, flashLabel,
	)

	// Composing main window
	right := container.NewVSplit(Actions, flashImage)
	right.SetOffset(0.0)
	left := CreateFirmwareTable()
	split := container.NewHSplit(left, right)
	split.SetOffset(0.7)
	mainContent := container.NewBorder(AddToolbar(a), nil, nil, nil, split)

	w.SetMainMenu(MakeMenu(a, w))
	w.SetContent(mainContent)
	w.Resize(fyne.NewSize(1240, 768))

	w.ShowAndRun()
}
