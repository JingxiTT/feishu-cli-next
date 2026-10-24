#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

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
BUILD_FILES = ('Makefile', 'next.mk', '.goreleaser.yml', 'go.mod', 'go.sum', 'package.json', 'build.sh', 'main.go', 'main_authsidecar.go', 'main_noauthsidecar.go', 'content_embed.go', 'LICENSE')


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


def exporter_source(surface, committed_catalog=False):
    """Select the real distribution surface without importing next into upstream."""
    source = EXPORTER.read_text(encoding='utf-8')
    if surface == 'legacy' and not committed_catalog:
        return source
    if surface not in ('next', 'legacy'):
        raise ValueError('surface must be next or legacy')
    anchor = 'rootcmd.WithoutPlugins(), rootcmd.WithoutStrictMode(),'
    if source.count(anchor) != 1:
        raise ValueError('inventory exporter Build options changed; update surface selection')
    source = source.replace('"github.com/larksuite/cli/internal/registry"',
                            '"github.com/larksuite/cli/internal/apicatalog"\n\t"github.com/larksuite/cli/internal/meta"')
    source = source.replace('func main() {', 'func main() {\n    data, err := os.ReadFile("internal/registry/meta_data.json")\n    if err != nil { panic(err) }\n    metadata, err := meta.Parse(data)\n    if err != nil { panic(err) }\n    frozenCatalog := apicatalog.New(apicatalog.SourceEmbedded, metadata.Services)\n')
    source = source.replace('registry.EmbeddedServicesTyped()', 'frozenCatalog.Services()')
    source = source.replace('registry.EmbeddedCatalog()', 'frozenCatalog')
    if surface == 'next':
        source = source.replace(anchor, anchor + '\n\t\trootcmd.WithAgentMode(),')
    return source


def build_context():
    """Record the actual Go target and host used for local registration execution."""
    values = json.loads(subprocess.check_output(
        ['go', 'env', '-json', 'GOOS', 'GOARCH', 'GOHOSTOS', 'GOHOSTARCH']))
    values['go_version'] = subprocess.check_output(['go', 'version']).decode().strip()
    return values


def export(root, metadata, brand, surface='legacy'):
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
        (work / 'scripts/next_inventory_export.go').write_text(exporter_source(surface, (root / 'internal/registry/catalog/manifest.json').is_file()), encoding='utf-8')
        (work / 'internal/registry/meta_data.json').write_bytes(metadata)
        env = {k: v for k, v in os.environ.items() if not k.startswith(('LARKSUITE_CLI_', 'FEISHU_'))}
        env.update(LARKSUITE_CLI_CONFIG_DIR=str(Path(temporary) / 'config'),
                   LARKSUITE_CLI_REMOTE_META='off', LARKSUITE_CLI_BRAND=brand,
                   LARKSUITE_CLI_APP_ID='inventory-synthetic-app',
                   LARKSUITE_CLI_APP_SECRET='inventory-synthetic-secret',
                   HTTP_PROXY='http://127.0.0.1:9', HTTPS_PROXY='http://127.0.0.1:9',
                   ALL_PROXY='http://127.0.0.1:9', NO_PROXY='',
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
    committed_catalog = (reference / 'internal/registry/catalog/manifest.json').is_file()
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
        'metadata_source': ('internal/registry/catalog/manifest.json + services/*.json' if committed_catalog else 'https://open.feishu.cn/api/tools/open/api_definition?protocol=meta'),
        'metadata_note': ('Exact service JSON from the official committed snapshot, assembled into the typed registry document for comparison; both brands use this catalog.' if committed_catalog else 'Frozen build catalog fetched by target canonical build; absent in clean reference. Both brands use this exact catalog; remote overlay updates are separate runtime behavior.'),
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


def additions(reference, target, prefix=''):
    """List new dictionary contracts separately from compatibility failures."""
    found = []
    if isinstance(reference, dict) and isinstance(target, dict):
        for key, value in target.items():
            path = prefix + '/' + key
            if key not in reference:
                found.append(path)
            else:
                found.extend(additions(reference[key], value, path))
    return sorted(found)


def registration_changes(reference, target, prefix=''):
    """Bind each interface delta to its exact values, including new subtrees."""
    changes = []
    if isinstance(reference, dict) and isinstance(target, dict):
        for key in sorted(reference.keys() | target.keys()):
            path = prefix + '/' + key
            if key not in target:
                changes.append({'path': path, 'kind': 'missing',
                                'reference_sha256': digest(canonical(reference[key]))})
            elif key not in reference:
                changes.append({'path': path, 'kind': 'added',
                                'candidate_sha256': digest(canonical(target[key]))})
            else:
                changes.extend(registration_changes(reference[key], target[key], path))
    elif reference != target:
        changes.append({'path': prefix, 'kind': 'changed',
                        'reference_sha256': digest(canonical(reference)),
                        'candidate_sha256': digest(canonical(target))})
    return changes


def platform_reference(reference_dir, context):
    """Choose only the three documented platform profiles, never a generic fallback."""
    goos = context['GOOS']
    if goos not in ('darwin', 'linux', 'windows') or goos != context['GOHOSTOS']:
        raise ValueError('capability gate requires matching Go target/host OS: darwin, linux, or windows')
    profiles_path = reference_dir / 'configured-brand-platforms.json'
    profile = read(profiles_path)['platforms'][goos]
    path = reference_dir / profile['reference_file']
    if digest(path.read_bytes()) != profile['reference_sha256']:
        raise ValueError('configured platform reference checksum mismatch')
    return path, {**profile, 'GOOS': goos,
                  'profiles_sha256': digest(profiles_path.read_bytes())}


def capability_gate(report, mapping):
    """Exact interface reconciliation; source review remains independent."""
    failures = []
    for field in ('surface', 'reference_registrations_sha256', 'metadata_sha256'):
        if report.get(field) != mapping.get(field):
            failures.append({'kind': 'context_mismatch', 'field': field})
    if report.get('surface') != 'next' or set(report.get('counts', {})) != {'feishu', 'lark'}:
        failures.append({'kind': 'configured_next_brands_required'})
    if not report.get('tracked_source_consistent'):
        failures.append({'kind': 'mixed_source_snapshot'})
    for brand, value in report.get('reference_presence', {}).items():
        for field in ('missing_commands', 'missing_shortcuts', 'missing_event_keys',
                      'removed_flags', 'changed_existing_flags'):
            if value.get(field):
                failures.append({'kind': field, 'brand': brand, 'details': value[field]})
        if not value.get('raw_api_present'):
            failures.append({'kind': 'raw_api_missing', 'brand': brand})
    for row in report.get('differences', []):
        if row['kind'] not in ('changed', 'missing', 'changed_source_requires_review'):
            failures.append({'kind': 'unresolved_inventory_difference', 'difference': row})
    expected = {canonical(row) for row in mapping['registration_changes']}
    actual = {canonical(row) for row in report['registration_changes']}
    for row in sorted(actual - expected):
        failures.append({'kind': 'unmapped_registration_change', 'change': json.loads(row)})
    for row in sorted(expected - actual):
        failures.append({'kind': 'mapped_contract_not_present', 'change': json.loads(row)})
    # A mapping cannot waive removal, existing command flags, or brand guards.
    for row in report['registration_changes']:
        if (row['kind'] == 'missing' or
                (row['kind'] == 'changed' and '/commands/' in row['path'] and
                 ('/flags/' in row['path'] or row['path'].endswith(('/hidden', '/disable_flag_parsing'))))):
            failures.append({'kind': 'protected_reference_contract', 'change': row})
    return {'status': 'fail' if failures else 'pass', 'failures': failures,
            'mapping_revision': mapping['candidate_revision'],
            'checked_registration_changes': len(report['registration_changes']),
            'semantic_review': 'separate; source differences and original report status remain unchanged',
            'live_verification': 'not_run'}


def repository_state(root):
    """Bind evidence to tracked working bytes, not only a possibly stale HEAD."""
    files = {}
    for name in tracked(root):
        if name:
            path = root / name
            files[name] = digest(path.read_bytes()) if path.is_file() else None
    return {'revision': git(root, 'rev-parse', 'HEAD'),
            'local_changes': git(root, 'status', '--porcelain=v1'),
            'tracked_files': len(files), 'tracked_tree_sha256': digest(canonical(files)),
            'scope': 'tracked working files only; untracked files are listed in local_changes but not compiled by this exporter'}


def presence(reference, current):
    removed_flags, changed_flags = [], []
    for path, before in reference['commands'].items():
        after = current['commands'].get(path, {})
        for name, flag in before.get('flags', {}).items():
            candidate = after.get('flags', {}).get(name)
            if candidate is None:
                removed_flags.append({'command': path, 'flag': name})
            elif flag != candidate:
                changed_flags.append({'command': path, 'flag': name,
                                      'reference': flag, 'candidate': candidate})
    return {'missing_commands': sorted(set(reference['commands']) - set(current['commands'])),
            'missing_shortcuts': sorted(set(reference.get('shortcuts', {})) - set(current.get('shortcuts', {}))),
            'missing_event_keys': sorted(set(reference.get('events', {})) - set(current.get('events', {}))),
            'removed_flags': removed_flags, 'changed_existing_flags': changed_flags,
            'raw_api_present': 'api' in current['commands']}


def check(reference_dir, target, surface='next', reference_registrations=None):
    registration_path = reference_registrations or reference_dir / 'registrations.json.gz'
    frozen = read(registration_path)
    metadata = gzip.decompress((reference_dir / 'meta_data.json.gz').read_bytes())
    coverage = read(reference_dir / 'coverage.json')
    if digest(metadata) != coverage['reference']['metadata_sha256']:
        raise ValueError('frozen metadata checksum does not match coverage.json')
    errors, added, counts, new_commands, new_root_flags = [], [], {}, {}, {}
    states, retained, changes = {}, {}, []
    initial_state = repository_state(target)
    for brand, reference in frozen.items():
        states[brand] = {'before': repository_state(target)}
        current = export(target, metadata, brand, surface)
        states[brand]['after'] = repository_state(target)
        retained[brand] = presence(reference, current)
        changes.extend(registration_changes(reference, current, '/' + brand))
        errors.extend(compare_contract(reference, current, '/' + brand))
        added.extend(additions(reference, current, '/' + brand))
        new_commands[brand] = {k: v for k, v in current['commands'].items() if k not in reference['commands']}
        new_root_flags[brand] = {k: v for k, v in current['commands']['']['flags'].items()
                                 if k not in reference['commands']['']['flags']}
        counts[brand] = {'reference_commands': len(reference['commands']),
                         'candidate_commands': len(current['commands']),
                         'reference_shortcuts': len(reference.get('shortcuts', {})),
                         'candidate_shortcuts': len(current.get('shortcuts', {})),
                         'reference_event_keys': len(reference.get('events', {})),
                         'candidate_event_keys': len(current.get('events', {}))}
    sources = read(reference_dir / 'sources.json')
    for name, expected in sources['files'].items():
        path = target / name
        if not path.is_file():
            errors.append({'path': name, 'kind': 'missing_source'})
        elif digest(path.read_bytes()) != expected:
            errors.append({'path': name, 'kind': 'changed_source_requires_review',
                           'reference_sha256': expected, 'candidate_sha256': digest(path.read_bytes())})
    final_state = repository_state(target)
    observations = [initial_state, final_state] + [v for pair in states.values() for v in pair.values()]
    consistent = len({v['tracked_tree_sha256'] for v in observations}) == 1
    if not consistent:
        errors.append({'path': '/', 'kind': 'tracked_source_changed_during_check'})
    return {'schema_version': 1, 'status': 'pass' if not errors else 'review_required',
            'surface': surface, 'reference_surface': 'legacy', 'counts': counts,
            'build_context': build_context(),
            'candidate_brand_context': 'explicit startup brand and isolated synthetic environment account',
            'reference_registrations': str(registration_path),
            'reference_context_notice': ('Explicit alternate registration supplement; inspect its capture context.' if reference_registrations else 'Original frozen bootstrap inventory; use the configured-brand supplement for runtime brand conditions.'),
            'reference_registrations_sha256': digest(registration_path.read_bytes()),
            'exporter_source_sha256': digest(exporter_source(surface, (target / 'internal/registry/catalog/manifest.json').is_file()).encode()),
            'metadata_sha256': digest(metadata), 'registration_changes': changes,
            'candidate_revision': initial_state['revision'],
            'candidate_state_before': initial_state, 'candidate_state_after': final_state,
            'brand_snapshot_observations': states, 'tracked_source_consistent': consistent,
            'reference_presence': retained,
            'new_commands': new_commands, 'new_root_flags': new_root_flags,
            'differences': errors, 'additions': sorted(added), 'live_verification': 'not_run'}



def validate_brand_surface(brand, values):
    commands = values['commands']
    apps = [value for path, value in commands.items() if path == 'apps' or path.startswith('apps ')]
    if not apps or 'apps' not in commands:
        raise RuntimeError('configured Apps registration missing')
    if brand == 'lark' and not all(value['hidden'] and value['disable_flag_parsing'] for value in apps):
        raise RuntimeError('configured Lark Apps deny subtree was not captured')
    if brand == 'feishu' and (commands['apps']['hidden'] or commands['apps']['disable_flag_parsing']):
        raise RuntimeError('configured Feishu Apps registration is unexpectedly guarded')


def capture_brands(reference_dir, reference, output):
    coverage = read(reference_dir / 'coverage.json')
    if git(reference, 'rev-parse', 'HEAD') != coverage['reference']['revision']:
        raise ValueError('brand supplement must use the frozen reference revision')
    if source_inventory(reference)['files'] != read(reference_dir / 'sources.json')['files']:
        raise ValueError('brand supplement reference bytes differ from frozen source inventory')
    metadata = gzip.decompress((reference_dir / 'meta_data.json.gz').read_bytes())
    if digest(metadata) != coverage['reference']['metadata_sha256']:
        raise ValueError('frozen metadata checksum does not match coverage.json')
    variants = {brand: export(reference, metadata, brand) for brand in ('feishu', 'lark')}
    for brand, values in variants.items():
        validate_brand_surface(brand, values)
    save(output, variants)
    return {'schema_version': 1, 'status': 'captured', 'build_context': build_context(),
            'reference_revision': coverage['reference']['revision'],
            'reference_local_changes': git(reference, 'status', '--porcelain=v1'),
            'metadata_sha256': digest(metadata), 'registrations': str(output),
            'registrations_sha256': digest(output.read_bytes()),
            'exporter_source_sha256': digest(exporter_source('legacy', (reference / 'internal/registry/catalog/manifest.json').is_file()).encode()),
            'brand_context': 'Production environment credential provider plus explicit WithStartupBrand and an isolated synthetic app account; no command executes; remote metadata disabled and external proxies point to an unreachable loopback port.',
            'frozen_snapshot_boundary': 'Existing registrations.json.gz is an unrestricted bootstrap inventory: setting brand environment alone did not configure the legacy registration guard. This supplement preserves it and captures configured runtime brand conditions.',
            'hidden_commands': {brand: sorted(p for p, c in value['commands'].items() if c['hidden']) for brand, value in variants.items()}}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='action', required=True)
    frozen = sub.add_parser('freeze')
    frozen.add_argument('--reference', type=Path, required=True)
    frozen.add_argument('--metadata', type=Path, required=True)
    frozen.add_argument('--out', type=Path, default=ROOT / 'baseline')
    checker = sub.add_parser('check')
    checker.add_argument('--reference-dir', type=Path, default=ROOT / 'baseline')
    checker.add_argument('--target', type=Path, default=ROOT)
    checker.add_argument('--surface', choices=('next', 'legacy'), default='next',
                         help='candidate distribution Build options (default: next); use legacy for upstream self-reconciliation')
    checker.add_argument('--reference-registrations', type=Path, help='explicit configured-brand supplement; original frozen bootstrap snapshot stays unchanged')
    gate = sub.add_parser('gate', help='configured-brand interface guard; source review stays separate')
    gate.add_argument('--reference-dir', type=Path, default=ROOT / 'baseline')
    gate.add_argument('--target', type=Path, default=ROOT)
    gate.add_argument('--mapping', type=Path, default=ROOT / 'baseline/next-interface-mapping.json.gz')
    gate.add_argument('--out', type=Path, required=True)
    brands = sub.add_parser('capture-brands')
    brands.add_argument('--reference-dir', type=Path, default=ROOT / 'baseline')
    brands.add_argument('--reference', type=Path, required=True)
    brands.add_argument('--out', type=Path, required=True)
    args = parser.parse_args(argv)
    try:
        if args.action == 'freeze':
            result = freeze(args.reference.resolve(), args.metadata.resolve(), args.out.resolve())
        elif args.action == 'gate':
            reference_dir = args.reference_dir.resolve()
            context = build_context()
            reference_path, profile = platform_reference(reference_dir, context)
            result = check(reference_dir, args.target.resolve(), 'next', reference_path)
            if result['build_context'] != context:
                raise ValueError('Go build context changed during capability collection')
            result['platform_reference'] = profile
            mapping = read(args.mapping)
            # Only the exact reference hash changes by platform. All intentional
            # interface delta values remain the reviewed, shared mapping.
            mapping = {**mapping, 'reference_registrations_sha256': profile['reference_sha256']}
            result['capability_gate'] = capability_gate(result, mapping)
            result['capability_gate']['mapping_sha256'] = digest(args.mapping.read_bytes())
            result['capability_gate']['reference_provenance'] = profile.get('provenance')
            result['capability_gate']['native_reference_capture'] = profile.get('native_reference_capture')
            save(args.out, result)
            print(json.dumps({'report': str(args.out), **result['capability_gate']}, sort_keys=True))
            return 0 if result['capability_gate']['status'] == 'pass' else 1
        elif args.action == 'capture-brands':
            result = capture_brands(args.reference_dir.resolve(), args.reference.resolve(), args.out.resolve())
        else:
            result = check(args.reference_dir.resolve(), args.target.resolve(), args.surface,
                           args.reference_registrations.resolve() if args.reference_registrations else None)
        print(json.dumps(result, sort_keys=True))
        return 1 if result.get('status') == 'review_required' else 0
    except (OSError, ValueError, KeyError, TypeError, RuntimeError, subprocess.SubprocessError) as error:
        failure = {'status': 'error', 'error': str(error)}
        if args.action == 'gate':
            save(args.out, failure)
        print(json.dumps(failure), file=sys.stderr)
        return 2


if __name__ == '__main__':
    sys.exit(main())
