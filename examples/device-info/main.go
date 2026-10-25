package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/temamagic/pix-backpack-go/ble"
)

func run() (err error) {
	device := flag.String("device", "", "advertised name or peripheral address")
	flag.Parse()
	conn, err := ble.Open(ble.Options{Device: *device})
	if err != nil {
		return err
	}
	defer func() {
		if e := conn.Close(); err == nil {
			err = e
		}
	}()
	fmt.Printf("%+v\n", conn.Info)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
