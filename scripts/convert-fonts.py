#!/usr/bin/env python3
"""Convert pinned Terminus 4.49.1 BDF inputs into renamed PIX Mono OFL subsets.

Usage: python3 scripts/convert-fonts.py /path/to/terminus-font-4.49.1
Only reads data; upstream conversion programs are not executed.
"""
import hashlib
import json
from pathlib import Path
import sys

SOURCES = {
    'mono': ('ter-u16n.bdf', '5197662b22bf9f3e68d4af9f969a7fefa3edae40dd82ae969a147381130fb4ae'),
    'mono-bold': ('ter-u16b.bdf', '92861fecd6f94c5ecc764fc43df5a69eb000ae13a75f652c6525db0495274f0a'),
}
# Printable ASCII, Russian letters including Yo/yo, and the numero sign.
CODEPOINTS = [*range(0x20, 0x7f), *range(0x410, 0x450), 0x401, 0x451, 0x2116]


def convert(source):
    glyphs = {}
    for block in source.split('STARTCHAR ')[1:]:
        lines = block.splitlines()
        encoding = int(next(line.split()[1] for line in lines if line.startswith('ENCODING ')))
        if encoding not in CODEPOINTS:
            continue
        assert 'BBX 8 16 0 -4' in lines
        start = lines.index('BITMAP') + 1
        rows = lines[start:lines.index('ENDCHAR')]
        assert len(rows) == 16 and all(len(row) == 2 for row in rows)
        # BDF stores leftmost pixel in the high bit. Center the 16 rows in 20.
        bits = ['00000000'] * 2 + [format(int(row, 16), '08b') for row in rows] + ['00000000'] * 2
        glyphs[chr(encoding)] = ' '.join(bits)
    assert set(map(ord, glyphs)) == set(CODEPOINTS), 'missing requested characters'
    return dict(sorted(glyphs.items()))


def main():
    source = Path(sys.argv[1])
    target = Path(__file__).resolve().parent.parent / 'pix' / 'fonts'
    target.mkdir(exist_ok=True)
    for name, (filename, digest) in SOURCES.items():
        data = (source / filename).read_bytes()
        assert hashlib.sha256(data).hexdigest() == digest, filename
        glyphs = convert(data.decode('ascii'))
        (target / (name + '.json')).write_text(json.dumps(glyphs, ensure_ascii=False, indent=2) + '\n')
        print(name, len(glyphs), 'glyphs')


if __name__ == '__main__':
    main()
