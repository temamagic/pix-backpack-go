package pix

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Pack represents pixpack v1 animations, scrolling texts and custom fonts.
type Pack struct {
	Version        int             `json:"pixpack"`
	ExportedAt     int64           `json:"exportedAt,omitempty"`
	Animations     []PackAnimation `json:"animations,omitempty"`
	ScrollingTexts []ScrollingText `json:"scrollingTexts,omitempty"`
	Fonts          []CustomFont    `json:"fonts,omitempty"`
}

// PackAnimation contains a named interchange animation.
type PackAnimation struct {
	Name      string      `json:"name"`
	Direction string      `json:"direction"`
	Frames    []PackFrame `json:"frames"`
}

// PackFrame contains an interchange frame as rows of hex colors.
type PackFrame struct {
	Duration int        `json:"duration"`
	Pixels   [][]string `json:"pixels"`
}

// ScrollingText contains an interchange text entry, foreground, font and speed.
type ScrollingText struct {
	Name  string `json:"name"`
	Text  string `json:"text"`
	Color string `json:"color"`
	Speed int    `json:"speed"`
	Font  string `json:"font"`
}

// CustomFont contains optional overrides over a named built-in font; empty BaseFont starts empty.
type CustomFont struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	BaseFont    string            `json:"baseFont"`
	Glyphs      map[string]string `json:"glyphs"`
}

// ReadPack reads a single nonempty pixpack v1 JSON document (up to 32 MiB).
func ReadPack(r io.Reader) (Pack, error) {
	var p Pack
	d := json.NewDecoder(io.LimitReader(r, 32<<20))
	if err := d.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Pack{}, fmt.Errorf("expected a single JSON document")
	}
	if p.Version != 1 || len(p.Animations)+len(p.ScrollingTexts)+len(p.Fonts) == 0 {
		return Pack{}, fmt.Errorf("expected nonempty pixpack v1")
	}
	return p, nil
}

// OpenPack reads a pixpack file, optionally gzip compressed.
func OpenPack(path string) (Pack, error) {
	f, err := os.Open(path)
	if err != nil {
		return Pack{}, err
	}
	defer func() { _ = f.Close() }() // No file writes to flush.
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return Pack{}, e
		}
		defer func() { _ = z.Close() }() // Read-only decompressor; checksum errors are reported by Read.
		return ReadPack(z)
	}
	return ReadPack(f)
}

// Animation converts the selected pack animation to validated display frames.
func (p Pack) Animation(index, width, height int) (Animation, error) {
	if index < 0 || index >= len(p.Animations) {
		return Animation{}, fmt.Errorf("animation index out of range")
	}
	data, err := json.Marshal(Pack{Version: 1, Animations: []PackAnimation{p.Animations[index]}})
	if err != nil {
		return Animation{}, err
	}
	return ReadPixpack(strings.NewReader(string(data)), width, height, 0)
}

// ParseRGB parses one #RRGGBB color.
func ParseRGB(s string) (RGB, error) {
	if len(s) != 7 || s[0] != '#' {
		return RGB{}, fmt.Errorf("colour must be #RRGGBB")
	}
	b, err := hex.DecodeString(s[1:])
	if err != nil {
		return RGB{}, err
	}
	return RGB{b[0], b[1], b[2]}, nil
}

// Font resolves a custom font or a named built-in font.
func (p Pack) Font(name string) (Font, error) {
	for _, custom := range p.Fonts {
		if custom.Name != name {
			continue
		}
		font, err := BuiltinFont(custom.BaseFont)
		if custom.BaseFont == "" {
			font, err = Font{}, nil
		}
		if err != nil {
			return nil, err
		}
		for char, s := range custom.Glyphs {
			r := []rune(char)
			if len(r) != 1 {
				return nil, fmt.Errorf("glyph key must be one Unicode code point")
			}
			glyph, e := ParseGlyph(s)
			if e != nil {
				return nil, e
			}
			font[r[0]] = glyph
		}
		if _, ok := font[' ']; !ok {
			font[' '] = Glyph{}
		}
		return font, nil
	}
	return BuiltinFont(name)
}

// Scroll uploads the selected scrolling text entry using its font and color.
func (p Pack) Scroll(c *Client, index, width, height int) error {
	if index < 0 || index >= len(p.ScrollingTexts) {
		return fmt.Errorf("scrolling text index out of range")
	}
	text := p.ScrollingTexts[index]
	font, err := p.Font(text.Font)
	if err != nil {
		return err
	}
	color, err := ParseRGB(text.Color)
	if err != nil {
		return err
	}
	return c.ScrollText(text.Text, font, color, text.Speed, width, height)
}
