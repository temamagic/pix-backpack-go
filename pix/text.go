package pix

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Glyph contains an 8x20 bitmap in row-major order, values 0 or 1.
type Glyph [160]byte

// Font maps Unicode characters to caller-owned binary 8x20 glyphs.
type Font map[rune]Glyph

// ParseGlyph parses twenty space-separated rows of eight binary digits.
func ParseGlyph(s string) (Glyph, error) {
	var glyph Glyph
	rows := strings.Split(s, " ")
	if len(rows) != 20 {
		return glyph, fmt.Errorf("glyph must contain 20 rows")
	}
	for y, row := range rows {
		if len(row) != 8 {
			return glyph, fmt.Errorf("glyph row must contain 8 bits")
		}
		for x, c := range row {
			if c != '0' && c != '1' {
				return glyph, fmt.Errorf("invalid glyph bit")
			}
			glyph[y*8+x] = byte(c - '0')
		}
	}
	return glyph, nil
}

//go:embed fonts/mono.json
var monoFont []byte

//go:embed fonts/mono-bold.json
var monoBoldFont []byte

//go:embed fonts/OFL.txt
var monoLicense string

// BuiltinFontLicense returns the copyright and OFL-1.1 terms for PIX Mono fonts.
// Keep this notice with redistributed binaries that embed these fonts.
func BuiltinFontLicense() string { return monoLicense }

// BuiltinFont returns an independent copy of demo, mono or mono-bold.
// PIX Mono fonts contain ASCII and Russian letters (including Yo/yo), plus numero.
// They are OFL-1.1 derivatives of Terminus; see fonts/OFL.txt.
// Application-extracted fonts are not distributed.
func BuiltinFont(name string) (Font, error) {
	switch name {
	case "mono":
		return ReadFont(bytes.NewReader(monoFont))
	case "mono-bold":
		return ReadFont(bytes.NewReader(monoBoldFont))
	case "demo":
	default:
		return nil, fmt.Errorf("unknown font %q; use mono, mono-bold, demo or supply a custom Font", name)
	}
	patterns := map[rune][]string{
		' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
		'A': {"00100", "01010", "10001", "11111", "10001", "10001", "10001"},
		'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
		'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
		'X': {"10001", "01010", "00100", "00100", "00100", "01010", "10001"},
	}
	font := Font{}
	for r, rows := range patterns {
		var glyph Glyph
		for y, row := range rows {
			for x, bit := range row {
				if bit == '1' {
					glyph[(y*2+3)*8+x+1] = 1
					glyph[(y*2+4)*8+x+1] = 1
				}
			}
		}
		font[r] = glyph
	}
	return font, nil
}

// ReadFont reads a JSON object mapping Unicode characters to twenty rows of
// eight binary digits separated by spaces. Input is capped at 1 MiB. A blank
// space is added if absent because ScrollText prepends it. TTF/OTF are not accepted.
func ReadFont(r io.Reader) (Font, error) {
	data, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read font: %w", err)
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("font exceeds 1 MiB")
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("font must contain valid UTF-8")
	}
	var rows map[string]string
	d := json.NewDecoder(bytes.NewReader(data))
	if err := d.Decode(&rows); err != nil {
		return nil, fmt.Errorf("decode font: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("expected a single font JSON document")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("font must contain glyphs")
	}
	font := Font{}
	for key, value := range rows {
		runes := []rune(key)
		if len(runes) != 1 {
			return nil, fmt.Errorf("glyph key %q must contain one Unicode code point", key)
		}
		glyph, err := ParseGlyph(value)
		if err != nil {
			return nil, fmt.Errorf("glyph %q: %w", key, err)
		}
		font[runes[0]] = glyph
	}
	if _, ok := font[' ']; !ok {
		font[' '] = Glyph{}
	}
	return font, nil
}

// LoadFont opens a user-provided JSON bitmap font; see ReadFont for the format.
func LoadFont(path string) (Font, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // Read-only file; no writes to flush.
	return ReadFont(f)
}

// ScrollText uploads glyphs as ordinary frame-memory sprites, then activates
// firmware scrolling mode. The text scrolls without ongoing host traffic.
// Up to 500 runes including a leading space fit one safe SET_CONFIG packet.
func (c *Client) ScrollText(text string, font Font, foreground RGB, speedMS, width, height int) error {
	if width < 8 || width > 25 || height != 20 || speedMS < 1 || speedMS > 65535 {
		return fmt.Errorf("text requires width>=8, height=20 and speed 1..65535 ms")
	}
	runes := append([]rune{' '}, []rune(text)...)
	if len(runes) > 500 {
		return fmt.Errorf("text exceeds 499 characters")
	}
	glyphs := []Glyph{}
	lookup := map[rune]byte{}
	sequence := make([]byte, len(runes))
	for i, r := range runes {
		glyph, ok := font[r]
		if !ok {
			return fmt.Errorf("font lacks character %q", r)
		}
		for _, v := range glyph {
			if v > 1 {
				return fmt.Errorf("glyph %q contains non-binary pixels", r)
			}
		}
		index, ok := lookup[r]
		if !ok {
			if len(glyphs) == 255 {
				return fmt.Errorf("text exceeds 255 distinct glyphs")
			}
			index = byte(len(glyphs))
			lookup[r] = index
			glyphs = append(glyphs, glyph)
		}
		sequence[i] = index
	}
	config := binary.LittleEndian.AppendUint16([]byte{2, 0}, uint16(speedMS))
	config = append(config, 8, 20)
	config = binary.LittleEndian.AppendUint16(config, uint16(len(sequence)))
	config = append(config, sequence...)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.command(7, 0, 0, 0); err != nil {
		return err
	}
	if err := c.command(2, 0); err != nil {
		return err
	}
	if err := c.setPalette(0, []RGB{{}, foreground}); err != nil {
		return err
	}
	for i, glyph := range glyphs {
		indices := make([]byte, width*height)
		for y := 0; y < 20; y++ {
			copy(indices[y*width:y*width+8], glyph[y*8:y*8+8])
		}
		if err := c.setFrame(i, 0, indices); err != nil {
			return err
		}
	}
	if err := c.setConfig(0, config); err != nil {
		return err
	}
	return c.command(2, byte(len(glyphs)))
}
