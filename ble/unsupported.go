//go:build (!darwin && !linux && !windows) || (darwin && !cgo)

package ble

import (
	"io"
	"time"

	"github.com/temamagic/pix-backpack-go/pix"
)

// Options configures discovery and optional output.
type Options struct {
	Device     string
	Timeout    time.Duration
	Log, Trace io.Writer
}

// Device describes a discovered peripheral.
type Device struct {
	Address, Name string
	RSSI          int16
}

// Info contains information read from the peripheral.
type Info struct {
	Address, Name, Service, Firmware, Hardware          string
	Width, Height, MaxFrames, Brightness, BrightnessRaw int
	MaxFramesRaw                                        string
}

// Connection is unavailable on this platform.
type Connection struct {
	Client *pix.Client
	Info   Info
}

// Scan reports an unavailable BLE backend.
func Scan(Options) ([]Device, error) { return nil, ErrUnsupportedPlatform }

// Open reports an unavailable BLE backend.
func Open(Options) (*Connection, error) { return nil, ErrUnsupportedPlatform }

// Read reports an unavailable BLE backend.
func (*Connection) Read(uint16) ([]byte, error) { return nil, ErrUnsupportedPlatform }

// SetBrightness reports an unavailable BLE backend.
func (*Connection) SetBrightness(int) error { return ErrUnsupportedPlatform }

// SetDemo reports an unavailable BLE backend.
func (*Connection) SetDemo(bool) error { return ErrUnsupportedPlatform }

// SubscribeRPC reports an unavailable BLE backend.
func (*Connection) SubscribeRPC(func([]byte)) error { return ErrUnsupportedPlatform }

// Close reports an unavailable BLE backend.
func (*Connection) Close() error { return ErrUnsupportedPlatform }
