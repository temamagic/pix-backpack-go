# PIX Mono fonts (OFL-1.1)

`mono.json` (PIX Mono) and `mono-bold.json` (PIX Mono Bold) are renamed, subsetted
8×20 adaptations of Terminus Font 4.49.1, normal/bold 8×16. They contain 162 glyphs:
printable ASCII U+0020–007E, Russian U+0410–044F, Ё/ё (U+0401/U+0451) and № (U+2116).
Two blank rows are added above and below each bitmap; columns and MSB-first order
are preserved. No TTF rasterizer or runtime system font is needed.

Original font copyright: (C) 2020 Dimitar Toshkov Zhekov.
The original Reserved Font Name is "Terminus Font". Our derivative primary names
are PIX Mono and PIX Mono Bold. These font assets remain under [SIL OFL 1.1](OFL.txt),
not project MIT. Bundling is permitted subject to the included license and notices.
When distributing binaries, retain the font copyright and OFL license with them.
Original app fonts are not included or used.

Source: [official Terminus homepage](https://terminus-font.sourceforge.net/),
[4.49.1 source archive](https://downloads.sourceforge.net/project/terminus-font/terminus-font-4.49/terminus-font-4.49.1.tar.gz).
Only the pinned font data is read; upstream conversion programs are not executed.

| Input | SHA-256 |
|---|---|
| terminus-font-4.49.1.tar.gz | `d961c1b781627bf417f9b340693d64fc219e0113ad3a3af1a3424c7aa373ef79` |
| ter-u16n.bdf | `5197662b22bf9f3e68d4af9f969a7fefa3edae40dd82ae969a147381130fb4ae` |
| ter-u16b.bdf | `92861fecd6f94c5ecc764fc43df5a69eb000ae13a75f652c6525db0495274f0a` |

Reproduce from the extracted official archive (outside the repository):

```sh
python3 scripts/convert-fonts.py /path/to/terminus-font-4.49.1
```

The converter validates input hashes and writes deterministic UTF-8 JSON. The
original `demo` font remains Go-authored project code under MIT.
