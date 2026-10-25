package pix

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// CollectionItem describes one animation written by ExportCollection.
type CollectionItem struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	Frames     int    `json:"frames"`
	Colors     int    `json:"colors"`
	Direction  string `json:"direction"`
	DurationMS int    `json:"duration_ms"`
	File       string `json:"file"`
}

// ExportCollection writes exact PNG first frames, animated GIFs, independent
// pixpacks and an HTML gallery. It refuses to overwrite existing export files.
func ExportCollection(path, out string) ([]CollectionItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }() // No file writes to flush.
	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		z, e := gzip.NewReader(f)
		if e != nil {
			return nil, e
		}
		defer func() { _ = z.Close() }() // Read-only decompressor; checksum errors are reported by Read.
		r = z
	}
	data, err := io.ReadAll(io.LimitReader(r, 32<<20))
	if err != nil {
		return nil, err
	}
	var pack struct {
		Version    int               `json:"pixpack"`
		Animations []json.RawMessage `json:"animations"`
	}
	if err = json.Unmarshal(data, &pack); err != nil {
		return nil, err
	}
	if pack.Version != 1 {
		return nil, fmt.Errorf("expected pixpack v1")
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return nil, err
	}
	var items []CollectionItem
	for index, raw := range pack.Animations {
		var metadata struct {
			Name, Direction string
			Frames          []struct{ Pixels [][]string }
		}
		if err = json.Unmarshal(raw, &metadata); err != nil {
			return nil, err
		}
		if len(metadata.Frames) == 0 || len(metadata.Frames[0].Pixels) == 0 {
			return nil, fmt.Errorf("empty animation %d", index)
		}
		height := len(metadata.Frames[0].Pixels)
		width := len(metadata.Frames[0].Pixels[0])
		single, err := json.Marshal(struct {
			Version    int               `json:"pixpack"`
			Animations []json.RawMessage `json:"animations"`
		}{1, []json.RawMessage{raw}})
		if err != nil {
			return nil, err
		}
		a, err := ReadPixpack(bytes.NewReader(single), width, height, 0)
		if err != nil {
			return nil, fmt.Errorf("animation %d: %w", index, err)
		}
		p, err := prepareStream(a)
		if err != nil {
			return nil, fmt.Errorf("animation %d: %w", index, err)
		}
		slug := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return '-'
		}, metadata.Name)
		slug = strings.Trim(slug, "-")
		if slug == "" {
			slug = "animation"
		}
		stem := fmt.Sprintf("%02d-%s", index, slug)
		item := CollectionItem{Index: index, Name: metadata.Name, Frames: len(a.Frames), Colors: len(p.Palette), Direction: metadata.Direction, File: stem}
		pal := make(color.Palette, len(p.Palette))
		for i, c := range p.Palette {
			pal[i] = color.RGBA{c.R, c.G, c.B, 255}
		}
		g := gif.GIF{Config: image.Config{ColorModel: pal, Width: width, Height: height}, LoopCount: 0}
		for _, i := range playbackOrder(len(a.Frames), a.Direction) {
			im := image.NewPaletted(image.Rect(0, 0, width, height), pal)
			copy(im.Pix, p.Frames[i])
			g.Image = append(g.Image, im)
			g.Delay = append(g.Delay, max(1, (a.Frames[i].Duration+5)/10))
			g.Disposal = append(g.Disposal, gif.DisposalNone)
			item.DurationMS += a.Frames[i].Duration
		}
		if a.Direction >= 2 {
			g.LoopCount = -1
		}
		var pngBytes, gifBytes bytes.Buffer
		if err = png.Encode(&pngBytes, g.Image[0]); err != nil {
			return nil, err
		}
		if err = gif.EncodeAll(&gifBytes, &g); err != nil {
			return nil, err
		}
		for suffix, content := range map[string][]byte{".png": pngBytes.Bytes(), ".gif": gifBytes.Bytes(), ".pixpack": append(single, '\n')} {
			if err = writeNew(filepath.Join(out, stem+suffix), content); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	manifest, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return nil, err
	}
	if err = writeNew(filepath.Join(out, "manifest.json"), append(manifest, '\n')); err != nil {
		return nil, err
	}
	t := template.Must(template.New("gallery").Parse(`<!doctype html><meta charset="utf-8"><title>PIX collection</title><style>body{background:#15171b;color:#eee;font:16px system-ui;margin:32px}main{display:grid;grid-template-columns:repeat(auto-fill,minmax(190px,1fr));gap:24px}article{background:#23262c;padding:16px;border-radius:12px}img{image-rendering:pixelated;width:128px;height:160px;object-fit:contain;background:black}a{color:#9bcaff}</style><h1>PIX collection — {{len .}} animations</h1><main>{{range .}}<article><img src="{{.File}}.gif" alt="{{.Name}}"><h3>{{.Index}} · {{.Name}}</h3><p>{{.Frames}} frames · {{.Colors}} colours</p><a href="{{.File}}.png">PNG</a> · <a href="{{.File}}.gif">GIF</a> · <a href="{{.File}}.pixpack">pixpack</a></article>{{end}}</main>`))
	var html bytes.Buffer
	if err = t.Execute(&html, items); err != nil {
		return nil, err
	}
	if err = writeNew(filepath.Join(out, "index.html"), html.Bytes()); err != nil {
		return nil, err
	}
	if len(items) > 0 {
		if err = writeContactSheet(out, items); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func writeContactSheet(out string, items []CollectionItem) error {
	sheet := image.NewRGBA(image.Rect(0, 0, 1000, 220*((len(items)+4)/5)))
	font, err := BuiltinFont("demo")
	if err != nil {
		return err
	}
	for i, item := range items {
		f, e := os.Open(filepath.Join(out, item.File+".png"))
		if e != nil {
			return e
		}
		im, e := png.Decode(f)
		closeErr := f.Close()
		if e == nil {
			e = closeErr
		}
		if e != nil {
			return e
		}
		left, top := (i%5)*200, (i/5)*220
		for y := 0; y < 160; y++ {
			for x := 0; x < 128; x++ {
				sheet.Set(left+36+x, top+10+y, im.At(x*im.Bounds().Dx()/128, y*im.Bounds().Dy()/160))
			}
		}
		label := fmt.Sprintf("%02d %s", item.Index, item.Name)
		for n, r := range label {
			if n >= 24 {
				break
			}
			glyph, ok := font[r]
			if !ok {
				continue
			}
			for y := 0; y < 20; y++ {
				for x := 0; x < 8; x++ {
					if glyph[y*8+x] != 0 {
						sheet.Set(left+4+n*8+x, top+180+y, color.RGBA{230, 230, 230, 255})
					}
				}
			}
		}
	}
	var b bytes.Buffer
	if err = png.Encode(&b, sheet); err != nil {
		return err
	}
	return writeNew(filepath.Join(out, "contact-sheet.png"), b.Bytes())
}
func writeNew(path string, b []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
