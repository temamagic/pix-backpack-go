#!/usr/bin/env python3
"""Check staged paths/blobs and optionally every blob reachable from HEAD.

This lightweight policy check detects common mistakes, not every possible secret.
Synthetic image/pixpack fixtures are permitted. Native executables and app assets
are not. Read Git objects rather than ignored local build outputs.
"""
import argparse
import re
import subprocess
import sys
from pathlib import PurePosixPath

MAX_BYTES = 1024 * 1024
BLOCKED_PARTS = {'research', 'exports', 'decompiled', 'original-apps', 'raw-captures', 'node_modules', 'vendor', 'bin'}
BLOCKED_SUFFIXES = {'.apk', '.ipa', '.app', '.bundle', '.exe', '.dll', '.dylib', '.so', '.zip', '.tar', '.gz', '.7z', '.p12', '.pfx', '.key', '.pem', '.log', '.trace'}
MAGIC = (b'\x7fELF', b'MZ', b'\xcf\xfa\xed\xfe', b'\xce\xfa\xed\xfe', b'\xfe\xed\xfa\xcf', b'\xfe\xed\xfa\xce', b'\xca\xfe\xba\xbe', b'PK\x03\x04', b'\x1f\x8b')
SECRETS = [re.compile(p) for p in (
    rb'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----',
    rb'gh[pousr]_[A-Za-z0-9]{36,}',
    rb'github_pat_[A-Za-z0-9_]{60,}',
    rb'AKIA[0-9A-Z]{16}',
    rb'xox[baprs]-[A-Za-z0-9-]{20,}',
    rb'(?i)(?:api[_-]?key|access[_-]?token|password|secret)\s*[:=]\s*[\'\"][A-Za-z0-9_+/=-]{16,}[\'\"]',
)]


def git(*args):
    return subprocess.check_output(['git', *args])


def problems(path, data):
    p = PurePosixPath(path.lower())
    issues = []
    if (set(p.parts) & BLOCKED_PARTS or any(part.endswith('.app') or 'decompiled' in part or part.startswith('preservation-backup') for part in p.parts)
            or p.suffix in BLOCKED_SUFFIXES or p.name.startswith('.env') or p.name in {'fonts.json', 'android.bundle'}):
        issues.append('private/application/generated path')
    if len(data) > MAX_BYTES:
        issues.append('blob exceeds 1 MiB; review policy before adding a fixture')
    if data.startswith(MAGIC):
        issues.append('executable or unexpected archive magic')
    if any(pattern.search(data) for pattern in SECRETS):
        issues.append('possible credential/private key (content redacted)')
    return issues


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--history', action='store_true', help='also inspect all blobs reachable from HEAD')
    args = parser.parse_args()
    entries = []
    for raw in git('ls-files', '--stage', '-z').split(b'\0'):
        if not raw:
            continue
        meta, name = raw.split(b'\t', 1)
        mode, oid, stage = meta.decode().split()
        if stage != '0' or mode not in {'100644', '100755'}:
            print('FAIL: unresolved index, symlink or submodule:', name.decode(errors='replace'))
            return 1
        entries.append((oid, name.decode()))
    if args.history and subprocess.run(['git', 'rev-parse', '--verify', 'HEAD'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL).returncode == 0:
        for line in git('rev-list', '--objects', 'HEAD').decode().splitlines():
            oid, _, name = line.partition(' ')
            if git('cat-file', '-t', oid).strip() == b'blob':
                entries.append((oid, name))
    failed = False
    for oid, name in sorted(set(entries)):
        # Limit reads even if an accidentally staged blob is huge.
        size = int(git('cat-file', '-s', oid))
        data = git('cat-file', 'blob', oid) if size <= MAX_BYTES else b''
        issues = problems(name, data)
        if size > MAX_BYTES:
            issues.append('blob exceeds 1 MiB')
        for issue in issues:
            print(f'FAIL: {name}: {issue}')
            failed = True
    if not entries:
        print('FAIL: no staged/tracked files; stage the intended public tree first')
        return 1
    if not failed:
        print(f'Publication check passed: {len(set(entries))} staged/historical blob-path pairs')
    return int(failed)


if __name__ == '__main__':
    sys.exit(main())
