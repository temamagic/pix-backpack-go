// Package ble connects PIX protocol clients to desktop Bluetooth LE.
// Use one scan/connection session at a time. Characteristic access, including
// reads and brightness changes, must not overlap uploads on one connection.
package ble

import "errors"

// ErrUnsupportedPlatform indicates that no desktop backend is built.
var ErrUnsupportedPlatform = errors.New("PIX BLE requires macOS with CGO, Linux with BlueZ, or Windows")
