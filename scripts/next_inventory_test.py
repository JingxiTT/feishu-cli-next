"""Synthetic inventory unit tests; process E2E is separately selectable."""
import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

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
