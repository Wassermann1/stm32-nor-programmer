# STM23 NOR Programmer — Serial Protocol Specification

This document describes the text-based command protocol used for communication between the GUI application and the STM32-based programmer firmware over USB CDC (Virtual COM Port).

---

## 📡 General Protocol Rules

| Aspect | Specification |
|--------|--------------|
| **Transport** | USB CDC / UART (115200 baud, 8N1, no flow control) |
| **Command Format** | ASCII text, newline-terminated (`\r\n`) |
| **Response Format** | ASCII text or binary, always ends with `\r\n` for text responses |
| **Hex Values** | Uppercase, zero-padded (`%02X`, `%04X`, `%08X`) |
| **Byte Order** | Little-endian for multi-byte values (LSB first) |
| **Binary Transfers** | Raw bytes, **no** additional framing or escaping |

> ⚠️ **Critical**: After sending/receiving binary data (`WRITE_BIN`, `read_full`), the firmware re-initializes the CDC RX buffer. The host must wait for the final text response before sending the next command.

---

## 📋 Command Reference

### 1. `READ_ID`
Read Flash ID via FSMC parallel interface (faster, hardware-specific). Returns extended ID set.

**Request:**
```
READ_ID\r\n
```

**Response (Success):**
```
MM DDDD DDDD DDDD\r\n
```
| Field | Description | Format |
|-------|-------------|--------|
| `MM` | Manufacturer ID (lower byte) | 2-digit hex |
| `DDDD` (×3) | Device ID registers 1/2/3 | 4-digit hex each |

**Example:**
```
C2 22B9 0000 0000\r\n
```

**Response (Error):**
```
ERR\r\n
```

---

### 2. `READ_FULL <bytes>`
Read raw data from Flash memory starting at address `0x00000000`.

**Request:**
```
READ_FULL <bytes>\r\n
```
| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `<bytes>` | decimal uint32 | Number of bytes to read | `1048576` (1 MiB) |

**Protocol Flow:**
```
[Host] → read_full 131072\r\n
[Device] → START_DUMP\r\n
[Device] → <131072 raw bytes>
[Device] → END_DUMP\r\n
```

**Notes:**
- Data is read via 16-bit FSMC bus; transfers are automatically aligned to even addresses.
- Binary payload is sent **immediately** after `START_DUMP\r\n` with no delimiters.
- Host must read exactly `<bytes>` bytes until `END_DUMP\r\n` marker.

---

### 3. `WRITE_BIN <addr> <len>`
Write a block of raw data to Flash memory.
**Request:**
```
WRITE_BIN <addr> <len>\r\n
```
| Parameter | Type | Description | Constraints |
|-----------|------|-------------|-------------|
| `<addr>` | hex uint32 | Start address in Flash memory | Must be even (16-bit aligned) |
| `<len>` | hex uint32 | Number of bytes to write | `64 < len ≤ 512`, must be even and a multiple of the standard usb paket of 64B|

**Protocol Flow (Handshake):**
```
[Host] → WRITE_BIN 00004000 0200\r\n
[Device] → READY\r\n
[Host] → <512 raw bytes>
[Device] → WRITE_OK\r\n   (or error response)
```

**Possible Device Responses:**
| Response | Meaning |
|----------|---------|
| `READY\r\n` | Ready to receive binary payload |
| `WRITE_OK\r\n` | Block successfully programmed |
| `Error recieving data\r\n` | Command parse error |
| `Error wrong data size\r\n` | `len` out of bounds or odd |
| `Error timeout\r\n` | Binary payload not received in time |
| `Error on writing\r\n` | Flash programming failed |
| `Error on writing TIMEOUT\r\n` | Flash busy timeout during program |

**Critical Host Requirements:**
1. Wait for `READY\r\n` **before** sending binary data.
2. Send **exactly** `<len>` raw bytes (no `\r\n`, no framing).
3. After sending binary, wait for final text response (`WRITE_OK` or error).
4. Do **not** send the next command until the final response is fully received.

---

### 4. `CHIP_ERASE`
Perform a full chip erase operation.

**Request:**
```
CHIP_ERASE\r\n
```

**Response:**
| Response | Meaning |
|----------|---------|
| `Starting chip erase\r\n` | Operation initiated (interim) |
| `ChipErase SUCCESS\r\n` | Erase completed successfully |
| `ChipErase ERROR\r\n` | Erase failed (generic) |
| `ChipErase ERROR - timeout\r\n` | Erase operation timed out |

> ⏱️ **Note**: Erase may take several seconds to minutes. Host should implement a long timeout (≥300 s).

---

### 5. `CONFIG <tAS> <tAH> <tDS> <tBT>`
Update FSMC bus timing parameters. Parameters calculated from selected NOR timings and passed as CPU cycles to STM

**Request:**
```
CONFIG <tAS> <tAH> <tDS> <tBT>\r\n
```
| Parameter | Description | Typical Range |
|-----------|-------------|---------------|
| `<tAS>` | Address Setup Time | 0-15 cycles  |
| `<tAH>` | Address Hold Time  | 0-15 cecles  |
| `<tDS>` | Data Setup Time    | 0-15 cycles  |
| `<tBT>` | Bus Turnaround     | 0–254 cycles |

**Response (Success):**
```
FSMC timing updated\r\n
```

**Response (Error):**
```
FSMC timing update failed\r\n
```

**Response (Usage Hint):**
```
Usage: config <tAS_uibt8> <tAH_uint8> <tDS_uint8> <tBT_uint8>\r\n
```

---

### 6. `RESET_FLASH`
Reset Flash memory to Read mode (sends `0xF0` command).

**Request:**
```
RESET_FLASH\r\n
```

**Response:**
```
OK\r\n
```

---

## 🔄 Typical Operation Sequences

### 🔍 Read Chip ID
```
[Host] → READ_ID\r\n
[Device] → 0xC2 0x22B9 0x0000 0x0000\r\n
```

### 📥 Full Dump (128 KB)
```
[Host] → READ_FULL 131072\r\n
[Device] → START_DUMP\r\n
[Device] → <131072 raw bytes>
[Device] → END_DUMP\r\n
```

### 📤 Write 512-Byte Block
```
[Host] → WRITE_BIN 00001000 0200\r\n
[Device] → READY\r\n
[Host] → <512 raw bytes>
[Device] → WRITE_OK\r\n
```

### 🗑️ Erase Chip
```
[Host] → CHIP_ERASE\r\n
[Device] → Starting chip erase\r\n
... (wait up to 300s) ...
[Device] → ChipErase SUCCESS\r\n
```