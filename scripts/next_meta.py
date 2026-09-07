#!/usr/bin/env python3
"""Restore the audited build catalog; remote refresh is an explicit inventory change."""
import gzip
import hashlib
import json
import os
from pathlib import Path
import tempfile

ROOT = Path(__file__).resolve().parents[1]


def restore(root=ROOT):
    reference = root / 'docs/reference'
    expected = json.loads((reference / 'coverage.json').read_text())['reference']['metadata_sha256']
    content = gzip.decompress((reference / 'meta_data.json.gz').read_bytes())
    if hashlib.sha256(content).hexdigest() != expected:
        raise ValueError('frozen metadata checksum mismatch; restore the audited snapshot')
    destination = root / 'internal/registry/meta_data.json'
    if destination.exists() and destination.read_bytes() == content:
        return
    with tempfile.NamedTemporaryFile(dir=destination.parent, prefix='.next-meta-', delete=False) as temporary:
        name = temporary.name
        try:
            temporary.write(content)
            temporary.flush()
            os.fsync(temporary.fileno())
        except BaseException:
            os.unlink(name)
            raise
    try:
        os.replace(name, destination)
    finally:
        if os.path.exists(name):
            os.unlink(name)


if __name__ == '__main__':
    restore()
