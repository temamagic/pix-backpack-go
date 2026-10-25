package pix

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"image/gif"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWidgetAndInputVectors(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	calls := []func() error{func() error { return c.Countdown(5) }, c.Stopwatch, func() error { return c.ControlStopwatch(StopwatchStart) }, func() error { return c.Bike(BikeLeft) }, func() error { return c.StartGame(GameCrawler, 400) }, func() error { return c.GameInput(ButtonRight) }, func() error { return c.StartGame(GamePixBlocks, 400) }, func() error { return c.StartGame(GamePixelBreaker, 400) }}
	for _, f := range calls {
		if err := f(); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"000700000800050005", "010700000900010000ffffffffffffffff", "02080101", "03070000070300", "040700001200", "05080204", "0607000010900100", "070700001100"}
	for i, p := range r.packets {
		if hex.EncodeToString(p) != want[i] {
			t.Fatalf("packet %d: %x expected %s", i, p, want[i])
		}
	}
	r = &recorder{}
	c = NewClient(r)
	for i := 0; i < 256; i++ {
		if err := c.Input(4); err != nil {
			t.Fatal(err)
		}
	}
	if r.packets[254][2] != 255 || r.packets[255][2] != 1 {
		t.Fatal("input IDs must wrap 255 -> 1 separately from call IDs")
	}
}

func TestPackCustomFont(t *testing.T) {
	glyph := strings.TrimSpace(strings.Repeat("11111111 ", 20))
	pack := Pack{Version: 1, ScrollingTexts: []ScrollingText{{Text: "Ж", Color: "#123456", Speed: 80, Font: "custom"}}, Fonts: []CustomFont{{Name: "custom", BaseFont: "demo", Glyphs: map[string]string{"Ж": glyph}}}}
	data, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ReadPack(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err = parsed.Scroll(NewClient(r), 0, 16, 20); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.packets[2][3:], []byte{0, 0, 0, 0x12, 0x34, 0x56}) {
		t.Fatal("text palette mismatch")
	}
	for _, bad := range []string{"#12345z", "#fff", "123456", "#1234567"} {
		if _, err = ParseRGB(bad); err == nil {
			t.Fatal("invalid RGB accepted")
		}
	}
}
func TestDiceAndTorch(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	if err := c.Dice(6, 16, 20, 2048, RGB{255, 255, 255}); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, index := range r.packets[3][6:] {
		if index == 1 {
			count++
		}
	}
	if count != 24 {
		t.Fatalf("dice pixels=%d", count)
	}
	before := len(r.packets)
	if err := c.Dice(7, 16, 20, 2048, RGB{}); err == nil || len(r.packets) != before {
		t.Fatal("invalid die reached transport")
	}
	if err := c.Torch(16, 20, 2048, RGB{255, 0, 0}); err != nil {
		t.Fatal(err)
	}
}
func TestClockVector(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	zone := time.FixedZone("UTC+3", 10800)
	now := time.Unix(1700000000, 0).In(zone)
	if err := c.Clock(DefaultClock(now)); err != nil {
		t.Fatal(err)
	}
	p := r.packets[1]
	if p[1] != 7 || p[4] != 6 || binary.LittleEndian.Uint32(p[5:9]) != 1700010800 || binary.LittleEndian.Uint32(p[9:13]) != 1700010740 {
		t.Fatalf("clock epochs: %x", p)
	}
	if !bytes.Equal(p[13:25], []byte{0, 0, 1, 1, 6, 0, 0, 0, 0, 0, 1, 1}) || !bytes.Equal(p[25:], append([]byte{2}, []byte("UTC+3\x00\x00")...)) {
		t.Fatalf("clock fields: %x", p)
	}
}
func TestLivePartialUpdateAndResync(t *testing.T) {
	r := &recorder{}
	c := NewClient(r)
	live, err := c.StartLive(16, 20, []RGB{{}, {255, 0, 0}}, make([]byte, 320))
	if err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, 320)
	pixels[17] = 1
	n, err := live.Update(pixels)
	if err != nil || n != 7 {
		t.Fatalf("%d %v", n, err)
	}
	if got := hex.EncodeToString(r.packets[7]); got != "07000000110001" {
		t.Fatal(got)
	}
	n, err = live.Update(pixels)
	if err != nil || n != 0 || len(r.packets) != 8 {
		t.Fatal("unchanged frame wrote data")
	}
	pixels[17] = 0
	pixels[18] = 1
	r.failAt = 9
	if _, err = live.Update(pixels); err == nil {
		t.Fatal("expected failure")
	}
	r.failAt = 0
	if n, err = live.Update(pixels); err != nil || n != 326 || !bytes.Equal(r.packets[8][4:6], []byte{0, 0}) {
		t.Fatal("uncertain state must trigger full resync")
	}
	pixels[3] = 2
	if _, err = live.Update(pixels); err == nil {
		t.Fatal("out-of-palette update accepted")
	}
}
func TestStreamBeyondDeviceFrameLimit(t *testing.T) {
	a := Animation{Width: 1, Height: 1}
	for i := 0; i < 300; i++ {
		a.Frames = append(a.Frames, Frame{Pixels: []RGB{{byte(i % 2), 0, 0}}, Duration: 1})
	}
	r := &recorder{}
	c := NewClient(r)
	stats, err := c.Stream(context.Background(), a, 1)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Frames != 300 || len(r.packets) != 306 {
		t.Fatalf("%+v packets=%d", stats, len(r.packets))
	}
	for _, p := range r.packets[7:] {
		if p[1] != 0 {
			t.Fatal("stream must not clear, reset frame count, write palette or flash between frames")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := len(r.packets)
	if _, err = c.Stream(ctx, a, 1); err != context.Canceled || len(r.packets) != before {
		t.Fatal("cancelled stream wrote to device")
	}
}
func TestPlaybackOrder(t *testing.T) {
	for direction, want := range map[byte][]int{0: {0, 1, 2}, 1: {0, 1, 2, 1}, 2: {0, 1, 2}, 3: {0, 1, 2, 1, 0}} {
		if !reflect.DeepEqual(playbackOrder(3, direction), want) {
			t.Fatal(direction)
		}
	}
}
func TestFontsAndScrollingPacket(t *testing.T) {
	for name, want := range map[string]int{"demo": 5} {
		f, err := BuiltinFont(name)
		if err != nil || len(f) != want {
			t.Fatalf("%s: %d %v", name, len(f), err)
		}
	}
	font, err := BuiltinFont("demo")
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	c := NewClient(r)
	if err = c.ScrollText("AA", font, RGB{255, 255, 255}, 80, 16, 20); err != nil {
		t.Fatal(err)
	}
	config := r.packets[len(r.packets)-2]
	if !bytes.Equal(config[4:], []byte{2, 0, 80, 0, 8, 20, 3, 0, 0, 1, 1}) {
		t.Fatalf("config %x", config)
	}
	frame := r.packets[4][6:]
	glyph := font['A']
	for y := 0; y < 20; y++ {
		if !bytes.Equal(frame[y*16:y*16+8], glyph[y*8:y*8+8]) {
			t.Fatal("glyph row mapping")
		}
		if !bytes.Equal(frame[y*16+8:y*16+16], make([]byte, 8)) {
			t.Fatal("right glyph padding")
		}
	}
	before := len(r.packets)
	if err = c.ScrollText("Ж", font, RGB{}, 80, 16, 20); err == nil || len(r.packets) != before {
		t.Fatal("unsupported glyph reached transport")
	}
}
func TestCollectionExport(t *testing.T) {
	// Exact original 2x1 pixels survive both GIF and independent pixpack export.
	dir := t.TempDir()
	input := filepath.Join(dir, "in.pixpack")
	data := []byte(`{"pixpack":1,"animations":[{"name":"../A","direction":"alternate","frames":[{"duration":40,"pixels":[["#FF0000","#0000FF"]]},{"duration":80,"pixels":[["#000000","#00FF00"]]}]}]}`)
	if err := os.WriteFile(input, data, 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "export")
	items, err := ExportCollection(input, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].File != "00-a" {
		t.Fatal(items)
	}
	f, err := os.Open(filepath.Join(out, "00-a.gif"))
	if err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(f)
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Image) != 2 || g.Delay[0] != 4 || g.Delay[1] != 8 {
		t.Fatal(g)
	}
	r, _, _, _ := g.Image[0].At(0, 0).RGBA()
	if r != 65535 {
		t.Fatal("red pixel changed")
	}
	if _, err = ExportCollection(input, out); err == nil {
		t.Fatal("export overwrote files")
	}
	if _, err = os.Stat(filepath.Join(out, "contact-sheet.png")); err != nil {
		t.Fatal(err)
	}
}
