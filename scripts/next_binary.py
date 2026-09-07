#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Manage only PREFIX/bin/feishu-cli-next and its SHA-256 ownership file."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
import tempfile

NAME = 'feishu-cli-next' + ('.exe' if os.name == 'nt' else '')


def regular(path):
    try:
        mode = path.lstat().st_mode
    except FileNotFoundError:
        return False
    if not stat.S_ISREG(mode):
        raise ValueError(f'Refusing non-regular file: {path}')
    return True


def digest(path):
    value = hashlib.sha256()
    with path.open('rb') as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b''):
            value.update(chunk)
    return value.hexdigest()


def staged_file(directory, content=None, source=None, mode=0o600):
    fd, name = tempfile.mkstemp(prefix='.next-install-', dir=directory)
    path = Path(name)
    try:
        with os.fdopen(fd, 'wb') as out:
            if source:
                with source.open('rb') as incoming:
                    for chunk in iter(lambda: incoming.read(1024 * 1024), b''):
                        out.write(chunk)
            else:
                out.write(content)
            out.flush()
            os.fchmod(out.fileno(), mode) if hasattr(os, 'fchmod') else os.chmod(path, mode)
            os.fsync(out.fileno())
        return path
    except BaseException:
        path.unlink(missing_ok=True)
        raise


def manage(action, prefix, source=None):
    prefix = Path(prefix).resolve()
    directory = prefix / 'bin'
    if action == 'uninstall' and not directory.exists() and not directory.is_symlink():
        return False
    directory.mkdir(parents=True, exist_ok=True)
    if directory.is_symlink() or not directory.is_dir():
        raise ValueError(f'Refusing non-directory or symlink: {directory}')
    binary = directory / NAME
    manifest = directory / ('.' + NAME + '.sha256')
    lock = directory / ('.' + NAME + '.install.lock')
    fd = os.open(lock, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    os.close(fd)
    staged = checksum = None
    try:
        exists = regular(binary)
        owned = regular(manifest)
        if exists != owned:
            raise ValueError('Binary ownership collision or incomplete installation; preserve and inspect existing files')
        previous = None
        if exists:
            if manifest.stat().st_size != 65:
                raise ValueError('Invalid binary ownership checksum')
            previous = manifest.read_text()
            if not re.fullmatch('[0-9a-f]{64}\n', previous) or digest(binary) != previous.strip():
                raise ValueError('Installed binary was edited; refusing to replace or remove it')
        if action == 'uninstall':
            if not exists:
                return False
            binary.unlink()
            manifest.unlink()
            return True
        if action == 'update' and not exists:
            raise ValueError('No managed binary to update; use install')
        source = Path(source)
        if not regular(source):
            raise ValueError('Source binary does not exist')
        staged = staged_file(directory, source=source, mode=0o755)
        expected = digest(staged) + '\n'
        if previous == expected:
            return False
        if action == 'install' and exists:
            raise ValueError('A different managed binary is installed; use update-install')
        checksum = staged_file(directory, content=expected.encode())
        if exists:
            os.replace(staged, binary)
        else:
            os.link(staged, binary) # Exclusive publication; never overwrites a collision.
        os.replace(checksum, manifest)
        return True
    finally:
        for path in (staged, checksum):
            if path:
                path.unlink(missing_ok=True)
        lock.unlink()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('install', 'update', 'uninstall'))
    parser.add_argument('--prefix', required=True, type=Path)
    parser.add_argument('--source', type=Path)
    args = parser.parse_args()
    if args.action != 'uninstall' and args.source is None:
        parser.error('--source is required for installation/update')
    try:
        changed = manage(args.action, args.prefix, args.source)
        print(json.dumps({'ok': True, 'action': args.action, 'changed': changed,
                          'path': str(args.prefix.resolve() / 'bin' / NAME)}, separators=(',', ':')))
    except (OSError, ValueError) as error:
        print(json.dumps({'ok': False, 'error': str(error)}, separators=(',', ':')), file=__import__('sys').stderr)
        raise SystemExit(1)


if __name__ == '__main__':
    main()
