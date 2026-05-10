# STM23 NOR Programmer (GUI)

A modern, cross-platform graphical interface for reading, writing, and verifying NOR Flash memory chips. Built with [Go](https://go.dev/) and the [Fyne](https://fyne.io/) UI framework.

> ⚠️ **Important:** This repository contains the **GUI/Frontend only**. It requires compatible hardware programmer and firmware that communicates with the application via a custom serial protocol.

## 📦 Supported NOR Flash Chips

The application includes built-in profiles for the following chips. Auto-detection occurs after connecting to the programmer.

| Model | Manufacturer | Capacity | Device ID(s) |
| :--- | :--- | :--- | :--- |
| `S29AL016M` | Spansion | 2 MB | `0x2249` |
| `S29GL064N` | Spansion | 8 MB | `0x227E`, `0x2210`, `0x2200` |
| `S29GL128P` | Spansion | 16 MB | `0x227E`, `0x2221`, `0x2201` |
| `MX29LV400BT` | Macronix | 512 KB | `0x22B9` |
| `ES29LV800DB` | Excel Semiconductor | 1 MB | `0x005B` |
| `TV005700002` | Toshiba | 16 MB | `0x0003` |

*Support for additional chips can be added by extending the internal `NOR_Flash` configuration maps.*

## 🚀 Quick Start

### Run from Source
```bash
git clone https://github.com/Wassermann1/stm23-nor-programmer.git
cd stm23-nor-programmer
go mod tidy
go run .
```

### Pre-built Releases
Download the latest stable build for your OS from the [Releases](https://github.com/Wassermann1/stm23-nor-programmer/releases) page.

## 📖 Usage Guide

1. **Connect Hardware:** Plug the programmer into a USB port and connect the target NOR Flash chip according to the pinout diagram.
2. **Select Port:** Open the application, choose the correct COM/Serial port from the dropdown, and click **Connect Programmer**.
3. **Identify Chip:** Click **Read ID**. The application will automatically query the chip, match its signature, and display the model, capacity, and icon.
4. **Memory Operations:**
   - 📥 **Read Dump:** Reads the entire chip into memory and displays it in a live HEX table.
   - 📤 **Write Dump:** Writes a loaded `.bin` file to the Flash memory chunk by chunk.
   - 🔍 **Verify Dump:** Byte-by-byte compares the in-memory firmware with the chip contents. Enabled after a successful write.
   - 🗑️ **Erase Chip:** Performs a full chip erase operation.
5. **File Management:** Use the `File` menu to open existing `.bin` dumps or save the current memory buffer to disk.

## 🛠️ Build & Development

**Requirements:** Go `1.23+`

For proper resource bundling and metadata embedding, use the official Fyne CLI:

```bash
# Install Fyne CLI
go install fyne.io/tools/cmd/fyne@latest

# Build for the current OS
fyne package -os $(go env GOOS) -release -out stm23-nor-programmer
```

Cross-compilation for Windows and Linux is fully automated via GitHub Actions (see `.github/workflows/release.yml`). Simply push a version tag (`v0.1.0`) to trigger a release build.

## ⚠️ Important Notes

- **Communication Protocol:** The GUI expects a custom serial command set (`read_full`, `WRITE_BIN`, `chip_erase`, `RESET_FLASH`, `START_DUMP`/`END_DUMP` markers, etc.). Ensure your programmer firmware implements this exact protocol.
- **Safety First:** Always verify chip orientation, pinout, and voltage levels before applying power. Incorrect connections can permanently damage the Flash chip or programmer.
- **Disclaimer:** This software is provided "as is". The authors are not responsible for hardware damage, data loss, or bricked devices resulting from its use.

## License

This project is licensed under the terms of the [MIT License](LICENSE).