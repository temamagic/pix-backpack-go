package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/temamagic/pix-backpack-go/ble"
	"github.com/temamagic/pix-backpack-go/pix"
)

func run() (err error) {
	device := flag.String("device", "", "advertised name or peripheral address")
	file := flag.String("file", "", "exact-size PNG/JPEG/GIF/pixpack")
	flag.Parse()
	if *file == "" {
		return fmt.Errorf("-file is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	conn, err := ble.Open(ble.Options{Device: *device})
	if err != nil {
		return err
	}
	defer func() {
		if e := conn.Close(); err == nil {
			err = e
		}
	}()
	a, err := pix.Load(*file, conn.Info.Width, conn.Info.Height, 0)
	if err != nil {
		return err
	}
	return conn.Client.UploadContext(ctx, a, conn.Info.MaxFrames, false)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
