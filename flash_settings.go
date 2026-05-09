package main

import "fyne.io/fyne/v2/data/binding"

type NOR_TimingSpec struct {
	tAS uint16 // address setup to WE# low (ns)
	tAH uint16 // address hold from WE# high (ns)
	tWP uint16 // write enable pulse width (ns)
	tDS uint16 // data setup to WE# high (ns)
}

type NOR_Flash struct {
	Name           string
	ManufacturerID string // hex string without "0x", e.g. "01"
	DeviceID       string // hex string, e.g. "227E"
	Capacity       int    //NOR memory size bytes
	Specs          NOR_TimingSpec
}

const (
	_  = iota             // пропускаем 0
	KB = 1 << (10 * iota) // 1 << (10*1) = 1024
	MB                    // 1 << (10*2) = 1048576
)

var CurrentFlash = binding.NewItem[*NOR_Flash](func(n1, n2 *NOR_Flash) bool {
	if n1 == nil && n2 == nil {
		return true
	}
	if n1 == nil || n2 == nil {
		return false
	}
	return n1.Name == n2.Name
})

var flashSpecs = map[string]NOR_Flash{
	"S29GL128P": {
		Name:           "Spansion S29GL128P",
		ManufacturerID: "01",
		DeviceID:       "227E",
		Capacity:       16 * MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 30,
		},
	},
	"MX29LV400BT": {
		Name:           "Macronix MX29LV400BT",
		ManufacturerID: "C2",
		DeviceID:       "B9",
		Capacity:       512 * KB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 0,
			tWP: 35,
			tDS: 30,
		},
	},
	"ES29LV800DB": {
		Name:           "Excel Semiconductor ES29LV800DB",
		ManufacturerID: "4A",
		DeviceID:       "5B",
		Capacity:       MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 0,
			tWP: 35,
			tDS: 30,
		},
	},
}
