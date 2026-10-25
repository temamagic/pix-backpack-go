package pix

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinBilingualFonts(t *testing.T) {
	for _, name := range []string{"mono", "mono-bold"} {
		t.Run(name, func(t *testing.T) {
			font, err := BuiltinFont(name)
			if err != nil {
				t.Fatal(err)
			}
			if len(font) != 162 {
				t.Fatalf("glyphs=%d", len(font))
			}
			for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyzАБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯабвгдеёжзийклмнопрстуфхцчшщъыьэюя0123456789 №!?" {
				g, ok := font[r]
				if !ok {
					t.Fatalf("missing %q", r)
				}
				if r != ' ' && g == (Glyph{}) {
					t.Fatalf("blank %q", r)
				}
				for _, bit := range g {
					if bit > 1 {
						t.Fatal("nonbinary bitmap")
					}
				}
			}
			if font[' '] != (Glyph{}) {
				t.Fatal("space must be blank")
			}
			r := &recorder{}
			c := NewClient(r)
			text := "Hello, PIX! Привет, мир! Ёжик 123"
			if err := c.ScrollText(text, font, RGB{0, 255, 0}, 80, 16, 20); err != nil {
				t.Fatal(err)
			}
			seen := map[rune]byte{}
			var sequence []byte
			for _, ch := range " " + text {
				id, ok := seen[ch]
				if !ok {
					id = byte(len(seen))
					seen[ch] = id
				}
				sequence = append(sequence, id)
			}
			if len(r.packets) != len(seen)+5 {
				t.Fatalf("packets=%d glyphs=%d", len(r.packets), len(seen))
			}
			for ch, id := range seen {
				frame := r.packets[3+int(id)]
				glyph := font[ch]
				if len(frame) != 326 {
					t.Fatal("wrong GATT frame size")
				}
				for y := 0; y < 20; y++ {
					if !bytes.Equal(frame[6+y*16:6+y*16+8], glyph[y*8:y*8+8]) {
						t.Fatalf("glyph %q row %d", ch, y)
					}
					if !bytes.Equal(frame[6+y*16+8:6+(y+1)*16], make([]byte, 8)) {
						t.Fatal("right half must be blank")
					}
				}
			}
			config := r.packets[len(r.packets)-2]
			if !bytes.Equal(config[12:], sequence) {
				t.Fatal("Unicode sequence differs from glyph IDs")
			}
			delete(font, 'Я')
			fresh, _ := BuiltinFont(name)
			if _, ok := fresh['Я']; !ok {
				t.Fatal("shared font state")
			}
		})
	}
	normal, _ := BuiltinFont("mono")
	bold, _ := BuiltinFont("mono-bold")
	if normal['A'] == bold['A'] {
		t.Fatal("normal and bold should differ")
	}
}

type brokenFontReader struct{}

func (brokenFontReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestReadAndLoadCustomFont(t *testing.T) {
	glyph := strings.TrimSpace(strings.Repeat("10100101 ", 20))
	data, err := json.Marshal(map[string]string{"Ж": glyph, "A": glyph})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "custom.json")
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	font, err := LoadFont(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(font) != 3 || font['Ж'][0] != 1 || font['Ж'][1] != 0 || font[' '] != (Glyph{}) {
		t.Fatal("font decoded incorrectly")
	}
	r := &recorder{}
	if err := NewClient(r).ScrollText("AЖ", font, RGB{255, 255, 255}, 80, 16, 20); err != nil {
		t.Fatal(err)
	}
	if len(r.packets) != 8 {
		t.Fatal(len(r.packets))
	}
	_, err = ReadFont(brokenFontReader{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	if _, err := LoadFont(filepath.Join(t.TempDir(), "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	for _, bad := range []string{`null`, `{}`, `[]`, string(data) + `{}`, `{"AB":"` + glyph + `"}`, `{"Ж":"00000000"}`, string([]byte{0xff}), strings.Repeat(" ", (1<<20)+1)} {
		if _, err := ReadFont(strings.NewReader(bad)); err == nil {
			t.Fatal("invalid font accepted")
		}
	}
}

func TestCustomFontRejectsMissingRuneBeforeWrite(t *testing.T) {
	font, _ := BuiltinFont("mono")
	r := &recorder{}
	if err := NewClient(r).ScrollText("🙂", font, RGB{}, 80, 16, 20); err == nil || len(r.packets) != 0 {
		t.Fatal("unknown rune reached transport")
	}
}

func FuzzReadFont(f *testing.F) {
	f.Add(`{}`)
	f.Add(`{"A":"` + strings.TrimSpace(strings.Repeat("10000001 ", 20)) + `"}`)
	f.Fuzz(func(t *testing.T, s string) {
		font, err := ReadFont(strings.NewReader(s))
		if err == nil {
			for _, g := range font {
				for _, v := range g {
					if v > 1 {
						t.Fatal("nonbinary glyph")
					}
				}
			}
		}
	})
}

func TestBuiltinFontLicense(t *testing.T) {
	license := BuiltinFontLicense()
	for _, part := range []string{"Dimitar Toshkov Zhekov", "SIL OPEN FONT LICENSE Version 1.1", "Reserved Font Name"} {
		if !strings.Contains(license, part) {
			t.Fatal("missing embedded font license notice")
		}
	}
}

func TestPackFontWithoutBaseOrSpace(t *testing.T) {
	p := Pack{Fonts: []CustomFont{{Name: "mine", Glyphs: map[string]string{"Ж": strings.TrimSpace(strings.Repeat("10100101 ", 20))}}}}
	font, err := p.Font("mine")
	if err != nil {
		t.Fatal(err)
	}
	if font[' '] != (Glyph{}) || len(font) != 2 {
		t.Fatal("expected an automatic blank space")
	}
	r := &recorder{}
	if err := NewClient(r).ScrollText("Ж", font, RGB{}, 80, 16, 20); err != nil {
		t.Fatal(err)
	}
}
