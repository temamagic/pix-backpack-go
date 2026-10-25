# Agent guide

## Start here

- `README.ru.md` and `README.md`: user-facing behavior and limitations. Keep both
  versions aligned and preserve human wording changes.
- `docs/architecture.md`: package boundaries, concurrency and cancellation.
- `docs/protocol/README.md`: UUIDs, packet formats and evidence confidence.
- `docs/compatibility.md` and `docs/hardware.md`: platform support and actual
  hardware observations. Compilation alone does not establish BLE compatibility.
- `THIRD_PARTY_NOTICES.md` and font licenses: provenance and redistribution terms.

## Code map

- `pix/`: portable protocol client, media readers, live updates, scrolling text,
  fonts and synthetic tests. It must not import the CLI or require Bluetooth.
- `ble/`: desktop transport using pinned TinyGo Bluetooth. macOS needs CGO,
  CoreBluetooth and Xcode Command Line Tools; unsupported builds return an error.
- `cmd/pixctl/`: CLI, including `text`, `-font`, `-font-file` and pixpack import.
- `pix/fonts/`: licensed bitmap subsets, OFL text and source/conversion metadata.
- `scripts/`: reproducible font conversion and publication checks.
- `examples/`: complete examples using public module imports.
- `.github/workflows/ci.yml`: native OS/Go matrix, lint and publication policy.

The module root is this directory, not its parent. Module path:
`github.com/temamagic/pix-backpack-go`. Minimum Go: 1.25.0.

## Verify changes

```sh
gofmt -w ble pix cmd examples
goimports -w ble pix cmd examples
go mod tidy
go build ./...
go vet ./...
go test ./...
go test -race ./...
golangci-lint config verify
golangci-lint run
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/check-publication.py --history
```

Publication checks inspect staged/tracked objects: stage intended files before
running them and review `git diff --cached`. Tool versions/install commands are in
`docs/development.md`; do not silently raise the module minimum for dev tooling.
Run hardware commands with `./build.sh pixctl` so macOS privacy metadata is embedded.
Do not run hardware tests in ordinary unit tests or CI. `.gitattributes` preserves LF.

## Protocol and hardware rules

- One Writer.Write is one complete GATT write with response. Never fix transport
  issues by arbitrarily splitting packet bytes. Preserve synthetic golden vectors.
- Validate inputs before writing. Preserve call/input counters and packet limits.
- Use one active rendering/session per connection; direct native characteristic
  calls must not overlap client operations or Close.
- Existing BLE calls cannot be interrupted mid-call by context. Document that limit.
- Classify evidence as Verified, Observed, Inferred or Unknown. A successful GATT
  write is delivery evidence, not a visual or semantic firmware confirmation.
- For new hardware tests, get the user's visual observations, record OS/model/
  firmware and what was actually seen. Redact addresses/names from public logs.
- Do not save to flash, restart, alter PIN or probe OTA unless explicitly requested.
  RAM-only text/display tests do not establish persistence or other firmware support.

## Fonts and publication

Never add original apps, decompiled code, extracted proprietary resources, private
captures, secrets or generated executables. The sibling research archive is outside
this Git root and must stay private. Do not import or depend on it for builds/tests.
Redistributable fonts are allowed only with verified provenance, license notices
and reproducible conversion. Keep OFL font assets under OFL rather than project MIT;
respect reserved font names and distinguish third party design from AI-authored code.
Do not restore the removed pixel/bold/slim/legacy app resources.

## Git and documentation

Use Conventional Commits. Preserve unrelated user edits, avoid destructive history
rewrites and inspect remote state before publishing. Publishing needs user
instruction; do not create tags/releases automatically. Keep API GoDoc and both
READMEs consistent with actual implementation and hardware evidence. The project
was developed using AI under human research direction; retain that disclosure.
