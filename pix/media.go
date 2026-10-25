package pix

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strconv"
	"strings"
)

// Load accepts PNG/JPEG/GIF and pixpack[.gz]. Images must match the device
// exactly: resizing pixel art implicitly can destroy it. GIFs are composited
// with disposal handling. Quantization, if desired, must be requested explicitly.
func Load(path string, width, height, index int) (Animation, error) {
	f, err := os.Open(path)
	if err != nil {
		return Animation{}, err
	}
	defer func() { _ = f.Close() }() // No file writes to flush.
	if strings.HasSuffix(strings.ToLower(path), ".pixpack") || strings.HasSuffix(strings.ToLower(path), ".pixpack.gz") {
		var r io.Reader = f
		if strings.HasSuffix(strings.ToLower(path), ".gz") {
			z, e := gzip.NewReader(f)
			if e != nil {
				return Animation{}, e
			}
			defer func() { _ = z.Close() }() // Read-only decompressor; checksum errors are reported by Read.
			r = z
		}
		return ReadPixpack(io.LimitReader(r, 32<<20), width, height, index)
	}
	if strings.HasSuffix(strings.ToLower(path), ".gif") {
		g, e := gif.DecodeAll(f)
		if e != nil {
			return Animation{}, e
		}
		if g.Config.Width != width || g.Config.Height != height {
			return Animation{}, fmt.Errorf("GIF must be %dx%d", width, height)
		}
		a := Animation{Width: width, Height: height}
		canvas := image.NewNRGBA(image.Rect(0, 0, width, height))
		var bg color.Color = color.Black
		if pal, ok := g.Config.ColorModel.(color.Palette); ok && int(g.BackgroundIndex) < len(pal) {
			bg = pal[g.BackgroundIndex]
		}
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
		for i, frame := range g.Image {
			before := image.NewNRGBA(canvas.Bounds())
			copy(before.Pix, canvas.Pix)
			draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
			duration := g.Delay[i] * 10
			if duration == 0 {
				duration = 100
			}
			a.Frames = append(a.Frames, Frame{Pixels: pixels(canvas), Duration: duration})
			if i < len(g.Disposal) {
				switch g.Disposal[i] {
				case gif.DisposalBackground:
					draw.Draw(canvas, frame.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
				case gif.DisposalPrevious:
					canvas = before
				}
			}
		}
		return a, nil
	}
	im, _, err := image.Decode(f)
	if err != nil {
		return Animation{}, err
	}
	if im.Bounds().Dx() != width || im.Bounds().Dy() != height {
		return Animation{}, fmt.Errorf("image must be %dx%d; resize it explicitly first", width, height)
	}
	return Animation{Width: width, Height: height, Frames: []Frame{{Pixels: pixels(im), Duration: 300}}}, nil
}

func parseHex(s string) (uint32, error) {
	n, err := strconv.ParseUint(s, 16, 24)
	return uint32(n), err
}

// Quantize maps all frames to the same standard 216-colour web-safe palette.
func Quantize(a Animation) Animation {
	out := a
	out.Frames = make([]Frame, len(a.Frames))
	for i, f := range a.Frames {
		out.Frames[i] = Frame{Duration: f.Duration, Pixels: make([]RGB, len(f.Pixels))}
		for j, c := range f.Pixels {
			r, g, b, _ := color.Palette(palette.WebSafe).Convert(color.RGBA{c.R, c.G, c.B, 255}).RGBA()
			out.Frames[i].Pixels[j] = RGB{byte(r >> 8), byte(g >> 8), byte(b >> 8)}
		}
	}
	return out
}

// RGBA() composites alpha against black, the physical display's off colour.
func pixels(im image.Image) []RGB {
	p := make([]RGB, 0, im.Bounds().Dx()*im.Bounds().Dy())
	for y := im.Bounds().Min.Y; y < im.Bounds().Max.Y; y++ {
		for x := im.Bounds().Min.X; x < im.Bounds().Max.X; x++ {
			r, g, b, _ := im.At(x, y).RGBA()
			p = append(p, RGB{byte(r >> 8), byte(g >> 8), byte(b >> 8)})
		}
	}
	return p
}

// ReadPixpack reads an animation at index with the required dimensions from pixpack v1.
func ReadPixpack(r io.Reader, width, height, index int) (Animation, error) {
	if width < 1 || height < 1 || width > 255 || height > 255 || width*height > 506 {
		return Animation{}, fmt.Errorf("invalid display dimensions")
	}
	var pack struct {
		Version    int `json:"pixpack"`
		Animations []struct {
			Direction string `json:"direction"`
			Frames    []struct {
				Duration int        `json:"duration"`
				Pixels   [][]string `json:"pixels"`
			} `json:"frames"`
		} `json:"animations"`
	}
	d := json.NewDecoder(io.LimitReader(r, 32<<20))
	if err := d.Decode(&pack); err != nil {
		return Animation{}, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return Animation{}, fmt.Errorf("expected one JSON document")
	}
	if pack.Version != 1 || index < 0 || index >= len(pack.Animations) {
		return Animation{}, fmt.Errorf("expected pixpack v1 with animation index %d", index)
	}
	src := pack.Animations[index]
	directions := map[string]byte{"normal": 0, "alternate": 1, "normal-stop": 2, "alternate-stop": 3}
	direction, ok := directions[src.Direction]
	if !ok {
		return Animation{}, fmt.Errorf("invalid direction %q", src.Direction)
	}
	a := Animation{Width: width, Height: height, Direction: direction}
	for i, f := range src.Frames {
		if len(f.Pixels) != height {
			return Animation{}, fmt.Errorf("frame %d must have %d rows", i, height)
		}
		frame := Frame{Duration: f.Duration}
		for _, row := range f.Pixels {
			if len(row) != width {
				return Animation{}, fmt.Errorf("frame %d must have %d columns", i, width)
			}
			for _, s := range row {
				var c RGB
				if len(s) != 7 || s[0] != '#' {
					return Animation{}, fmt.Errorf("invalid colour %q", s)
				}
				var value uint32
				// ParseUint rejects whitespace and partial hex tokens.
				var err error
				value, err = parseHex(s[1:])
				if err != nil {
					return Animation{}, err
				}
				c = RGB{byte(value >> 16), byte(value >> 8), byte(value)}
				frame.Pixels = append(frame.Pixels, c)
			}
		}
		a.Frames = append(a.Frames, frame)
	}
	if len(a.Frames) < 1 || len(a.Frames) > 10000 {
		return Animation{}, fmt.Errorf("expected 1..10000 frames")
	}
	for i, f := range a.Frames {
		if f.Duration < 1 || f.Duration > 65535 {
			return Animation{}, fmt.Errorf("frame %d: duration must be 1..65535 ms", i)
		}
	}
	return a, nil
}
