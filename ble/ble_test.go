//go:build (darwin && cgo) || linux || windows

package ble

import (
	"errors"
	"io"
	"testing"
)

func TestNumber(t *testing.T) {
	for _, tc := range []struct {
		data  []byte
		want  int
		valid bool
	}{
		{[]byte{255}, 255, true}, {[]byte{0, 8}, 2048, true}, {[]byte{1, 2, 0, 0}, 513, true},
		{nil, 0, false}, {[]byte{1, 2, 3}, 0, false}, {make([]byte, 5), 0, false},
	} {
		got, err := number(tc.data)
		if (err == nil) != tc.valid || (tc.valid && got != tc.want) {
			t.Fatalf("%x: %d %v", tc.data, got, err)
		}
	}
}

type failLog struct{}

func (failLog) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type writer struct{ called bool }

func (w *writer) Write(p []byte) (int, error) { w.called = true; return len(p), nil }
func TestTraceFailureStopsWrite(t *testing.T) {
	w := &writer{}
	_, err := (tracingWriter{w, failLog{}}).Write([]byte{1})
	if !errors.Is(err, io.ErrClosedPipe) || w.called {
		t.Fatal(err)
	}
}
