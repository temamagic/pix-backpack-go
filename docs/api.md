# API guide

Import `github.com/temamagic/pix-backpack-go/pix` for portable codecs and
`github.com/temamagic/pix-backpack-go/ble` for the desktop transport.

- `Prepare(Animation)` validates dimensions, durations, palette and packet limits
  without writing. Pixel order is row-major RGB; alpha composites against black.
- `Client.Upload`/`UploadContext` upload exact colors; flash storage is opt-in.
  `Quantize` explicitly maps colors to WebSafe 216. MaxFrames is read from the
  device and further constrained by the recovered packet format.
- `Load` reads exact-size PNG/JPEG/GIF or pixpack[.gz]; `ReadPixpack` selects an
  animation; `ReadPack`/`OpenPack` expose all interchange sections.
- `StartLive` initializes frame 0; `Live.Update` sends the changed range.
  Keep the palette fixed and do not change modes during a live session.
  After an error the next update resends the full frame. Do not mutate input
  slices concurrently with a call.
- `Stream(ctx, animation, loops)` performs host playback. Zero loops means until
  cancellation. Stopping leaves the last frame displayed. No ongoing flash writes.
- `Font` contains 8×20 binary glyphs. `BuiltinFont("mono")` and `mono-bold` provide
  licensed Latin/Russian fonts; `demo` contains space/A/P/I/X only. `ReadFont` and
  `LoadFont` read user-provided JSON glyph maps and add a blank space if absent.
  Custom pack fonts can use an empty `baseFont`. Old extracted fonts are excluded.
  See [font guide](fonts.md) and `BuiltinFontLicense` for the font OFL notices.
- Clock, bike, countdown, stopwatch and game helpers encode recovered configs;
  see [protocol](protocol/README.md) for confidence and exact messages.
- `Command`, `SetConfig`, `SetFrame`, `SetPalette`, `SetFrameCount`, `SetDurations`,
  `SetDirection`, `Save`, `Clear`, `Restart`, `Input` expose low-level operations.
  SetFrame offset counts pixels; SetConfig offset counts bytes.
- `ExportCollection` exports user-provided content into a new output directory,
  without overwriting files. Missing demo-font glyphs in contact-sheet labels are
  skipped; animation pixels are unaffected.

`ble.Scan(Options)` finds advertised PIX names/services. `ble.Open` requires exactly
one match (use Options.Device for a name/address filter), reads dimensions, frame
capacity and brightness and optionally firmware/hardware strings. Always handle
Close errors. Options.Log and Trace are optional and may contain private data.
Notifications are raw copied bytes, not decoded RPC success acknowledgments.

See [examples](../examples/README.md) for complete error-handling code and
[architecture](architecture.md) for timeout/concurrency contracts. Errors wrap
transport failures with `%w` where context is added; use errors.Is/As, not text.
