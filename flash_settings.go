package main

import (
	"fmt"

	"fyne.io/fyne/v2/data/binding"
)

const HCLK_MHZ = 168
const MARGIN_NS = 12 // ~2 HCLK cycles at 168MHz, covers PCB trace + latch prop delay

// NOR_TimingSpec holds datasheet AC parameters in nanoseconds.
// All values are minimums unless noted.
type NOR_TimingSpec struct {
	// Write path
	TAS  uint16 // address setup before NWE↓ (some datasheets: tAS or tAVS)
	TAH  uint16 // address hold after NWE↑ (tAH)
	TWP  uint16 // NWE pulse width low (tWP)
	TWPH uint16 // NWE high between consecutive writes (tWPH)
	TDS  uint16 // data setup before NWE↑ (tDS)
	TDH  uint16 // data hold after NWE↑ (tDH)
	// Read path
	TRC  uint16 // read cycle time, addr→addr (tRC)
	TACC uint16 // address access time, addr valid → data valid (tACC or tAA)
	TOE  uint16 // OE access time, NOE↓ → data valid (tOE or tCO)
	TOEH uint16 // OE high → data hi-Z, bus release (tOEH or tHZ)
	// Recovery
	TCEPH uint16 // NE high time between accesses (tCEPH or tCE2)
}

// FSMC_TimingRegs holds the actual register values to write.
// These are sent over serial — STM just applies them directly.
type FSMC_TimingRegs struct {
	AddressSetupTime      uint8 // ADDSET [0..15]
	AddressHoldTime       uint8 // ADDHLD [1..15]
	DataSetupTime         uint8 // DATAST [1..255]
	BusTurnAroundDuration uint8 // BUSTURN [0..15]
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
	_  = iota
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

var FlashSpecs = map[string]NOR_Flash{
	"S29GL128P": {
		Name: "Spansion S29GL128P", ManufacturerID: SpansionManID,
		DeviceID: SpansionDevID, DeviceID2: "2221", DeviceID3: "2201",
		Capacity: 16 * MB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 55, TWPH: 20, TDS: 45, TDH: 0,
			TRC: 90, TACC: 90, TOE: 25, TOEH: 0, TCEPH: 20,
		},
	},
	"S29GL064N": {
		Name: "Spansion S29GL064N", ManufacturerID: SpansionManID,
		DeviceID: SpansionDevID, DeviceID2: "2210", DeviceID3: "2200",
		Capacity: 8 * MB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 55, TWPH: 20, TDS: 45, TDH: 0,
			TRC: 90, TACC: 90, TOE: 25, TOEH: 0, TCEPH: 20,
		},
	},
	"S29AL016M": {
		Name: "Spansion S29AL016M", ManufacturerID: SpansionManID,
		DeviceID: "2249",
		Capacity: 2 * MB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 55, TWPH: 20, TDS: 45, TDH: 0,
			TRC: 90, TACC: 90, TOE: 30, TOEH: 0, TCEPH: 20,
		},
	},
	"MX29LV400BT": {
		Name: "Macronix MX29LV400BT", ManufacturerID: "C2",
		DeviceID: "22B9",
		Capacity: 512 * KB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 45, TWPH: 20, TDS: 40, TDH: 0,
			TRC: 70, TACC: 70, TOE: 25, TOEH: 0, TCEPH: 15,
		},
	},
	"ES29LV800DB": {
		Name: "Excel Semiconductor ES29LV800DB", ManufacturerID: "004A",
		DeviceID: "005B",
		Capacity: 1 * MB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 45, TWPH: 20, TDS: 40, TDH: 0,
			TRC: 90, TACC: 90, TOE: 30, TOEH: 0, TCEPH: 20,
		},
	},
	"TV005700002": {
		Name: "Toshiba TV005700002", ManufacturerID: "0098",
		DeviceID: "0003",
		Capacity: 16 * MB,
		Specs: NOR_TimingSpec{
			TAS: 0, TAH: 0, TWP: 30, TWPH: 15, TDS: 30, TDH: 0,
			TRC: 70, TACC: 70, TOE: 20, TOEH: 0, TCEPH: 15,
		},
	},
}

func nsToCycles(ns uint16) uint32 {
	return (uint32(ns)*HCLK_MHZ + 999) / 1000
}

func clamp(v, min, max uint32) uint32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// CalcFSMCTimings converts datasheet ns values → FSMC register values.
// Returns the registers and a human-readable breakdown for logging.
func CalcFSMCTimings(s *NOR_TimingSpec) (FSMC_TimingRegs, string) {
	margin := uint32(MARGIN_NS) * HCLK_MHZ / 1000 // margin in cycles

	// --- ADDHLD ---
	// Must satisfy: tAH, tWPH, tDH (all are "after NWE↑" constraints)
	addhldNS := s.TAH
	if s.TWPH > addhldNS {
		addhldNS = s.TWPH
	}
	if s.TDH > addhldNS {
		addhldNS = s.TDH
	}
	addhld := clamp(nsToCycles(addhldNS)+margin, 1, 15)

	// --- DATAST lower bound ---
	// Must satisfy: tWP (NWE low pulse) and tOE (NOE low pulse)
	// Both signals are low during DATAST window
	datastMinNS := s.TWP
	if s.TOE > datastMinNS {
		datastMinNS = s.TOE
	}
	datast := nsToCycles(datastMinNS) + margin

	// --- ADDSET ---
	// In multiplexed mode: address must be stable for latch capture.
	// tAS is typically 0 on most NOR chips (addr and NE can assert together),
	// but the latch (74HC373) adds ~7ns propagation, so always add that.
	const LATCH_PROP_NS = uint16(8) // 74HC373/573 typical tPD
	addsetNS := s.TAS
	if LATCH_PROP_NS > addsetNS {
		addsetNS = LATCH_PROP_NS
	}
	addset := nsToCycles(addsetNS) + margin

	// --- Verify tRC / tACC: total read window = (ADDSET+1 + DATAST+1) cycles ---
	// Must cover both tRC and tACC
	totalNeededNS := s.TRC
	if s.TACC > totalNeededNS {
		totalNeededNS = s.TACC
	}
	totalNeeded := nsToCycles(totalNeededNS) + margin
	// Grow DATAST if total is insufficient (preserves address phase)
	for (addset + 1 + datast + 1) < totalNeeded {
		datast++
	}

	// --- tDS check (data setup within DATAST window) ---
	// Data must be valid tDS ns before NWE rises.
	// The DATAST window is (datast+1)*tCK ns total.
	// As long as tDS < (datast+1)*tCK this is satisfied — data is on bus from start.
	datastActualNS := uint16((datast + 1) * 1000 / HCLK_MHZ)
	dsWarning := ""
	if s.TDS > 0 && s.TDS >= datastActualNS {
		dsWarning = fmt.Sprintf(" !! tDS=%dns not satisfied by DATAST window=%dns", s.TDS, datastActualNS)
	}

	// --- BUSTURN ---
	// Must cover tOEH (bus release after NOE↑) and tCEPH (NE recovery)
	busturnNS := s.TOEH
	if s.TCEPH > busturnNS {
		busturnNS = s.TCEPH
	}
	busturn := nsToCycles(busturnNS) + margin
	if busturn < 1 {
		busturn = 1
	} // minimum 1 for stability

	// Clamp all to register widths
	addset = clamp(addset, 0, 15)
	addhld = clamp(addhld, 1, 15)
	datast = clamp(datast-1, 0, 254) // register = actual_cycles - 1
	busturn = clamp(busturn, 0, 15)

	regs := FSMC_TimingRegs{
		AddressSetupTime:      uint8(addset),
		AddressHoldTime:       uint8(addhld),
		DataSetupTime:         uint8(datast),
		BusTurnAroundDuration: uint8(busturn),
	}

	tckNS := float32(1000) / HCLK_MHZ
	summary := fmt.Sprintf(
		"ADDSET=%d (%.1fns) ADDHLD=%d (%.1fns) DATAST=%d (%.1fns) BUSTURN=%d (%.1fns) | "+
			"tRC_actual=%.1fns tWP_actual=%.1fns%s",
		regs.AddressSetupTime, float32(regs.AddressSetupTime+1)*tckNS,
		regs.AddressHoldTime, float32(regs.AddressHoldTime)*tckNS,
		regs.DataSetupTime, float32(regs.DataSetupTime+1)*tckNS,
		regs.BusTurnAroundDuration, float32(regs.BusTurnAroundDuration)*tckNS,
		float32(regs.AddressSetupTime+1+regs.DataSetupTime+1)*tckNS,
		float32(regs.DataSetupTime+1)*tckNS,
		dsWarning,
	)
	return regs, summary
}

// LookupFlash finds a flash by the 4 IDs returned from the READ_ID command.
// Returns nil if not found.
func LookupFlash(manufID, devID, devID2, devID3 string) *NOR_Flash {
	for i := range FlashSpecs {
		f := FlashSpecs[i]
		if f.ManufacturerID != manufID || f.DeviceID != devID {
			continue
		}
		// Spansion S29GL family: same ManID+DevID, need DevID2+DevID3 to tell apart
		if manufID == SpansionManID && devID == SpansionDevID {
			if f.DeviceID2 == devID2 && f.DeviceID3 == devID3 {
				return &f
			}
			continue
		}
		// Everyone else: ManID+DevID is sufficient
		return &f
	}
	return nil
}
