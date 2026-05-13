package main

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	flashImageMap = map[string]string{
		"Spansion S29AL016M":              "static/S29AL016M.png",
		"Spansion S29GL064N":              "static/S29GL064N.png",
		"Spansion S29GL128P":              "static/S29GL128P.png",
		"Macronix MX29LV400BT":            "static/MX29LV400BT.png",
		"Excel Semiconductor ES29LV800DB": "static/ES29LV800.png",
		"Toshiba TV005700002":             "static/TV005700002.png",
	}
	flashImageCache = make(map[string]fyne.Resource)
	flashImage      *canvas.Image
	flashLabel      *widget.Label
)

// CreateFlashDisplay - creates current flash lable and refreshes current flas image
func CreateFlashDisplay() (*widget.Label, *fyne.Container) {
	flashImage = canvas.NewImageFromResource(nil)
	flashImage.FillMode = canvas.ImageFillContain

	flashLabel = widget.NewLabel("No Flash Detected")
	flashLabel.TextStyle.Bold = true
	flashLabel.Alignment = fyne.TextAlignCenter

	CurrentFlash.AddListener(binding.NewDataListener(func() {
		fyne.Do(updateFlashDisplay)
	}))
	updateFlashDisplay()

	imageContainer := container.NewStack(flashImage)
	return flashLabel, imageContainer
}

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

		if path, ok := flashImageMap[flash.Name]; ok {
			resource, err := loadEmbeddedImage(path)
			if err != nil {
				slog.Warn("could not load image", "path", path, "err", err)
				flashImage.Resource = theme.NewThemedResource(theme.QuestionIcon())
			} else {
				flashImage.Resource = resource
			}
			flashImage.FillMode = canvas.ImageFillContain
			flashImage.ScaleMode = canvas.ImageScaleSmooth
		} else {
			flashImage.Resource = theme.NewThemedResource(theme.QuestionIcon())
		}
	}

	flashLabel.Refresh()
	flashImage.Refresh()
}

func loadEmbeddedImage(path string) (fyne.Resource, error) {
	if cached, ok := flashImageCache[path]; ok {
		return cached, nil
	}
	data, err := res.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file %s: %w", path, err)
	}

	resource := fyne.NewStaticResource(path, data)
	flashImageCache[path] = resource
	return resource, nil
}
