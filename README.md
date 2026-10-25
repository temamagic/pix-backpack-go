# Pix Backpack Go

English · [Русский](README.ru.md)

Independent Go library and CLI for preserving programmable PIX Backpack displays.

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8)](go.mod)
[![CI](https://github.com/temamagic/pix-backpack-go/actions/workflows/ci.yml/badge.svg)](https://github.com/temamagic/pix-backpack-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)
[![Go Reference](https://pkg.go.dev/badge/github.com/temamagic/pix-backpack-go/pix.svg)](https://pkg.go.dev/github.com/temamagic/pix-backpack-go/pix)
[![Code quality: golangci-lint](https://img.shields.io/badge/code_quality-golangci--lint-2C9FAF)](.golangci.yml)
[![Last commit](https://img.shields.io/github/last-commit/temamagic/pix-backpack-go)](https://github.com/temamagic/pix-backpack-go/commits/main/)
[![AI-generated](https://img.shields.io/badge/code-AI--generated-8A2BE2)](#ai-authorship-and-disclaimer)

PIX backpacks contain an LED display originally controlled by mobile apps. The goal of this
project is to preserve local control and protocol knowledge after the manufacturer
closed in January 2022.
No cloud account is required. This is an independent project, unaffiliated with PIX.

All project code was written using AI; see [authorship and disclaimer](#ai-authorship-and-disclaimer).

## Features

- Platform independent RGB palette and command encoding with synthetic golden tests.
- Exact-size PNG/JPEG/GIF and `.pixpack` animation import; explicit WebSafe quantization.
- Animation upload (up to 254 frames), live partial updates and cancellable host streaming.
- Scrolling text with custom 8×20 glyphs and bundled PIX Mono / PIX Mono Bold fonts for English and Russian.
- Clock, countdown, stopwatch, bike and firmware game controls; Snake and dice.
- CLI discovery, device information, image display, tracing and collection export.

Tested on a real PIX Original V1 backpack with firmware 2.5.1 and a 16×20 display,
connected to macOS Apple Silicon.
Game controls, other firmware and Mini remain experimental; flash persistence,
RPC response decoding, OTA and PIN management are unverified or unavailable.

## Installation

```sh
go get github.com/temamagic/pix-backpack-go
```

Requires Go 1.25 or later, as required by the pinned Bluetooth dependency.

## Quick start

This example creates command packets without Bluetooth or proprietary assets:

```go
package main

import (
    "fmt"
    "github.com/temamagic/pix-backpack-go/pix"
)

type packets struct{}
func (packets) Write(p []byte) (int, error) {
    fmt.Printf("%x\n", p)
    return len(p), nil
}

func main() {
    animation := pix.Animation{Width: 1, Height: 1, Frames: []pix.Frame{
        {Pixels: []pix.RGB{{R: 255}}, Duration: 300},
    }}
    if err := pix.NewClient(packets{}).Upload(animation, 1, false); err != nil {
        panic(err)
    }
}
```

To use real hardware on macOS, install Go and Xcode Command Line Tools, grant
Bluetooth access, then clone the repository and build a CLI with embedded privacy metadata:

```sh
git clone https://github.com/temamagic/pix-backpack-go.git
cd pix-backpack-go
./build.sh pixctl
./bin/pixctl scan
./bin/pixctl info -device 'ADVERTISED_NAME_OR_ADDRESS'
./bin/pixctl show -device 'ADVERTISED_NAME_OR_ADDRESS' picture.png
./bin/pixctl text -device 'ADVERTISED_NAME_OR_ADDRESS' 'PIX'
```

The image must match the reported dimensions. `show` writes RAM by default;
`-save` explicitly requests flash storage, whose persistence is unverified.
Release the mobile app connection before connecting. Traces may include your
artwork and scan output includes peripheral identifiers; redact before sharing.

## Fonts and bilingual text

`mono` (default) and `mono-bold` support printable ASCII, Russian letters including
Ё/ё, digits and №. They are 8×20 bitmap adaptations of Terminus, distributed under
[SIL OFL 1.1](pix/fonts/OFL.txt); [source and conversion details](pix/fonts/README.md).
The small original `demo` font remains available. App-extracted fonts are excluded.

```sh
./bin/pixctl text -font mono 'Hello, PIX! Привет, мир! Ёжик 123'
./bin/pixctl text -font mono-bold 'Hello! Привет!'
./bin/pixctl text -font-file my-font.json 'My text / Мой текст'
./bin/pixctl font-license
```

Select a device with `-device` if more than one is found. In Go, use
`pix.LoadFont(path)` or `pix.ReadFont(reader)` and pass the result to `ScrollText`.
[Font guide](docs/fonts.md) covers the JSON format, pixpack overrides and examples.

## Compatibility

macOS BLE uses CoreBluetooth, CGO and Xcode Command Line Tools. Hardware observations
are available for Apple Silicon. The macOS Intel build has been checked, but not
operation on a physical Intel Mac. Linux/BlueZ and Windows/WinRT remain experimental:
a successful build does not confirm BLE packet delivery on hardware. The `pix`
package works independently of BLE; on unsupported platforms `ble` returns
`ErrUnsupportedPlatform`.

## Documentation

- [BLE protocol and confidence levels](docs/protocol/README.md)
- [API guide](docs/api.md) · [Go Reference](https://pkg.go.dev/github.com/temamagic/pix-backpack-go/pix)
- [Examples](examples/README.md)
- [Compatibility](docs/compatibility.md)
- [Architecture](docs/architecture.md) · [Development and hardware checks](docs/development.md)
- [Historical hardware observations](docs/hardware.md)
- [Provenance and acknowledgments](THIRD_PARTY_NOTICES.md)

## Preservation

Research used owner-supplied Android and iOS application materials and local BLE
experiments. This repository preserves protocol knowledge and a local-control
implementation for existing hardware. `.pixpack` is an application interchange
format, separate from BLE RPC.

The [former manufacturer's account](https://www.instagram.com/pix.backpack/) is a
historical reference. No applications, decompiled source, extracted app fonts or
animation collections are distributed here.

## AI authorship and disclaimer

All original code in this repository was written using AI. The human acted as
researcher: defining goals, providing instructions, finding and selecting source
materials, supplying hardware and reporting observed device behavior.
Reverse engineering, technical analysis, protocol reconstruction, implementation,
tests and documentation were carried out by AI under human direction.
This disclosure covers the project's own work; third party dependencies and
sources remain the work of their respective authors.

AI-generated code and protocol conclusions may contain errors. Passing automated
checks and individual hardware observations do not guarantee correctness,
completeness or compatibility with your device. Review the code and relevant
protocol assumptions before use, especially before writing to flash or sending
experimental commands. The software is provided "as is", without warranties,
as described in the [MIT license](LICENSE.md).

## License

Original implementation, original documentation and synthetic examples are under
[MIT](LICENSE.md). Third party applications and resources retain their respective
rights; company closure does not grant redistribution rights.

PIX names identify compatible hardware and do not imply endorsement.

PIX Mono / PIX Mono Bold fonts are separately licensed under [SIL OFL 1.1](pix/fonts/OFL.txt).

See [notices](THIRD_PARTY_NOTICES.md).
