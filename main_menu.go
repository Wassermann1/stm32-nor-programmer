package main

import (
	"fyne.io/fyne/v2"
)

var (
	OpenDeviceMenu,
	DeviceInfoMenu,
	CloseDeviceMenu *fyne.MenuItem
)

// MakeMenu - drop-down menu init
func MakeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {

	// a quit item will be appended to our first (File) menu
	openFileMenu := fyne.NewMenuItem("Open", func() { OpenFile(a) })
	saveAsFileMenu := fyne.NewMenuItem("Save As", func() { SaveAs(a, TableData) })
	file := fyne.NewMenu("File", openFileMenu, saveAsFileMenu)

	// Device menu
	refreshDeviceList := fyne.NewMenuItem("Refresh Device List", func() { GetPorts() })
	OpenDeviceMenu = fyne.NewMenuItem("Open Device", func() { OpenPort(w) })
	OpenDeviceMenu.Disabled = true
	DeviceInfoMenu = fyne.NewMenuItem("Device Info", func() { DeviceInfo(w) })
	DeviceInfoMenu.Disabled = true
	CloseDeviceMenu = fyne.NewMenuItem("Close Device", func() { CloseDevice(w) })
	CloseDeviceMenu.Disabled = true
	device := fyne.NewMenu("Device", refreshDeviceList, OpenDeviceMenu, DeviceInfoMenu, CloseDeviceMenu)

	// Main menu container
	main := fyne.NewMainMenu(
		file,
		device,
	)
	return main
}
