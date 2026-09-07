"""Synthetic inventory unit tests; process E2E is separately selectable."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

MODULE_PATH = Path(__file__).with_name('next_inventory.py')
spec = importlib.util.spec_from_file_location('inventory', MODULE_PATH)
inventory = importlib.util.module_from_spec(spec)
spec.loader.exec_module(inventory)


class InventoryTests(unittest.TestCase):
    def test_generated_nested_methods_preserve_types_and_options(self):
        fixture = {'services': [{'name': 'test', 'resources': {'parent': {
            'resources': {'child': {'methods': {'create': {'requestBody': {
                'value': {'type': 'array', 'properties': {'id': {'type': 'string', 'enum': ['01', '02']}}}}}}}}}}}]}
        methods = inventory.metadata_methods(fixture)
        self.assertEqual(list(methods), ['test parent child create'])
        self.assertEqual(methods['test parent child create']['requestBody']['value']['properties']['id']['enum'], ['01', '02'])
        with self.assertRaises(ValueError):
            inventory.metadata_methods({'services': []})

    def test_reconciliation_detects_missing_command_flag_schema_and_type(self):
        reference = {'commands': {'test': {'flags': {'data': {'type': 'string'}}}},
                     'schemas': {'value': {'type': 'string'}}, 'event': 'stream'}
        target = {'commands': {'test': {'flags': {}}}, 'schemas': {'value': {'type': 'number'}}}
        self.assertEqual(inventory.compare_contract(reference, target), [
            {'path': '/commands/test/flags/data', 'kind': 'missing'},
            {'path': '/schemas/value/type', 'kind': 'changed'},
            {'path': '/event', 'kind': 'missing'}])
        self.assertEqual(inventory.compare_contract(reference, dict(reference, added=True)), [])

    def test_conditional_surfaces_and_local_changes_captured(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'extension').mkdir()
            source = root / 'extension/public_windows.go'
            source.write_text('//go:build windows && authsidecar\npackage extension\ntype Public struct { Value string }\n')
            subprocess.run(['git', 'init', '-q', temporary], check=True)
            subprocess.run(['git', '-C', temporary, 'add', '.'], check=True)
            first = inventory.source_inventory(root)
            self.assertEqual(first['conditions'], {'extension/public_windows.go': 'windows && authsidecar'})
            self.assertEqual(first['public_symbol_index'], {'extension/public_windows.go': ['Public']})
            source.write_text(source.read_text().replace('string', 'int'))
            self.assertNotEqual(inventory.source_inventory(root)['files'], first['files'])

    def test_gzip_deterministic_and_roundtrips_unicode(self):
        with tempfile.TemporaryDirectory() as temporary:
            a, b = Path(temporary) / 'a.json.gz', Path(temporary) / 'b.json.gz'
            fixture = {'text': '飞书', 'values': ['', None, '001', '=literal']}
            inventory.save(a, fixture)
            inventory.save(b, fixture)
            self.assertEqual(a.read_bytes(), b.read_bytes())
            self.assertEqual(inventory.read(a), fixture)

    def test_exporter_selects_next_only_when_requested(self):
        legacy = inventory.exporter_source('legacy')
        self.assertNotIn('rootcmd.WithAgentMode()', legacy)
        self.assertEqual(inventory.exporter_source('next').count('rootcmd.WithAgentMode()'), 1)
        with self.assertRaises(ValueError):
            inventory.exporter_source('invalid')

    def test_next_additions_do_not_hide_removed_legacy_flags(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            metadata = b'{"services": []}'
            (root / 'meta_data.json.gz').write_bytes(inventory.gzip.compress(metadata))
            inventory.save(root / 'coverage.json', {'reference': {'metadata_sha256': inventory.digest(metadata)}})
            inventory.save(root / 'sources.json', {'files': {}})
            legacy = {'commands': {'': {'flags': {'identity': {'type': 'string', 'default': 'user'}}}}}
            next_surface = {'commands': {'': {'flags': {'result-fields': {'type': 'string', 'default': ''}}}, 'guide': {}}}
            inventory.save(root / 'registrations.json.gz', {'feishu': legacy})
            def fake_export(target, metadata, brand, surface):
                return next_surface if surface == 'next' else legacy
            with mock.patch.object(inventory, 'export', side_effect=fake_export) as export, mock.patch.object(inventory, 'git', return_value='fixture-revision'):
                result = inventory.check(root, root)
                self.assertEqual(export.call_args.args[-1], 'next')
                self.assertEqual(result['surface'], 'next')
                self.assertEqual(result['status'], 'review_required')
                self.assertIn({'path': '/feishu/commands//flags/identity', 'kind': 'missing'}, result['differences'])
                self.assertIn('/feishu/commands//flags/result-fields', result['additions'])
                self.assertIn('/feishu/commands/guide', result['additions'])
                self.assertEqual(inventory.check(root, root, 'legacy')['status'], 'pass')
                states = [{'revision': 'fixture-revision', 'tracked_tree_sha256': value}
                          for value in ('before', 'before', 'after', 'after')]
                with mock.patch.object(inventory, 'repository_state', side_effect=states):
                    changed = inventory.check(root, root, 'legacy')
                self.assertFalse(changed['tracked_source_consistent'])
                self.assertIn({'path': '/', 'kind': 'tracked_source_changed_during_check'}, changed['differences'])

    def test_check_cli_defaults_next_and_reports_explicit_legacy(self):
        for arguments, surface in (([], 'next'), (['--surface', 'legacy'], 'legacy')):
            with mock.patch.object(inventory, 'check', return_value={'status': 'pass'}) as check:
                with mock.patch('builtins.print'):
                    self.assertEqual(inventory.main(['check', *arguments]), 0)
                self.assertEqual(check.call_args.args[-1], surface)

    def test_repository_state_records_dirty_tracked_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / 'source.go').write_text('before')
            def fake_git(root, *args):
                if args == ('ls-files', '-z'):
                    return 'source.go'
                if args == ('status', '--porcelain=v1'):
                    return ' M source.go\n?? new.go'
                return 'fixture-revision'
            with mock.patch.object(inventory, 'git', side_effect=fake_git):
                before = inventory.repository_state(root)
                (root / 'source.go').write_text('after')
                after = inventory.repository_state(root)
            self.assertEqual(before['revision'], after['revision'])
            self.assertNotEqual(before['tracked_tree_sha256'], after['tracked_tree_sha256'])
            self.assertIn('?? new.go', after['local_changes'])

    def test_presence_does_not_confuse_added_flags_with_preserved_types(self):
        before = {'commands': {'api': {'flags': {'count': {'type': 'int'}}}},
                  'shortcuts': {'domain +operation': {}}, 'events': {'event.key': {}}}
        after = {'commands': {'api': {'flags': {'count': {'type': 'string'}, 'extra': {}}}},
                 'shortcuts': {}, 'events': {}}
        result = inventory.presence(before, after)
        self.assertTrue(result['raw_api_present'])
        self.assertEqual(result['missing_commands'], [])
        self.assertEqual(result['missing_shortcuts'], ['domain +operation'])
        self.assertEqual(result['missing_event_keys'], ['event.key'])
        self.assertEqual(result['changed_existing_flags'][0]['flag'], 'count')

    def test_process_reports_bad_metadata_to_stderr_with_exit_two(self):
        with tempfile.TemporaryDirectory() as temporary:
            metadata = Path(temporary) / 'metadata.json'
            metadata.write_text('{broken')
            result = subprocess.run([sys.executable, str(MODULE_PATH), 'freeze',
                '--reference', temporary, '--metadata', str(metadata), '--out', temporary], capture_output=True)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(result.stdout, b'')
            self.assertEqual(json.loads(result.stderr)['status'], 'error')


if __name__ == '__main__':
    unittest.main()
