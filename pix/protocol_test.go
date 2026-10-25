package pix

import (
	"encoding/hex"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

type recorder struct {
	packets [][]byte
	failAt  int
	short   bool
}

func (r *recorder) Write(p []byte) (int, error) {
	if r.failAt > 0 && len(r.packets)+1 == r.failAt {
		return 0, errors.New("disconnected")
	}
	r.packets = append(r.packets, append([]byte(nil), p...))
	if r.short {
		return len(p) - 1, nil
	}
	return len(p), nil
}
func sample() Animation {
	return Animation{Width: 2, Height: 2, Direction: 1, Frames: []Frame{
		{Pixels: []RGB{{255, 0, 0}, {0, 0, 0}, {0, 255, 0}, {255, 0, 0}}, Duration: 300},
		{Pixels: []RGB{{0, 0, 255}, {0, 255, 0}, {0, 0, 0}, {255, 0, 0}}, Duration: 1000},
	}}
}

// Expected packets independently follow Android modules 731/739/743-748/784.
func TestUploadGolden(t *testing.T) {
	r := &recorder{}
	if err := NewClient(r).Upload(sample(), 200, true); err != nil {
		t.Fatal(err)
	}
	want := []string{"0007000000", "010200", "020100ff000000000000ff000000ff", "03000000000000010200", "04000100000003020100", "050300002c01e803", "060401", "070202", "0805"}
	var got []string
	for _, p := range r.packets {
		got = append(got, hex.EncodeToString(p))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("packets\ngot  %v\nwant %v", got, want)
	}
}
func TestPaletteSplitAndFullFrame(t *testing.T) {
	a := Animation{Width: 16, Height: 20, Frames: []Frame{{Duration: 40, Pixels: make([]RGB, 320)}}}
	for i := range a.Frames[0].Pixels {
		a.Frames[0].Pixels[i] = RGB{byte(i), 0, 0}
	}
	r := &recorder{}
	if err := NewClient(r).Upload(a, 200, false); err != nil {
		t.Fatal(err)
	}
	if len(r.packets[2]) != 483 || len(r.packets[3]) != 291 || r.packets[3][2] != 160 {
		t.Fatalf("bad palette split")
	}
	if len(r.packets[4]) != 326 || r.packets[4][6+255] != 255 || r.packets[4][6+256] != 0 {
		t.Fatal("frame must contain 320 row-major palette indices")
	}
}
func TestValidationBeforeWrite(t *testing.T) {
	for _, mutate := range []func(*Animation){
		func(a *Animation) { a.Frames[0].Duration = 65536 }, func(a *Animation) { a.Frames[0].Pixels = nil },
		func(a *Animation) { a.Direction = 4 }, func(a *Animation) { a.Frames = nil },
		func(a *Animation) { a.Width = 0 },
	} {
		a := sample()
		mutate(&a)
		r := &recorder{}
		if NewClient(r).Upload(a, 200, false) == nil || len(r.packets) != 0 {
			t.Fatal("invalid animation reached transport")
		}
	}
	r := &recorder{}
	if NewClient(r).Upload(sample(), 1, false) == nil || len(r.packets) != 0 {
		t.Fatal("device frame limit ignored")
	}
	a := Animation{Width: 16, Height: 20, Frames: []Frame{{Duration: 1, Pixels: make([]RGB, 320)}}}
	for i := range a.Frames[0].Pixels {
		a.Frames[0].Pixels[i] = RGB{byte(i), byte(i >> 8), 0}
	}
	if _, err := Prepare(a); err == nil {
		t.Fatal("palette overflow not rejected")
	}
	if _, err := Prepare(Quantize(a)); err != nil {
		t.Fatal(err)
	}
}
func TestFailureStopsUpload(t *testing.T) {
	r := &recorder{failAt: 4}
	if NewClient(r).Upload(sample(), 200, true) == nil || len(r.packets) != 3 {
		t.Fatal("upload must stop on first failure")
	}
	r = &recorder{short: true}
	if !errors.Is(NewClient(r).Clear(), io.ErrShortWrite) {
		t.Fatal("missing short write error")
	}
}
func TestCallIDWrap(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	for i := 0; i < 258; i++ {
		if err := c.Clear(); err != nil {
			t.Fatal(err)
		}
	}
	if r.packets[255][0] != 255 || r.packets[256][0] != 0 || r.packets[257][0] != 1 {
		t.Fatal("call counter did not wrap")
	}
}
func TestPixpack(t *testing.T) {
	s := `{"pixpack":1,"animations":[{"direction":"normal-stop","frames":[{"duration":500,"pixels":[["#ff0080","#000000"]]}]}]}`
	a, err := ReadPixpack(strings.NewReader(s), 2, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Direction != 2 || a.Frames[0].Pixels[0] != (RGB{255, 0, 128}) {
		t.Fatal(a)
	}
	for _, bad := range []string{strings.Replace(s, "#ff0080", "#ff00zz", 1), s + "{}", strings.Replace(s, `"pixpack":1`, `"pixpack":2`, 1)} {
		if _, err := ReadPixpack(strings.NewReader(bad), 2, 1, 0); err == nil {
			t.Fatal("invalid pixpack accepted")
		}
	}
}
func TestBrightness(t *testing.T) {
	for percent, want := range map[int]byte{0: 0, 1: 2, 50: 127, 100: 255} {
		got, err := Brightness(percent)
		if err != nil || got[0] != want {
			t.Fatalf("brightness %d: %v %v", percent, got, err)
		}
	}
	if _, err := Brightness(101); err == nil {
		t.Fatal("out of range")
	}
}
