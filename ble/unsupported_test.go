//go:build (!darwin && !linux && !windows) || (darwin && !cgo)

package ble

import (
	"errors"
	"testing"
)

func TestUnsupported(t *testing.T) {
	_, err := Open(Options{})
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatal(err)
	}
}
