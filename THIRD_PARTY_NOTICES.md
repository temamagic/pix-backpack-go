# Provenance and acknowledgments

The starting directory had no Git metadata, license headers, author history or
remote. The owner supplied the Go implementation and authorized MIT preparation.
Source inspection found protocol reconstruction and synthetic Go tests, not bundled
third party Go source. This is a source audit, not independent proof of authorship.
The contributors notice covers original contributions and does not claim ownership
of firmware, applications or others' artwork.

The owner identifies all original project code as written using AI. The human
provided research direction, instructions, source discovery/selection, hardware
and reports of observed behavior. AI performed reverse engineering, protocol
analysis, implementation, tests and documentation under human direction. This
does not describe the authorship of third party sources or dependencies. See
the README's AI authorship disclaimer.

One explicit exception was found: `pix/fonts.json` contained bitmap resources
extracted from the original Android/iOS apps. It is excluded from this repository.
`BuiltinFont` retains `demo`, an original five-character demonstration font,
plus separately licensed PIX Mono fonts described below. `Font`, `ParseGlyph`,
custom `.pixpack` glyphs and scrolling remain available. No pixel/bold/slim/legacy
application resource is relicensed.
Animation collections, application bundles, decompiled source and personal BLE
logs are retained only in the owner's private archive outside this Git root.

Protocol descriptions are independently phrased summaries of observed behavior.
Numeric module IDs in documentation identify private research evidence, not
vendored code. Synthetic test vectors describe protocol bytes and use no original
artwork. Existing hardware results are attributed to prior local project notes.

## Dependencies

Go resolves dependencies from `go.mod`/`go.sum`; none are vendored. Pinned
`tinygo.org/x/bluetooth` v0.16.0 uses a BSD 3-Clause license. Transitive desktop
modules include cbgo (Apache-2.0), logrus (MIT), godbus/dbus (BSD 2-Clause),
go-ole (MIT), winrt-go (MIT) and golang.org/x/sys (BSD 3-Clause).
Their upstream copyright and license terms continue to apply, especially when
redistributing compiled binaries. Consult each module's LICENSE in the module
cache; MIT for this project does not replace dependency licenses.

## Bundled OFL font assets

`pix/fonts/mono.json` and `mono-bold.json` are subsets and 8×20 adaptations of
Terminus Font 4.49.1 (8×16 normal/bold), Copyright (C) 2020 Dimitar Toshkov Zhekov,
with Reserved Font Name "Terminus Font". They are distributed under SIL OFL 1.1,
not the project's MIT license. Their derivative primary names are PIX Mono and
PIX Mono Bold. Font design is third party work, not AI-authored project code.

The full [font copyright and license](pix/fonts/OFL.txt) is included in source and
embedded for access via `pix.BuiltinFontLicense()` / `pixctl font-license`.
Retain these notices when redistributing binaries with the fonts. See
[provenance, pinned hashes and reproducible conversion](pix/fonts/README.md).
The converter is original project code; upstream font conversion programs are
not bundled or executed.
