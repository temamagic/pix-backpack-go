// Display bilingual text using a built-in or user-provided JSON bitmap font.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/temamagic/pix-backpack-go/ble"
	"github.com/temamagic/pix-backpack-go/pix"
)

func run() (err error) {
	device := flag.String("device", "", "advertised name or peripheral address")
	name := flag.String("font", "mono", "mono, mono-bold or demo")
	file := flag.String("font-file", "", "custom JSON font; overrides -font")
	text := flag.String("text", "Hello! Привет! Ёё 123", "text to scroll")
	flag.Parse()
	var font pix.Font
	if *file != "" {
		font, err = pix.LoadFont(*file)
	} else {
		font, err = pix.BuiltinFont(*name)
	}
	if err != nil {
		return err
	}
	conn, err := ble.Open(ble.Options{Device: *device})
	if err != nil {
		return err
	}
	defer func() {
		if e := conn.Close(); err == nil {
			err = e
		}
	}()
	return conn.Client.ScrollText(*text, font, pix.RGB{G: 255}, 80, conn.Info.Width, conn.Info.Height)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
