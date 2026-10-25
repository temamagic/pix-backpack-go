# Compatibility

| Platform | Build/backend | Hardware evidence |
|---|---|---|
| macOS Apple Silicon | CoreBluetooth, CGO, Xcode command line tools | Prior notes: Original V1 16×20, firmware 2.5.1, hardware 3.2 |
| macOS Intel | Same backend; requires an Intel native toolchain | Not tested on Intel hardware |
| Linux | TinyGo BlueZ over system D-Bus; Bluetooth adapter and permissions required | Experimental; no hardware test |
| Windows | TinyGo WinRT desktop backend | Experimental; no hardware test |
| Other OS / macOS CGO disabled | `pix` usable; `ble` returns ErrUnsupportedPlatform | No BLE support |

Compilation is not a claim of radio/GATT interoperability. In particular, long
write-with-response behavior must be tested on each OS/adapter; a 20-byte write
split cannot substitute for one PIX packet. No specific USB adapter is certified.

Original legacy and Mini UUIDs were observed in apps. Their device behavior,
16×16 text, notifications and firmware variants are unverified. API upload limits:
506 pixels per frame write, 256 colors, 254 durations, one-byte frame count.
Reported device capacity 2048 is not proof that a wider count command exists.

This preparation runs offline tests only. Previous device observations are in
[hardware.md](hardware.md), with identifying peripheral data removed.
