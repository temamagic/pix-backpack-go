# Fonts and custom scrolling text

PIX scrolling uploads glyph bitmaps as ordinary frames, then sends a sequence of
glyph indices to the firmware. The device does not need the font installed and
never receives a TTF/OTF file. Width is fixed at 8 pixels per character, height 20;
`mono`/`mono-bold` center a 16-row bitmap within that cell. Text scrolling currently
requires a 20-pixel-high display and width 8..25.

## Built-in fonts

| API/CLI name | Primary font name | Characters | License |
|---|---|---|---|
| `mono` (CLI default) | PIX Mono | 162: printable ASCII, Russian including Ё/ё, № | OFL-1.1 |
| `mono-bold` | PIX Mono Bold | Same 162 characters, bold | OFL-1.1 |
| `demo` | Original demonstration font | Space, A, P, I, X | Project MIT |

PIX Mono variants derive from Terminus Font 4.49.1; see
[font source, pinned hashes and conversion](../pix/fonts/README.md) and the
[complete font license](../pix/fonts/OFL.txt). Embed/distribute font notices with
binaries. The license text is also embedded: `pix.BuiltinFontLicense()` or
`pixctl font-license`. These fonts are independent of original PIX app resources.

```go
font, err := pix.BuiltinFont("mono")
if err != nil { return err }
return conn.Client.ScrollText("Hello! Привет! Ёё 123", font,
    pix.RGB{R: 40, G: 255, B: 40}, 80, conn.Info.Width, conn.Info.Height)
```

## Load your own JSON bitmap font

`pix.ReadFont(io.Reader)` and `pix.LoadFont(path)` accept a JSON object. Each key
must be one Unicode code point; each value is exactly twenty space-separated
rows of eight `0`/`1` digits. For example, one row is `00111100`.
Row 0 is the top row and the leftmost digit is the leftmost pixel. All 160 pixels
are binary; foreground color comes from ScrollText. A blank space glyph is added
if omitted, because scrolling prepends a space. An explicitly supplied space is
preserved. Font JSON is UTF-8, limited to 1 MiB and one document. Other glyphs are
not substituted silently: unsupported text fails before any device writes.

```go
font, err := pix.LoadFont("my-font.json")
if err != nil { return err }
return conn.Client.ScrollText("Hello / Привет", font,
    pix.RGB{R: 255, G: 128}, 90, conn.Info.Width, conn.Info.Height)
```

```sh
./build.sh pixctl
./bin/pixctl text -font-file my-font.json 'Hello / Привет'
```

The shipped `pix/fonts/mono.json` and `mono-bold.json` use this exact format and can
serve as licensed starting points. Preserve OFL notices and choose your own
primary name for derivatives. `-font-file` overrides `-font`; it cannot be combined
with `-pack`. TTF/OTF fonts must first be rasterized by your own tooling into this
bitmap format; this library does not load or rasterize arbitrary system fonts.

## Pixpack custom fonts

`Pack.Font(name)` supports named built-in base fonts plus per-character overrides;
empty `baseFont` starts with no base glyphs. Supply `glyphs` in the same row format.
`Pack.Scroll` uses the selected scrollingTexts entry. A missing space is filled
with a blank glyph. The reader accepts the pixpack v1 interchange structure implemented by `Pack`.
Do not assume third party pack fonts or artwork can be redistributed.

The API also accepts a caller-created `Font` map directly. Include a space glyph
in that map. The [display-text example](../examples/display-text/main.go) supports
both built-ins and file loading, device selection and explicit error handling.

## Limits and evidence

At most 499 user characters plus a leading space fit the config packet. At most
255 distinct glyphs are uploaded; duplicate characters reuse frame indices.
The library checks each glyph and the entire text before the first write. Text
is rendered by firmware after disconnect and is loaded into RAM without flash save.
The actual memory capacity of other devices and other firmware remains unverified.

Synthetic tests compare every uploaded glyph row, blank right padding, Unicode
sequence indices, normal/bold differences, independent font maps, file loading,
malformed JSON/keys/rows, size limits and rejection before writes. Hardware
results and human visual confirmation are recorded separately in
[hardware observations](hardware.md).
