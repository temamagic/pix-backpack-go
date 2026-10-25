package pix

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

type cancelWriter struct {
	recorder
	cancel context.CancelFunc
}

func (w *cancelWriter) Write(p []byte) (int, error) {
	n, e := w.recorder.Write(p)
	w.cancel()
	return n, e
}

func TestUploadCancellation(t *testing.T) {
	for _, preCancelled := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		w := &cancelWriter{cancel: cancel}
		if preCancelled {
			cancel()
		}
		err := NewClient(w).UploadContext(ctx, sample(), 200, true)
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		want := 1
		if preCancelled {
			want = 0
		}
		if len(w.packets) != want {
			t.Fatalf("packets=%d want=%d", len(w.packets), want)
		}
		cancel()
	}
}
func TestNilTransport(t *testing.T) {
	if !errors.Is(NewClient(nil).Clear(), io.ErrClosedPipe) {
		t.Fatal("nil transport should return error")
	}
}
func TestConcurrentCommands(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 40 {
				if err := c.Clear(); err != nil {
					t.Error(err)
				}
			}
		})
	}
	wg.Wait()
	if len(r.packets) != 320 {
		t.Fatal(len(r.packets))
	}
	for i, p := range r.packets {
		if p[0] != byte(i) {
			t.Fatal("call order")
		}
	}
}
func TestRemovedFontsAndIndependentCopies(t *testing.T) {
	for _, name := range []string{"pixel", "bold", "slim", "legacy"} {
		if _, err := BuiltinFont(name); err == nil {
			t.Fatalf("extracted font %s distributed", name)
		}
	}
	a, _ := BuiltinFont("demo")
	delete(a, 'A')
	b, _ := BuiltinFont("demo")
	if _, ok := b['A']; !ok {
		t.Fatal("shared mutable font")
	}
}
func FuzzReadPixpack(f *testing.F) {
	f.Add(`{"pixpack":1,"animations":[{"direction":"normal","frames":[{"duration":1,"pixels":[["#000000"]]}]}]}`)
	f.Add(`{}`)
	f.Fuzz(func(t *testing.T, s string) {
		a, err := ReadPixpack(strings.NewReader(s), 1, 1, 0)
		if err == nil {
			if _, err := prepareStream(a); err != nil {
				t.Fatalf("reader accepted invalid animation: %v", err)
			}
		}
	})
}
func FuzzParseGlyph(f *testing.F) {
	f.Add(strings.TrimSpace(strings.Repeat("00000000 ", 20)))
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		g, err := ParseGlyph(s)
		if err == nil {
			for _, b := range g {
				if b > 1 {
					t.Fatal("nonbinary glyph")
				}
			}
		}
	})
}

func TestPaletteBounds(t *testing.T) {
	for _, start := range []int{-1, 256, int(^uint(0) >> 1)} {
		r := &recorder{}
		if err := NewClient(r).SetPalette(start, []RGB{{}, {}}); err == nil || len(r.packets) != 0 {
			t.Fatalf("invalid palette start %d reached writer", start)
		}
	}
}
