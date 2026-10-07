# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import gzip
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('next_meta', Path(__file__).with_name('next_meta.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


class MetadataTest(unittest.TestCase):
    def test_hash_verification_and_exact_restoration(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            reference = root / 'docs/reference'
            reference.mkdir(parents=True)
            target = root / 'internal/registry/meta_data.json'
            target.parent.mkdir(parents=True)
            content = b'{"services":[{"name":"fixture"}]}\n'
            (reference / 'meta_data.json.gz').write_bytes(gzip.compress(content))
            manifest = reference / 'coverage.json'
            manifest.write_text(json.dumps({'reference': {'metadata_sha256': hashlib.sha256(content).hexdigest()}}))
            module.restore(root)
            self.assertEqual(target.read_bytes(), content)
            target.write_bytes(b'changed')
            module.restore(root)
            self.assertEqual(target.read_bytes(), content)
            manifest.write_text(json.dumps({'reference': {'metadata_sha256': 'invalid'}}))
            with self.assertRaises(ValueError):
                module.restore(root)
            self.assertEqual(target.read_bytes(), content)


if __name__ == '__main__':
    unittest.main()
