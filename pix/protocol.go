// Package pix implements the PIX backpack protocol recovered from Android 1.40.
package pix

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

// Suffixes lists the observed Original legacy, Original V1 and Mini UUID suffixes.
var Suffixes = []string{"-e984-11e7-b78e-ffd6fcc3450f", "-e984-11e7-b78e-ffd6fcc34510", "-9552-4325-8021-a85f8136a5c4"}

// UUID returns a PIX characteristic or service UUID (e.g. UUID(0x100, suffix)).
func UUID(id uint16, suffix string) string { return fmt.Sprintf("0000%04x%s", id, suffix) }

// RGB is an exact red, green and blue color with eight bits per channel.
type RGB struct{ R, G, B byte }

// Frame contains row-major pixels and its display duration in milliseconds.
type Frame struct {
	Pixels   []RGB // top-to-bottom rows, left-to-right columns
	Duration int   // milliseconds, 1..65535
}

// Animation is a sequence of equal-size RGB frames and a playback direction.
type Animation struct {
	Width, Height int
	Frames        []Frame
	Direction     byte // 0=normal, 1=alternate, 2=normal-stop, 3=alternate-stop
}

// Prepared holds validated palette indices and little-endian frame durations.
type Prepared struct {
	Width, Height int
	Palette       []RGB
	Frames        [][]byte
	Durations     []byte
	Direction     byte
}

// Prepare validates the entire animation before any writes reach the backpack.
// Colours are exact; more than 256 unique colours must be quantized by the caller.
func Prepare(a Animation) (Prepared, error) {
	p := Prepared{Width: a.Width, Height: a.Height, Direction: a.Direction}
	if a.Width < 1 || a.Width > 255 || a.Height < 1 || a.Height > 255 || a.Width*a.Height > 506 || len(a.Frames) < 1 || len(a.Frames) > 254 || a.Direction > 3 {
		return p, fmt.Errorf("invalid dimensions, direction or frame count (1..254); each GATT value must fit 512 bytes")
	}
	lookup := map[RGB]byte{}
	for i, frame := range a.Frames {
		if len(frame.Pixels) != a.Width*a.Height || frame.Duration < 1 || frame.Duration > 65535 {
			return Prepared{}, fmt.Errorf("frame %d: expected %d pixels and duration 1..65535 ms", i, a.Width*a.Height)
		}
		indices := make([]byte, len(frame.Pixels))
		for j, color := range frame.Pixels {
			index, ok := lookup[color]
			if !ok {
				if len(p.Palette) == 256 {
					return Prepared{}, fmt.Errorf("more than 256 colours across animation; quantize first")
				}
				index = byte(len(p.Palette))
				lookup[color] = index
				p.Palette = append(p.Palette, color)
			}
			indices[j] = index
		}
		p.Frames = append(p.Frames, indices)
		p.Durations = binary.LittleEndian.AppendUint16(p.Durations, uint16(frame.Duration))
	}
	return p, nil
}

// Writer must perform one complete GATT write WITH response, including long
// writes when needed. Arbitrarily slicing packets changes protocol semantics.
type Writer interface{ Write([]byte) (int, error) }

// Client serializes RPC operations on a writer and maintains session counters.
type Client struct {
	mu      sync.Mutex
	rpc     Writer
	callID  byte
	inputID byte
}

// NewClient creates a session for complete GATT writes through rpc.
func NewClient(rpc Writer) *Client { return &Client{rpc: rpc} }
func (c *Client) command(op byte, payload ...byte) error {
	if c.rpc == nil {
		return fmt.Errorf("PIX transport: %w", io.ErrClosedPipe)
	}
	packet := append([]byte{c.callID, op}, payload...)
	c.callID++
	n, err := c.rpc.Write(packet)
	if err != nil {
		return fmt.Errorf("opcode %d (%d bytes): %w", op, len(packet), err)
	}
	if n != len(packet) {
		return io.ErrShortWrite
	}
	return nil
}

// Upload follows module 739's order. Saving to flash is explicit, as opposed to
// the Android application's default. A failed upload can leave the display blank.
func (c *Client) Upload(a Animation, maxFrames int, save bool) error {
	return c.UploadContext(context.Background(), a, maxFrames, save)
}

// UploadContext checks cancellation before each complete GATT write. A write
// already in progress remains subject to the transport timeout. It serializes
// with other Client calls; cancellation does not interrupt mutex acquisition.
func (c *Client) UploadContext(ctx context.Context, a Animation, maxFrames int, save bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := Prepare(a)
	if err != nil {
		return err
	}
	if maxFrames < 1 || len(p.Frames) > maxFrames {
		return fmt.Errorf("%d frames exceed device limit %d", len(p.Frames), maxFrames)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	command := func(op byte, payload ...byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return c.command(op, payload...)
	}
	if err = command(7, 0, 0, 0); err != nil {
		return err
	}
	if err = command(2, 0); err != nil {
		return err
	}
	for start := 0; start < len(p.Palette); start += 160 {
		end := min(start+160, len(p.Palette))
		payload := []byte{byte(start)}
		for _, rgb := range p.Palette[start:end] {
			payload = append(payload, rgb.R, rgb.G, rgb.B)
		}
		if err = command(1, payload...); err != nil {
			return err
		}
	}
	for i, frame := range p.Frames {
		payload := binary.LittleEndian.AppendUint16(nil, uint16(i))
		payload = append(payload, 0, 0)
		payload = append(payload, frame...)
		if err = command(0, payload...); err != nil {
			return err
		}
	}
	if err = command(3, append([]byte{0, 0}, p.Durations...)...); err != nil {
		return err
	}
	if err = command(4, p.Direction); err != nil {
		return err
	}
	if err = command(2, byte(len(p.Frames))); err != nil {
		return err
	}
	if save {
		return command(5)
	}
	return nil
}

// Clear sets mode NONE without saving to flash.
func (c *Client) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.command(7, 0, 0, 0)
}

// Brightness encodes a percentage (0..100) as a raw brightness byte.
func Brightness(percent int) ([]byte, error) {
	if percent < 0 || percent > 100 {
		return nil, fmt.Errorf("brightness must be 0..100")
	}
	return []byte{byte(255 * percent / 100)}, nil
}
