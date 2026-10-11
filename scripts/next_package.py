#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT
"""Build checksummed next archives and a local npm wrapper; never publish."""
import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[1]
TARGETS = ('darwin/amd64', 'darwin/arm64', 'linux/amd64', 'linux/arm64',
           'linux/riscv64', 'windows/amd64', 'windows/arm64')
NAME = 'feishu-cli-next'


def archive_name(version, target, variant='normal'):
    os_name, arch = target.split('/')
    suffix = '' if variant == 'normal' else '-authsidecar'
    return f'{NAME}-{version}-{os_name}-{arch}{suffix}' + ('.zip' if os_name == 'windows' else '.tar.gz')


def write_archive(destination, files):
    """Stable metadata and ordering; binary builds also use -trimpath/-buildvcs=false."""
    if destination.suffix == '.zip':
        with zipfile.ZipFile(destination, 'x', compression=zipfile.ZIP_DEFLATED) as archive:
            for name, source in sorted(files.items()):
                info = zipfile.ZipInfo(name, (1980, 1, 1, 0, 0, 0))
                info.compress_type = zipfile.ZIP_DEFLATED
                info.external_attr = (0o100755 if name.endswith('.exe') else 0o100644) << 16
                archive.writestr(info, source.read_bytes())
    else:
        with destination.open('xb') as raw:
            with gzip.GzipFile(filename='', fileobj=raw, mode='wb', mtime=0) as zipped:
                with tarfile.open(fileobj=zipped, mode='w') as archive:
                    for name, source in sorted(files.items()):
                        data = source.read_bytes()
                        info = tarfile.TarInfo(name)
                        info.size = len(data)
                        info.mode = 0o755 if name == NAME else 0o644
                        archive.addfile(info, io.BytesIO(data))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, required=True, help='new directory (existing paths are refused)')
    parser.add_argument('--target', action='append', choices=TARGETS)
    parser.add_argument('--variant', action='append', choices=('normal', 'authsidecar'))
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    metadata = json.loads((ROOT / 'package.json').read_text())
    version = metadata['version']
    checksums = []
    with tempfile.TemporaryDirectory(prefix='next-build-') as temp:
        for target in args.target or TARGETS:
            goos, goarch = target.split('/')
            binary = Path(temp) / (NAME + ('.exe' if goos == 'windows' else ''))
            for variant in args.variant or ('normal', 'authsidecar'):
                command = ['go', 'build', '-trimpath', '-buildvcs=false', '-ldflags',
                           f'-s -w -X github.com/larksuite/cli/internal/build.Version={version} -X github.com/larksuite/cli/internal/build.Date=local']
                if variant == 'authsidecar':
                    command += ['-tags', 'authsidecar']
                command += ['-o', str(binary), '.']
                print(f'Building {target} {variant}', flush=True)
                subprocess.run(command, cwd=ROOT, env={**os.environ, 'GOOS': goos, 'GOARCH': goarch, 'CGO_ENABLED': '0'}, check=True)
                name = archive_name(version, target, variant)
                files = {binary.name: binary, 'README.md': ROOT / 'README.md', 'LICENSE': ROOT / 'LICENSE',
                         'CHANGELOG.md': ROOT / 'CHANGELOG.md', 'docs/distribution.md': ROOT / 'docs/distribution.md',
                         'docs/contracts/skill.md': ROOT / 'docs/contracts/skill.md'}
                write_archive(output / name, files)
                checksums.append(f'{hashlib.sha256((output / name).read_bytes()).hexdigest()}  {name}\n')
    (output / 'checksums.txt').write_text(''.join(sorted(checksums)))
    package = output / 'npm'
    (package / 'scripts').mkdir(parents=True)
    for name in ('install.js', 'install-wizard.js', 'run.js'):
        shutil.copy2(ROOT / 'scripts' / name, package / 'scripts' / name)
    for name in ('package.json', 'package-lock.json', 'LICENSE', 'CHANGELOG.md'):
        shutil.copy2(ROOT / name, package / name)
    (package / 'docs').mkdir()
    shutil.copy2(ROOT / 'docs/distribution.md', package / 'docs/distribution.md')
    (package / 'docs/contracts').mkdir()
    shutil.copy2(ROOT / 'docs/contracts/skill.md', package / 'docs/contracts/skill.md')
    shutil.copy2(output / 'checksums.txt', package / 'checksums.txt')
    print(f'Prepared {output}; local npm package: npm pack --ignore-scripts {package}')


if __name__ == '__main__':
    main()
