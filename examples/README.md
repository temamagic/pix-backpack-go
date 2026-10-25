# Examples

All examples import the public module; run from the repository root:

```sh
go run ./examples/discover
go run ./examples/device-info -device 'NAME_OR_ADDRESS'
go run ./examples/display-image -device 'NAME_OR_ADDRESS' -file picture.png
go run ./examples/display-text -device 'NAME_OR_ADDRESS' -font mono -text 'Hello! Привет!'
go run ./examples/display-text -device 'NAME_OR_ADDRESS' -font-file my-font.json -text 'Hello! Привет!'
./build.sh text
./bin/pix-text -device 'NAME_OR_ADDRESS' -font mono -text 'Hello! Привет!'
./build.sh live
./bin/pix-live -device 'NAME_OR_ADDRESS'
```

Discover only scans; device-info also demonstrates connect/read/close. The image
example accepts exact-size PNG/JPEG/GIF/pixpack using reported dimensions, so it
also demonstrates animation upload without saving to flash. Live generates
original graphics. On macOS, examples that access Bluetooth should be built with
the Info.plist linker metadata in build.sh (set `package`/`output` equivalently),
or launched by a host already authorized for Bluetooth. None runs in unit tests.

Text is also available through `pixctl text 'Hello! Привет!'`; custom fonts use
`LoadFont`/`ReadFont`, a `Font` map or pixpack. See [font guide](../docs/fonts.md).
The original extracted font names are intentionally unavailable.
