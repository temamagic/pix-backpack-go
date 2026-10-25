# Development

Go 1.25 is the minimum because the pinned Bluetooth module requires it. Source
uses standard Go only; macOS BLE pulls cbgo/CGO/CoreBluetooth. Do not update BLE
dependencies without repeating long-write and hardware checks.

```sh
go mod tidy
gofmt -w ble pix cmd examples
go install golang.org/x/tools/cmd/goimports@v0.42.0
goimports -w ble pix cmd examples
go build ./...
go vet ./...
go test ./...
go test -race ./...
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
golangci-lint config verify
golangci-lint run
python3 scripts/check-publication.py --history
```

The pinned lint tool requires Go 1.26+ to install; CI uses Go 1.27.1 for lint and
also tests the module minimum 1.25.0. This does not raise the module minimum.
The build script creates ignored `bin/` outputs and embeds macOS Bluetooth privacy
metadata. `go build ./...` compiles but does not execute BLE calls.

To test local library changes from a separate consumer project, create a directory outside this repository:

```sh
go mod init example.com/pix-consumer
go mod edit -require=github.com/temamagic/pix-backpack-go@v0.0.0
go mod edit -replace=github.com/temamagic/pix-backpack-go=/absolute/path/to/pix-backpack-go
```

Use the README portable example as `main.go`, then `go mod tidy`, `go run .` and
`go build .`. No archive files or `internal` imports should be required.

## Fuzzing

```sh
go test ./pix -run '^$' -fuzz FuzzReadPixpack -fuzztime 10s
go test ./pix -run '^$' -fuzz FuzzParseGlyph -fuzztime 10s
go test ./pix -run '^$' -fuzz FuzzReadFont -fuzztime 10s
```

## Manual hardware verification (not CI)

1. Record OS, architecture, adapter, device model, firmware and dimensions. Redact
   device identifiers in shared results. Disconnect other controlling apps.
2. Build `./build.sh pixctl`; scan and select one peripheral. Read `info`.
3. Create your own exact-size orientation PNG. Show it without `-save`; verify
   orientation/colors visually. Test live updates, unchanged frames and reconnect.
4. Display `PIX` using demo glyphs; confirm scrolling. Test clock, countdown and
   each game input individually. Record delivery and visual results separately.
5. Exercise interrupt/cancellation and connection loss, close, reconnect and retry.
   Confirm palette writes up to 483 bytes and full-frame writes work intact.
6. Test brightness/demo only intentionally; restore previous settings. Flash save
   and power cycling are separate opt-in checks; do not use OTA/PIN/restart probes.

New hardware results must specify what was actually observed. Historical notes
are not evidence of new transport or font changes.
