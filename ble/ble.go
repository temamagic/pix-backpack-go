//go:build (darwin && cgo) || linux || windows

// Package ble connects the protocol Client to desktop Bluetooth LE (CoreBluetooth
// on macOS, BlueZ on Linux). Use one connection/rendering session at a time.
package ble

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/temamagic/pix-backpack-go/pix"
	"tinygo.org/x/bluetooth"
)

// Options configures discovery, device selection and optional log/trace output.
type Options struct {
	Device  string
	Timeout time.Duration
	Log     io.Writer
	Trace   io.Writer
}

// Device describes a discovered Bluetooth peripheral.
type Device struct {
	Address, Name string
	RSSI          int16
	result        bluetooth.ScanResult
}

// Info contains device characteristics and optional firmware/hardware strings.
type Info struct {
	Address, Name, Service, Firmware, Hardware string
	Width, Height, MaxFrames, Brightness       int
	BrightnessRaw                              int
	MaxFramesRaw                               string
}

// Connection owns a BLE device and its protocol client; close it after use.
type Connection struct {
	Client    *pix.Client
	Info      Info
	device    bluetooth.Device
	chars     map[uint16]bluetooth.DeviceCharacteristic
	rpc       bluetooth.DeviceCharacteristic
	closeOnce sync.Once
	closeErr  error
}
type tracingWriter struct {
	target pix.Writer
	out    io.Writer
}

func (w tracingWriter) Write(p []byte) (int, error) {
	if _, err := fmt.Fprintf(w.out, "RPC %s\n", hex.EncodeToString(p)); err != nil {
		return 0, err
	}
	return w.target.Write(p)
}

// Scan discovers matching peripherals for Options.Timeout (default three seconds).
func Scan(o Options) ([]Device, error) {
	a := bluetooth.DefaultAdapter
	if err := a.Enable(); err != nil {
		return nil, fmt.Errorf("enable Bluetooth: %w", err)
	}
	return scan(a, o)
}
func scan(a *bluetooth.Adapter, o Options) ([]Device, error) {
	if o.Timeout <= 0 {
		o.Timeout = 3 * time.Second
	}
	if o.Timeout < time.Second {
		return nil, fmt.Errorf("scan timeout must be >=1s")
	}
	var found []Device
	seen := map[string]bool{}
	timerDone := make(chan struct{})
	timer := time.AfterFunc(o.Timeout, func() { defer close(timerDone); _ = a.StopScan() })
	defer func() {
		if !timer.Stop() {
			<-timerDone
		}
	}()
	err := a.Scan(func(_ *bluetooth.Adapter, r bluetooth.ScanResult) {
		matches := strings.Contains(strings.ToLower(r.LocalName()), "pix")
		for _, suffix := range pix.Suffixes {
			u, err := bluetooth.ParseUUID(pix.UUID(0x100, suffix))
			if err == nil && r.HasServiceUUID(u) {
				matches = true
			}
		}
		if o.Device != "" {
			matches = strings.EqualFold(r.Address.String(), o.Device) || strings.EqualFold(r.LocalName(), o.Device)
		}
		if matches && !seen[r.Address.String()] {
			seen[r.Address.String()] = true
			d := Device{r.Address.String(), r.LocalName(), r.RSSI, r}
			found = append(found, d)
			if o.Log != nil {
				_, _ = fmt.Fprintf(o.Log, "%s  %s  RSSI=%d\n", d.Address, d.Name, d.RSSI)
			}
		}
	})
	return found, err
}

// Open connects to exactly one matching device and reads its capabilities.
func Open(o Options) (*Connection, error) {
	a := bluetooth.DefaultAdapter
	if err := a.Enable(); err != nil {
		return nil, fmt.Errorf("enable Bluetooth: %w", err)
	}
	devices, err := scan(a, o)
	if err != nil {
		return nil, err
	}
	if len(devices) != 1 {
		return nil, fmt.Errorf("found %d matching devices; use Device to select one", len(devices))
	}
	d := devices[0]
	dev, err := a.Connect(d.result.Address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, err
	}
	c := &Connection{device: dev, chars: map[uint16]bluetooth.DeviceCharacteristic{}, Info: Info{Address: d.Address, Name: d.Name}}
	ok := false
	defer func() {
		if !ok {
			_ = dev.Disconnect()
		}
	}()
	services, err := dev.DiscoverServices(nil)
	if err != nil {
		return nil, err
	}
	var suffix string
	for _, service := range services {
		for _, s := range pix.Suffixes {
			if service.UUID().String() == pix.UUID(0x100, s) {
				suffix = s
				c.Info.Service = service.UUID().String()
				chars, e := service.DiscoverCharacteristics(nil)
				if e != nil {
					return nil, e
				}
				for _, ch := range chars {
					for _, id := range []uint16{0x101, 0x103, 0x104, 0x105, 0x106, 0x110} {
						if ch.UUID().String() == pix.UUID(id, s) {
							c.chars[id] = ch
						}
					}
				}
			}
		}
	}
	if suffix == "" {
		return nil, fmt.Errorf("no known PIX GATT service")
	}
	rpc, exists := c.chars[0x101]
	if !exists {
		return nil, fmt.Errorf("missing RPC characteristic")
	}
	c.rpc = rpc
	var writer pix.Writer = rpc
	if o.Trace != nil {
		writer = tracingWriter{rpc, o.Trace}
	}
	c.Client = pix.NewClient(writer)
	for _, item := range []struct {
		id  uint16
		dst *int
	}{{0x103, &c.Info.Width}, {0x104, &c.Info.Height}, {0x105, &c.Info.MaxFrames}, {0x106, &c.Info.BrightnessRaw}} {
		data, e := c.Read(item.id)
		if e != nil {
			return nil, fmt.Errorf("read characteristic %04x: %w", item.id, e)
		}
		n, e := number(data)
		if e != nil {
			return nil, e
		}
		*item.dst = n
		if item.id == 0x105 {
			c.Info.MaxFramesRaw = hex.EncodeToString(data)
		}
	}
	if c.Info.Width < 1 || c.Info.Width > 255 || c.Info.Height < 1 || c.Info.Height > 255 || c.Info.Width*c.Info.Height > 506 {
		return nil, fmt.Errorf("unsupported display dimensions %dx%d", c.Info.Width, c.Info.Height)
	}
	c.Info.Brightness = c.Info.BrightnessRaw * 100 / 255
	for _, service := range services {
		if service.UUID() != bluetooth.New16BitUUID(0x180a) {
			continue
		}
		chars, e := service.DiscoverCharacteristics(nil)
		if e != nil {
			continue
		}
		for _, ch := range chars {
			var dest *string
			switch ch.UUID() {
			case bluetooth.New16BitUUID(0x2a26):
				dest = &c.Info.Firmware
			case bluetooth.New16BitUUID(0x2a27):
				dest = &c.Info.Hardware
			default:
				continue
			}
			buf := make([]byte, 512)
			n, e := ch.Read(buf)
			if e == nil && n >= 0 && n <= len(buf) {
				*dest = strings.TrimRight(string(buf[:n]), "\x00")
			}
		}
	}
	ok = true
	return c, nil
}

// Read reads a known characteristic by its 16-bit PIX identifier.
func (c *Connection) Read(id uint16) ([]byte, error) {
	ch, ok := c.chars[id]
	if !ok {
		return nil, fmt.Errorf("missing characteristic %04x", id)
	}
	buf := make([]byte, 512)
	n, err := ch.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 0 || n > len(buf) {
		return nil, fmt.Errorf("read buffer overflow")
	}
	return buf[:n], nil
}
func number(b []byte) (int, error) {
	switch len(b) {
	case 1:
		return int(b[0]), nil
	case 2:
		return int(binary.LittleEndian.Uint16(b)), nil
	case 4:
		return int(binary.LittleEndian.Uint32(b)), nil
	}
	return 0, fmt.Errorf("unexpected integer length %d", len(b))
}

// SetBrightness writes a brightness percentage in the range 0..100.
func (c *Connection) SetBrightness(percent int) error {
	p, err := pix.Brightness(percent)
	if err != nil {
		return err
	}
	return c.write(0x106, p)
}

// SetDemo enables or disables the device demonstration mode.
func (c *Connection) SetDemo(enabled bool) error {
	value := byte(1)
	if enabled {
		value = 0
	}
	return c.write(0x110, []byte{value})
}
func (c *Connection) write(id uint16, p []byte) error {
	ch, ok := c.chars[id]
	if !ok {
		return fmt.Errorf("missing characteristic %04x", id)
	}
	n, err := ch.Write(p)
	if err == nil && n != len(p) {
		return io.ErrShortWrite
	}
	return err
}

// SubscribeRPC exposes raw firmware notifications without assuming their format.
// It can fail on firmware that does not support notifications on characteristic 0101.
func (c *Connection) SubscribeRPC(callback func([]byte)) error {
	if callback == nil {
		return fmt.Errorf("notification callback is required")
	}
	return c.rpc.EnableNotifications(func(p []byte) { callback(append([]byte(nil), p...)) })
}

// Close disconnects the peripheral once and returns the first disconnect error.
func (c *Connection) Close() error {
	c.closeOnce.Do(func() { c.closeErr = c.device.Disconnect() })
	return c.closeErr
}
