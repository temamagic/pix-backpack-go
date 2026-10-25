"""Synthetic tests for publication policy; no real credentials or app content."""
import importlib.util
from pathlib import Path
import unittest
import subprocess
import tempfile
import sys

spec = importlib.util.spec_from_file_location('policy', Path(__file__).with_name('check-publication.py'))
policy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(policy)


class PolicyTests(unittest.TestCase):
    def test_private_paths(self):
        for path in ['app.apk', 'Wrapped.app/Info.plist', 'research/note.md', 'bin/pixctl', 'a.zip', '.env.production', 'pix/fonts.json']:
            with self.subTest(path=path):
                self.assertTrue(policy.problems(path, b'content'))

    def test_magic_and_credentials(self):
        for data in [b'\x7fELFmore', b'PK\x03\x04', ('ghp_' + 'a' * 36).encode(), ('password="' + 'z' * 24 + '"').encode()]:
            self.assertTrue(policy.problems('innocent.txt', data))

    def test_synthetic_fixtures_allowed(self):
        for path, data in [('pix/testdata/a.pixpack', b'{"pixpack":1}'), ('pix/testdata/a.png', b'\x89PNG\r\n\x1a\n')]:
            self.assertFalse(policy.problems(path, data))

    def test_large_blob(self):
        self.assertTrue(policy.problems('fixture.dat', b'0' * (policy.MAX_BYTES + 1)))

    def test_deleted_private_file_is_still_rejected_in_history(self):
        script = Path(__file__).with_name('check-publication.py').resolve()
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def git(*args):
                subprocess.run(['git', '-c', 'user.name=Policy Test', '-c',
                                'user.email=policy@example.invalid', *args],
                               cwd=root, check=True, capture_output=True)
            git('init')
            (root / 'private.apk').write_bytes(b'synthetic app placeholder')
            git('add', '.')
            git('commit', '-m', 'test: synthetic forbidden object')
            (root / 'private.apk').unlink()
            (root / 'README.md').write_text('Safe public tree\n')
            git('add', '-A')
            git('commit', '-m', 'test: remove synthetic forbidden object')
            current = subprocess.run([sys.executable, str(script)], cwd=root, capture_output=True)
            history = subprocess.run([sys.executable, str(script), '--history'], cwd=root, capture_output=True)
            self.assertEqual(current.returncode, 0, current.stdout)
            self.assertEqual(history.returncode, 1, history.stdout)
            self.assertIn(b'private.apk', history.stdout)


if __name__ == '__main__':
    unittest.main()
