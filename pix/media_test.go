package pix

import (
	"compress/gzip"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGIFCompositing(t *testing.T) {
	pal := color.Palette{color.Black, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}}
	for _, disposal := range []byte{gif.DisposalPrevious, gif.DisposalBackground} {
		first := image.NewPaletted(image.Rect(0, 0, 3, 1), pal)
		first.Pix = []byte{1, 1, 1}
		second := image.NewPaletted(image.Rect(1, 0, 2, 1), pal)
		second.Pix[0] = 2
		third := image.NewPaletted(image.Rect(2, 0, 3, 1), pal)
		third.Pix[0] = 3
		g := gif.GIF{Image: []*image.Paletted{first, second, third}, Delay: []int{10, 20, 0}, Disposal: []byte{gif.DisposalNone, disposal, gif.DisposalNone}, Config: image.Config{ColorModel: pal, Width: 3, Height: 1}}
		path := filepath.Join(t.TempDir(), "test.gif")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = gif.EncodeAll(f, &g); err != nil {
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
		a, err := Load(path, 3, 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		want := RGB{255, 0, 0}
		if disposal == gif.DisposalBackground {
			want = RGB{}
		}
		if a.Frames[1].Pixels[1] != (RGB{0, 255, 0}) || a.Frames[2].Pixels[1] != want || a.Frames[2].Pixels[2] != (RGB{0, 0, 255}) {
			t.Fatalf("disposal %d: %+v", disposal, a)
		}
		if a.Frames[1].Duration != 200 || a.Frames[2].Duration != 100 {
			t.Fatal("wrong GIF delays")
		}
	}
}
func TestPNGAlphaAndDimensions(t *testing.T) {
	im := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	im.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	im.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 0})
	path := filepath.Join(t.TempDir(), "test.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := Load(path, 2, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Frames[0].Pixels[0] != (RGB{128, 0, 0}) || a.Frames[0].Pixels[1] != (RGB{}) {
		t.Fatal("alpha not composited on black")
	}
	if _, err = Load(path, 16, 20, 0); err == nil {
		t.Fatal("size mismatch ignored")
	}
}
func TestGzipPixpack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pixpack.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	z := gzip.NewWriter(f)
	_, err = z.Write([]byte(`{"pixpack":1,"animations":[{"direction":"normal","frames":[{"duration":40,"pixels":[["#123456"]]}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	a, err := Load(path, 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Frames[0].Pixels[0] != (RGB{0x12, 0x34, 0x56}) {
		t.Fatal(a)
	}
}
