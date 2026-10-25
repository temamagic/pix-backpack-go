# Architecture

`pix` is a portable protocol and media package. `Writer` needs one complete
write-with-response GATT operation per `Write`; long values must not be split into
unframed writes. `Client` serializes packet sequences with a mutex and maintains
call/input counters. A transport error stops uploads immediately; partial state
may remain on the display. `Live` sends the next full frame after a failed update.

`ble` discovers peripherals using TinyGo Bluetooth, resolves a known service,
reads device characteristics and supplies the RPC writer to `pix.Client`.
Failure after connect disconnects the peripheral. `Close` is idempotent.
Treat each connection as a single session: do not overlap direct characteristic
reads, brightness/demo writes, notifications or Close with client writes.

`cmd/pixctl` composes the packages. Neither library imports the CLI. Examples use
public imports only. `macos/Info.plist` and `build.sh` embed Bluetooth permission
metadata in executable builds; no Swift or project-owned Objective-C is needed.
The cbgo dependency uses CGO/Objective-C and CoreBluetooth on macOS.

`Stream` checks context before setup and each update and while waiting. UploadContext
checks cancellation before each packet. Existing native BLE calls cannot be
interrupted by context, and mutex acquisition waits for the active operation.
Scan Timeout bounds discovery only; it is not a whole-session deadline. Native
connect/discovery/write calls use the dependency's timeouts (typically 10 seconds
on Darwin). A blocked transport must implement its own I/O timeout.
