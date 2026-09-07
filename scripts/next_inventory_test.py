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

    def test_build_context_records_go_target_and_host(self):
        values = {'GOOS': 'linux', 'GOARCH': 'arm64', 'GOHOSTOS': 'linux', 'GOHOSTARCH': 'arm64'}
        with mock.patch.object(inventory.subprocess, 'check_output', side_effect=[json.dumps(values).encode(), b'go version fixture']):
            self.assertEqual(inventory.build_context(), {**values, 'go_version': 'go version fixture'})

    def test_exporter_selects_next_only_when_requested(self):
        legacy = inventory.exporter_source('legacy')
        self.assertNotIn('rootcmd.WithAgentMode()', legacy)
        self.assertEqual(inventory.exporter_source('next').count('rootcmd.WithAgentMode()'), 1)
        with self.assertRaises(ValueError):
            inventory.exporter_source('invalid')

    def test_exporter_configures_actual_startup_brand_and_captures_guard(self):
        for mode in ('next', 'legacy'):
            source = inventory.exporter_source(mode)
            self.assertIn('WithStartupBrand(core.LarkBrand(os.Getenv("LARKSUITE_CLI_BRAND")))', source)
            self.assertIn('_ "github.com/larksuite/cli/extension/credential/env"', source)
            self.assertIn('"disable_flag_parsing": c.DisableFlagParsing', source)
        reference = {'commands': {'apps': {'hidden': True, 'disable_flag_parsing': True}}}
        target = {'commands': {'apps': {'hidden': False, 'disable_flag_parsing': False}}}
        self.assertEqual(len(inventory.compare_contract(reference, target)), 2)

    def test_brand_supplement_rejects_unconfigured_lark_capture(self):
        bootstrap = {'commands': {'apps': {'hidden': False, 'disable_flag_parsing': False}}}
        with self.assertRaises(RuntimeError):
            inventory.validate_brand_surface('lark', bootstrap)
        inventory.validate_brand_surface('feishu', bootstrap)
        guarded = {'commands': {'apps': {'hidden': True, 'disable_flag_parsing': True}}}
        inventory.validate_brand_surface('lark', guarded)
        with self.assertRaises(RuntimeError):
            inventory.validate_brand_surface('feishu', guarded)

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
                configured = root / 'configured.json.gz'
                inventory.save(configured, {'feishu': next_surface})
                alternate = inventory.check(root, root, 'next', configured)
                self.assertEqual(alternate['status'], 'pass')
                self.assertEqual(alternate['reference_registrations_sha256'], inventory.digest(configured.read_bytes()))
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
                self.assertEqual(check.call_args.args[2], surface)

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

    def gate_fixture(self):
        before = {'commands': {'api': {'flags': {'count': {'type': 'int'}},
                                      'risk': 'write', 'hidden': False}},
                  'shortcuts': {}, 'events': {'key': {}}}
        after = json.loads(json.dumps(before))
        after['commands']['api']['risk'] = 'read'
        after['commands']['guide'] = {'flags': {'local': {'type': 'bool'}}}
        changes = inventory.registration_changes(before, after, '/feishu')
        report = {'surface': 'next', 'reference_registrations_sha256': 'reference',
                  'metadata_sha256': 'metadata', 'tracked_source_consistent': True,
                  'counts': {'feishu': {}, 'lark': {}},
                  'reference_presence': {'feishu': inventory.presence(before, after)},
                  'registration_changes': changes, 'status': 'review_required',
                  'differences': [{'kind': 'changed_source_requires_review', 'path': 'cmd/runtime.go'}]}
        mapping = {**{key: report[key] for key in ('surface', 'reference_registrations_sha256', 'metadata_sha256')},
                   'candidate_revision': 'reviewed', 'registration_changes': changes}
        return before, after, report, mapping

    def test_capability_gate_preserves_separate_source_review(self):
        _, _, report, mapping = self.gate_fixture()
        original = json.loads(json.dumps(report))
        self.assertEqual(inventory.capability_gate(report, mapping)['status'], 'pass')
        self.assertEqual(report, original)
        self.assertEqual(report['status'], 'review_required')

    def test_capability_gate_rejects_missing_source_without_approving_changed_source(self):
        _, _, report, mapping = self.gate_fixture()
        report['differences'].append({'kind': 'missing_source', 'path': 'extension/provider.go'})
        self.assertEqual(inventory.capability_gate(report, mapping)['status'], 'fail')

    def test_capability_gate_requires_exact_metadata_and_added_subtree(self):
        before, after, report, mapping = self.gate_fixture()
        for mutation in ('risk', 'new_flag_type', 'unmapped_addition', 'missing_addition'):
            candidate = json.loads(json.dumps(after))
            if mutation == 'risk':
                candidate['commands']['api']['risk'] = 'delete'
            elif mutation == 'new_flag_type':
                candidate['commands']['guide']['flags']['local']['type'] = 'string'
            elif mutation == 'unmapped_addition':
                candidate['extra'] = True
            else:
                del candidate['commands']['guide']
            report['registration_changes'] = inventory.registration_changes(before, candidate, '/feishu')
            with self.subTest(mutation=mutation):
                self.assertEqual(inventory.capability_gate(report, mapping)['status'], 'fail')

    def test_capability_gate_cannot_map_away_reference_loss_flags_or_guard(self):
        before, after, report, mapping = self.gate_fixture()
        for mutation in ('missing_command', 'flag_type', 'brand_guard', 'event'):
            candidate = json.loads(json.dumps(after))
            if mutation == 'missing_command':
                del candidate['commands']['api']
            elif mutation == 'flag_type':
                candidate['commands']['api']['flags']['count']['type'] = 'string'
            elif mutation == 'brand_guard':
                candidate['commands']['api']['hidden'] = True
            else:
                del candidate['events']['key']
            changes = inventory.registration_changes(before, candidate, '/feishu')
            report['registration_changes'] = mapping['registration_changes'] = changes
            report['reference_presence']['feishu'] = inventory.presence(before, candidate)
            with self.subTest(mutation=mutation):
                self.assertEqual(inventory.capability_gate(report, mapping)['status'], 'fail')

    def test_capability_gate_rejects_wrong_context_or_mixed_snapshot(self):
        for field, value in (('surface', 'legacy'), ('reference_registrations_sha256', 'bootstrap'),
                             ('metadata_sha256', 'unfrozen'), ('tracked_source_consistent', False),
                             ('counts', {'feishu': {}})):
            _, _, report, mapping = self.gate_fixture()
            report[field] = value
            with self.subTest(field=field):
                self.assertEqual(inventory.capability_gate(report, mapping)['status'], 'fail')

    def test_gate_cli_saves_report_on_success_rejection_and_collection_error(self):
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / 'report.json'
            mapping_path = Path(temporary) / 'mapping.json'
            _, _, report, mapping = self.gate_fixture()
            inventory.save(mapping_path, mapping)
            for fail in (False, True):
                report['tracked_source_consistent'] = not fail
                with mock.patch.object(inventory, 'check', return_value=report), mock.patch('builtins.print'):
                    code = inventory.main(['gate', '--mapping', str(mapping_path), '--out', str(output)])
                self.assertEqual(code, int(fail))
                saved = inventory.read(output)
                self.assertEqual(saved['status'], 'review_required')
                self.assertEqual(saved['capability_gate']['status'], 'fail' if fail else 'pass')
            with mock.patch.object(inventory, 'check', side_effect=RuntimeError('export failed')), mock.patch('builtins.print'):
                code = inventory.main(['gate', '--mapping', str(mapping_path), '--out', str(output)])
            self.assertEqual(code, 2)
            self.assertEqual(inventory.read(output), {'status': 'error', 'error': 'export failed'})

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
