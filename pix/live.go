package pix

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"
)

// Live owns frame 0 and a fixed palette. Use one rendering session per Client;
// starting a widget, uploading or changing the palette invalidates that session.
type Live struct {
	mu            sync.Mutex
	client        *Client
	width, height int
	palette       []RGB
	previous      []byte
	uncertain     bool
}

// StartLive initializes frame zero with a fixed palette and returns its update session.
func (c *Client) StartLive(width, height int, palette []RGB, initial []byte) (*Live, error) {
	if width < 1 || height < 1 || width > 255 || height > 255 || width*height > 506 || len(initial) != width*height || len(palette) < 1 || len(palette) > 256 {
		return nil, fmt.Errorf("invalid live canvas dimensions or palette")
	}
	for _, index := range initial {
		if int(index) >= len(palette) {
			return nil, fmt.Errorf("index %d outside palette", index)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.command(7, 0, 0, 0); err != nil {
		return nil, err
	}
	if err := c.command(2, 0); err != nil {
		return nil, err
	}
	if err := c.setPalette(0, palette); err != nil {
		return nil, err
	}
	if err := c.setFrame(0, 0, initial); err != nil {
		return nil, err
	}
	if err := c.command(3, 0, 0, 44, 1); err != nil {
		return nil, err
	}
	if err := c.command(4, 0); err != nil {
		return nil, err
	}
	if err := c.command(2, 1); err != nil {
		return nil, err
	}
	return &Live{client: c, width: width, height: height, palette: append([]RGB(nil), palette...), previous: bytes.Clone(initial)}, nil
}

// Update sends only the span from the first to last changed pixel. With a fixed
// palette this is one GATT write (7 bytes for one pixel, 326 for a whole 16x20
// screen), with no clearing, frame-count reset or saving to flash.
// It returns the number of RPC bytes sent; an identical frame sends zero.
func (l *Live) Update(indices []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(indices) != l.width*l.height {
		return 0, fmt.Errorf("expected %d indices", l.width*l.height)
	}
	for _, i := range indices {
		if int(i) >= len(l.palette) {
			return 0, fmt.Errorf("index %d outside palette", i)
		}
	}
	first, last := 0, len(indices)
	if !l.uncertain {
		for first < last && indices[first] == l.previous[first] {
			first++
		}
		for last > first && indices[last-1] == l.previous[last-1] {
			last--
		}
	}
	if first == last {
		return 0, nil
	}
	if err := l.client.SetFrame(0, first, indices[first:last]); err != nil {
		l.uncertain = true
		return 0, err
	}
	l.previous = append(l.previous[:0], indices...)
	l.uncertain = false
	return 6 + last - first, nil
}

// UpdateRGB maps exact colors into the live palette and updates the changed range.
func (l *Live) UpdateRGB(pixels []RGB) (int, error) {
	if len(pixels) != l.width*l.height {
		return 0, fmt.Errorf("pixel count mismatch")
	}
	lookup := map[RGB]byte{}
	for i, c := range l.palette {
		if _, ok := lookup[c]; !ok {
			lookup[c] = byte(i)
		}
	}
	indices := make([]byte, len(pixels))
	for i, c := range pixels {
		v, ok := lookup[c]
		if !ok {
			return 0, fmt.Errorf("colour %+v not in live palette", c)
		}
		indices[i] = v
	}
	return l.Update(indices)
}

// StreamStats counts host frames, RPC bytes and elapsed playback time excluding setup.
type StreamStats struct {
	Frames, Bytes int
	Elapsed       time.Duration
}

// Stream plays frames from the host using a shared fixed palette. loops=0 means
// until ctx is cancelled. Deadlines pace frames; slow BLE extends playback,
// without concurrent writes or an unbounded queue. Elapsed excludes setup.
func (c *Client) Stream(ctx context.Context, a Animation, loops int) (stats StreamStats, err error) {
	if loops < 0 {
		return stats, fmt.Errorf("loops must be >=0")
	}
	if err = ctx.Err(); err != nil {
		return stats, err
	}
	// Preparing the whole sequence ensures stable palette indices. Host streams
	// can contain more frames than firmware memory: prepare in validated chunks.
	p, err := prepareStream(a)
	if err != nil {
		return stats, err
	}
	live, err := c.StartLive(a.Width, a.Height, p.Palette, p.Frames[0])
	if err != nil {
		return stats, err
	}
	started := time.Now()
	defer func() { stats.Elapsed = time.Since(started) }()
	order := playbackOrder(len(a.Frames), a.Direction)
	if a.Direction >= 2 {
		loops = 1
	}
	for loop := 0; loops == 0 || loop < loops; loop++ {
		for _, i := range order {
			if err = ctx.Err(); err != nil {
				return stats, err
			}
			tick := time.Now()
			n, e := live.Update(p.Frames[i])
			if e != nil {
				return stats, e
			}
			stats.Bytes += n
			stats.Frames++
			wait := time.Until(tick.Add(time.Duration(a.Frames[i].Duration) * time.Millisecond))
			timer := time.NewTimer(max(0, wait))
			select {
			case <-ctx.Done():
				timer.Stop()
				return stats, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return stats, nil
}
func prepareStream(a Animation) (Prepared, error) {
	if len(a.Frames) < 1 || len(a.Frames) > 10000 {
		return Prepared{}, fmt.Errorf("stream needs 1..10000 frames")
	}
	// Prepare has a device packet limit on total durations. Build a single
	// flattened RGB frame sequence's palette without that irrelevant limit.
	p := Prepared{Width: a.Width, Height: a.Height, Direction: a.Direction}
	lookup := map[RGB]byte{}
	for _, f := range a.Frames {
		small := a
		small.Frames = []Frame{f}
		if _, err := Prepare(small); err != nil {
			return Prepared{}, err
		}
		indices := make([]byte, len(f.Pixels))
		for j, col := range f.Pixels {
			v, ok := lookup[col]
			if !ok {
				if len(p.Palette) == 256 {
					return Prepared{}, fmt.Errorf("stream palette exceeds 256 colours")
				}
				v = byte(len(p.Palette))
				lookup[col] = v
				p.Palette = append(p.Palette, col)
			}
			indices[j] = v
		}
		p.Frames = append(p.Frames, indices)
	}
	return p, nil
}
func playbackOrder(n int, direction byte) []int {
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	if direction == 1 || direction == 3 {
		for i := n - 2; i > 0; i-- {
			order = append(order, i)
		}
		if direction == 3 && n > 1 {
			order = append(order, 0)
		}
	}
	return order
}
