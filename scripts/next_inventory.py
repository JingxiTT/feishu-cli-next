#!/usr/bin/env python3
"""Freeze and reconcile the reference's real registrations without modifying it.

Source equality is conservative implementation evidence, not proof of live parity.
Only Python's standard library and the repository's Go toolchain are required.
"""
import argparse
import collections
import gzip
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

ROOT = Path(__file__).resolve().parents[1]
EXPORTER = Path(__file__).with_name('next_inventory_export.go')
CONTRACT_ROOTS = ('cmd/', 'shortcuts/', 'events/', 'extension/', 'internal/', 'errs/', 'tests/', 'sidecar/')
BUILD_FILES = ('Makefile', '.goreleaser.yml', 'go.mod', 'go.sum', 'package.json', 'build.sh', 'main.go', 'main_authsidecar.go', 'main_noauthsidecar.go', 'content_embed.go', 'LICENSE')


def digest(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, ensure_ascii=False, separators=(',', ':')).encode()


def git(root, *args):
    return subprocess.check_output(['git', '-C', str(root), *args]).decode().strip()


def tracked(root):
    return git(root, 'ls-files', '-z').split('\0')


def save(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    raw = json.dumps(data, ensure_ascii=False, sort_keys=True, indent=2).encode() + b'\n'
    if path.suffix == '.gz':
        raw = gzip.compress(raw, mtime=0)
    path.write_bytes(raw)


def read(path):
    raw = path.read_bytes()
    return json.loads(gzip.decompress(raw) if path.suffix == '.gz' else raw)


def source_inventory(root):
    files, conditions, exports = {}, {}, {}
    names = tracked(root)
    for name in names:
        path = root / name
        if not path.is_file():
            continue
        if name.startswith(CONTRACT_ROOTS) or name in BUILD_FILES or name.startswith(('scripts/', '.github/')):
            raw = path.read_bytes()
            files[name] = digest(raw)
            if name.endswith('.go'):
                text = raw.decode()
                tags = re.findall(r'^//go:build (.+)$', text, re.M)
                if tags:
                    conditions[name] = tags[0]
                # Evidence index only. Complete source hashes catch changes to
                # grouped declarations, signatures, and implementation as well.
                if name.startswith('extension/') and not name.endswith('_test.go'):
                    exports[name] = re.findall(r'^(?:type|func|const|var)\s+(?:\([^\n]+\)\s+)?([A-Z]\w*)', text, re.M)
    return {'files': files, 'conditions': conditions, 'public_symbol_index': exports,
            'test_files': [n for n in files if n.endswith(('_test.go', '.test.js', '.test.sh'))]}


def export(root, metadata, brand):
    """Compile actual registrations in a temporary copy, never the reference."""
    with tempfile.TemporaryDirectory(prefix='next-inventory-') as temporary:
        work = Path(temporary) / 'source'
        work.mkdir()
        for name in tracked(root):
            source = root / name
            if name and source.is_file():
                target = work / name
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, target)
        shutil.copyfile(EXPORTER, work / 'scripts/next_inventory_export.go')
        (work / 'internal/registry/meta_data.json').write_bytes(metadata)
        env = {k: v for k, v in os.environ.items() if not k.startswith(('LARKSUITE_CLI_', 'FEISHU_'))}
        env.update(LARKSUITE_CLI_CONFIG_DIR=str(Path(temporary) / 'config'),
                   LARKSUITE_CLI_REMOTE_META='off', LARKSUITE_CLI_BRAND=brand,
                   LARKSUITE_CLI_NO_UPDATE_NOTIFIER='1', LARKSUITE_CLI_NO_SKILLS_NOTIFIER='1')
        result = subprocess.run(['go', 'run', 'scripts/next_inventory_export.go'], cwd=work,
                                env=env, capture_output=True, timeout=240)
        if result.returncode:
            raise RuntimeError(result.stderr.decode()[-6000:])
        result = json.loads(result.stdout)
        if not result['services']:
            raise ValueError('empty generated service catalog; inventory would be incomplete')
        return result


def metadata_methods(metadata):
    methods = {}
    def walk(resources, prefix):
        for name, resource in sorted(resources.items()):
            path = prefix + ' ' + name
            for method, spec in sorted(resource.get('methods', {}).items()):
                methods[path + ' ' + method] = spec
            walk(resource.get('resources', {}), path)
    for service in metadata.get('services', []):
        walk(service.get('resources', {}), service['name'])
    if not methods:
        raise ValueError('metadata contains no API methods')
    return methods


def section(path, contract):
    domain = contract.get('domain')
    if domain:
        return 'events' if domain == 'event' else domain
    first = path.split(' ')[0]
    if first == 'event':
        return 'events'
    return 'foundation'


def freeze(reference, metadata_path, out):
    metadata = metadata_path.read_bytes()
    parsed = json.loads(metadata)
    methods = metadata_methods(parsed)
    sources = source_inventory(reference)
    variants = {brand: export(reference, metadata, brand) for brand in ('feishu', 'lark')}
    commands = variants['feishu']['commands']
    sections = sorted({section(k, v) for k, v in commands.items()} | {'events', 'extensions', 'packaging'})
    coverage = {'schema_version': 1, 'reference': {
        'revision': git(reference, 'rev-parse', 'HEAD'),
        'local_changes': git(reference, 'status', '--porcelain=v1'),
        'distribution': '@larksuite/cli', 'binary': 'lark-cli', 'license': 'MIT',
        'metadata_sha256': digest(metadata), 'metadata_services': sorted(s['name'] for s in parsed['services']),
        'metadata_source': 'https://open.feishu.cn/api/tools/open/api_definition?protocol=meta',
        'metadata_note': 'Frozen build catalog fetched by target canonical build; absent in clean reference. Both brands use this exact catalog; remote overlay updates are separate runtime behavior.',
        'go_version': subprocess.check_output(['go', 'version']).decode().strip()},
        'sections': sections, 'counts': {'commands_including_root': len(commands),
        'shortcuts': len(variants['feishu']['shortcuts']), 'generated_methods': len(methods),
        'events': len(variants['feishu']['events']), 'source_contract_files': len(sources['files']),
        'test_files': len(sources['test_files']), 'conditional_files': len(sources['conditions'])},
        'commands': {path: {'target': ('feishu-cli-next ' + path).strip(),
            'section': section(path, value), 'owner': section(path, value),
            'contract': 'registrations.json.gz#/feishu/commands/' + path,
            'implementation': 'shared-reference-source', 'verification': 'unverified',
            'test_evidence': 'sources.json#/test_files'} for path, value in sorted(commands.items())},
        'non_command_sections': {
            'events': {'contract': 'registrations.json.gz#/feishu/events', 'owner': 'events', 'verification': 'unverified'},
            'extensions': {'contract': 'sources.json#/public_symbol_index', 'owner': 'extensions', 'verification': 'unverified'},
            'packaging': {'contract': 'sources.json#/conditions', 'owner': 'packaging', 'verification': 'unverified'}},
        'verification_note': 'Presence, schema equality, and source equality are not functional or live verification. Section owners must supply operation-specific suites and evidence.'}
    save(out / 'coverage.json', coverage)
    save(out / 'registrations.json.gz', variants)
    save(out / 'sources.json', sources)
    # Preserve exact downloaded bytes, including original formatting, for rebuilds.
    (out / 'meta_data.json.gz').write_bytes(gzip.compress(metadata, mtime=0))
    return coverage['counts']


def compare_contract(reference, target, prefix=''):
    """Allow additions, but report missing or changed existing contracts."""
    errors = []
    if isinstance(reference, dict) and isinstance(target, dict):
        for key, value in reference.items():
            path = prefix + '/' + key
            if key not in target:
                errors.append({'path': path, 'kind': 'missing'})
            else:
                errors.extend(compare_contract(value, target[key], path))
    elif reference != target:
        errors.append({'path': prefix, 'kind': 'changed'})
    return errors


def check(reference_dir, target):
    frozen = read(reference_dir / 'registrations.json.gz')
    metadata = gzip.decompress((reference_dir / 'meta_data.json.gz').read_bytes())
    coverage = read(reference_dir / 'coverage.json')
    if digest(metadata) != coverage['reference']['metadata_sha256']:
        raise ValueError('frozen metadata checksum does not match coverage.json')
    errors = []
    for brand, reference in frozen.items():
        current = export(target, metadata, brand)
        errors.extend(compare_contract(reference, current, '/' + brand))
    sources = read(reference_dir / 'sources.json')
    for name, expected in sources['files'].items():
        path = target / name
        if not path.is_file():
            errors.append({'path': name, 'kind': 'missing_source'})
        elif digest(path.read_bytes()) != expected:
            errors.append({'path': name, 'kind': 'changed_source_requires_review'})
    return {'schema_version': 1, 'status': 'pass' if not errors else 'review_required',
            'differences': errors, 'live_verification': 'not_run'}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='action', required=True)
    frozen = sub.add_parser('freeze')
    frozen.add_argument('--reference', type=Path, required=True)
    frozen.add_argument('--metadata', type=Path, required=True)
    frozen.add_argument('--out', type=Path, default=ROOT / 'docs/reference')
    checker = sub.add_parser('check')
    checker.add_argument('--reference-dir', type=Path, default=ROOT / 'docs/reference')
    checker.add_argument('--target', type=Path, default=ROOT)
    args = parser.parse_args(argv)
    try:
        result = freeze(args.reference.resolve(), args.metadata.resolve(), args.out.resolve()) if args.action == 'freeze' else check(args.reference_dir.resolve(), args.target.resolve())
        print(json.dumps(result, sort_keys=True))
        return 1 if result.get('status') == 'review_required' else 0
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        print(json.dumps({'status': 'error', 'error': str(error)}), file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
