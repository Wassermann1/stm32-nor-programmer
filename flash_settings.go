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
	DeviceID2      string //For spansion memory
	DeviceID3      string //For spansion memory
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

const (
	SpansionManID string = "01"
	SpansionDevID string = "227E"
)

var SpansionFlashSpecs = map[string]NOR_Flash{
	"S29GL128P": {
		Name:           "Spansion S29GL128P",
		ManufacturerID: SpansionManID,
		DeviceID:       SpansionDevID,
		DeviceID2:      "2221",
		DeviceID3:      "2201",
		Capacity:       16 * MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 35,
		},
	},
	"S29GL064N": {
		Name:           "Spansion S29GL064N",
		ManufacturerID: SpansionManID,
		DeviceID:       SpansionDevID,
		DeviceID2:      "2210",
		DeviceID3:      "2200",
		Capacity:       8 * MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 35,
		},
	},
}

var FlashSpecs = map[string]NOR_Flash{
	"S29AL016M": {
		Name:           "Spansion S29AL016M",
		ManufacturerID: SpansionManID,
		DeviceID:       "2249",
		Capacity:       2 * MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 35,
		},
	},
	"MX29LV400BT": {
		Name:           "Macronix MX29LV400BT",
		ManufacturerID: "C2",
		DeviceID:       "22B9",
		Capacity:       512 * KB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 35,
		},
	},
	"ES29LV800DB": {
		Name:           "Excel Semiconductor ES29LV800DB",
		ManufacturerID: "4A",
		DeviceID:       "5B",
		Capacity:       MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 45,
			tWP: 35,
			tDS: 45,
		},
	},
	"TV005700002": {
		Name:           "Toshiba TV005700002",
		ManufacturerID: "98",
		DeviceID:       "0003",
		Capacity:       16 * MB,
		Specs: NOR_TimingSpec{
			tAS: 0,
			tAH: 30,
			tWP: 30,
			tDS: 30,
		},
	},
}
