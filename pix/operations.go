package pix

import (
	"encoding/binary"
	"fmt"
)

// Opcode identifies a PIX RPC operation.
type Opcode byte

// RPC opcodes observed in the original applications.
const (
	OpSetFrame Opcode = iota
	OpSetPalette
	OpSetFrameCount
	OpSetDurations
	OpSetDirection
	OpSave
	OpPrintMemory
	OpSetConfig
	OpInput
	OpRestart
)

// Mode identifies a firmware rendering mode.
type Mode byte

// Rendering modes observed in the original applications.
const (
	ModeNone         Mode = 0
	ModeAnimation    Mode = 1
	ModeScrolling    Mode = 2
	ModeClock        Mode = 6
	ModeBike         Mode = 7
	ModeCountdown    Mode = 8
	ModeStopwatch    Mode = 9
	ModePixBlocks    Mode = 16
	ModePixelBreaker Mode = 17
	ModeCrawler      Mode = 18
)

// Command exposes the recovered RPC for firmware features without a typed helper.
// It serializes with other Client operations. Payload excludes call ID/opcode.
func (c *Client) Command(op Opcode, payload []byte) error {
	if len(payload) > 510 {
		return fmt.Errorf("RPC payload exceeds 510 bytes")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.command(byte(op), payload...)
}
func (c *Client) setConfig(offset uint16, config []byte) error {
	p := binary.LittleEndian.AppendUint16(nil, offset)
	return c.command(byte(OpSetConfig), append(p, config...)...)
}

// SetConfig writes bytes at a configuration offset. Nonzero offsets are byte
// offsets in iOS _setRenderMode; this does not assume a larger firmware buffer.
func (c *Client) SetConfig(offset uint16, config []byte) error {
	if len(config) < 1 || len(config) > 508 {
		return fmt.Errorf("config must contain 1..508 bytes")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setConfig(offset, config)
}
func (c *Client) setPalette(start int, colors []RGB) error {
	for len(colors) > 0 {
		n := min(160, len(colors))
		p := []byte{byte(start)}
		for _, rgb := range colors[:n] {
			p = append(p, rgb.R, rgb.G, rgb.B)
		}
		if err := c.command(1, p...); err != nil {
			return err
		}
		start += n
		colors = colors[n:]
	}
	return nil
}

// SetPalette can recolour an active frame without uploading its indices again.
func (c *Client) SetPalette(start int, colors []RGB) error {
	if start < 0 || start > 255 || len(colors) < 1 || len(colors) > 256 || start+len(colors) > 256 {
		return fmt.Errorf("palette must fit indices 0..255")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setPalette(start, colors)
}
func (c *Client) setFrame(index, offset int, indices []byte) error {
	p := binary.LittleEndian.AppendUint16(nil, uint16(index))
	p = binary.LittleEndian.AppendUint16(p, uint16(offset))
	return c.command(0, append(p, indices...)...)
}

// SetFrame updates palette indices in frame memory. Offset counts pixels, as
// used by iOS useLivePreview (row*displayWidth) and FrameTestScreen (one pixel).
func (c *Client) SetFrame(index, offset int, indices []byte) error {
	if index < 0 || index > 65535 || offset < 0 || offset > 65535 || len(indices) < 1 || len(indices) > 506 || offset+len(indices) > 65536 {
		return fmt.Errorf("invalid frame index, offset or length")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.setFrame(index, offset, indices)
}

// SetFrameCount retains the ONE-byte format used by both supplied apps.
// The device's reported memory capacity (2048) is not proof of a 16-bit opcode.
func (c *Client) SetFrameCount(n int) error {
	if n < 0 || n > 255 {
		return fmt.Errorf("frame count must fit the recovered uint8 format (0..255)")
	}
	return c.Command(OpSetFrameCount, []byte{byte(n)})
}

// SetDurations writes 1..254 frame durations in milliseconds at offset zero.
func (c *Client) SetDurations(durations []int) error {
	if len(durations) < 1 || len(durations) > 254 {
		return fmt.Errorf("1..254 durations required")
	}
	p := []byte{0, 0}
	for _, d := range durations {
		if d < 1 || d > 65535 {
			return fmt.Errorf("duration must be 1..65535 ms")
		}
		p = binary.LittleEndian.AppendUint16(p, uint16(d))
	}
	return c.Command(OpSetDurations, p)
}

// SetDirection sets normal, alternating, normal-stop or alternating-stop playback (0..3).
func (c *Client) SetDirection(direction byte) error {
	if direction > 3 {
		return fmt.Errorf("invalid direction")
	}
	return c.Command(OpSetDirection, []byte{direction})
}

// Save requests flash persistence; persistence after power cycling is unverified.
func (c *Client) Save() error { return c.Command(OpSave, nil) }

// Restart reboots the device; callers must establish a new BLE connection.
func (c *Client) Restart() error { return c.Command(OpRestart, nil) }

// Input uses a separate 1..255 counter, independent of RPC call IDs.
func (c *Client) Input(button byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inputID == 255 {
		c.inputID = 0
	}
	c.inputID++
	return c.command(8, c.inputID, button)
}
