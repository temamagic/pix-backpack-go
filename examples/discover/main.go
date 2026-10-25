package main

import (
	"fmt"
	"os"

	"github.com/temamagic/pix-backpack-go/ble"
)

func main() {
	devices, err := ble.Scan(ble.Options{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, d := range devices {
		fmt.Printf("%s %s RSSI=%d\n", d.Address, d.Name, d.RSSI)
	}
}
