// Algorithmic live graphics: each frame is generated from the current time,
// then updates frame 0 in place. Stop with Ctrl+C.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"time"

	"github.com/temamagic/pix-backpack-go/ble"
	"github.com/temamagic/pix-backpack-go/pix"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() (err error) {
	address := flag.String("device", "", "device address from pixctl scan")
	flag.Parse()
	conn, err := ble.Open(ble.Options{Device: *address, Log: os.Stderr})
	if err != nil {
		return err
	}
	defer func() {
		if e := conn.Close(); err == nil {
			err = e
		}
	}()
	w, h := conn.Info.Width, conn.Info.Height
	palette := []pix.RGB{{}, {R: 255, G: 30}, {R: 255, G: 120}, {R: 255, G: 240}, {R: 20, G: 220, B: 40}, {G: 140, B: 255}, {R: 80, G: 40, B: 255}, {R: 220, G: 40, B: 200}}
	live, err := conn.Client.StartLive(w, h, palette, make([]byte, w*h))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			pixels := make([]byte, w*h)
			t := float64(now.UnixMilli()) / 700
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					v := (math.Sin(float64(x)/3+t) + math.Sin(float64(y)/4-t) + 2) / 4
					pixels[y*w+x] = byte(min(7, int(v*8)))
				}
			}
			if _, err = live.Update(pixels); err != nil {
				return err
			}
		}
	}
}
