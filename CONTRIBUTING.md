# Contributing

Coding agents should start with [AGENTS.md](AGENTS.md) for the code map and verification rules.

Use Go 1.25 or later; macOS BLE requires CGO and Xcode command line tools.
Run the commands in [development](docs/development.md) before opening a PR.
Use Conventional Commits and describe behavior, evidence, tests and limitations.

Add opcodes as typed helpers in `pix`, with synthetic golden bytes and invalid
input tests. Keep BLE details in `ble`. Do not change write boundaries to fix
transport errors. Label protocol claims Verified/Observed/Inferred/Unknown and
identify firmware and the kind of evidence. Hardware tests must be opt-in, never
part of ordinary `go test`. Document hardware steps and results separately.

Report bugs with OS/architecture, Go version, device/firmware, steps and expected
vs actual behavior. Remove peripheral identifiers, artwork and secrets from logs.
Never submit original applications, decompiled source, extracted fonts/artwork,
private captures or content without redistribution rights. Use synthetic fixtures.
