import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
from read_fixture import dispatcher, load_runtime, matches, subset, typed_equal, verify
from section_fixture import SectionFixture
from codex_runner import cli_environment


class ReadFixtureTests(unittest.TestCase):
    def test_selector_predicates_are_typed_and_do_not_ignore_identity(self):
        self.assertFalse(typed_equal(0, False))
        self.assertFalse(subset({'a': False}, {'a': 0}))
        rule = {'identities': ['user'], 'query': {'kind': ['one', 'two']},
                'query_keys': ['kind'], 'forbidden_body_keys': ['scope'],
                'optional_body': {'revision': [-1, 1]}, 'body_contains': {'ids': ['a']}}
        body = {'ids': ['a'], 'revision': 1}
        self.assertTrue(matches(rule, {'kind': ['one']}, body, 'user'))
        for query, data, identity in [({'kind': ['one', 'two']}, body, 'user'),
                                      ({'kind': ['one'], 'cursor': ['']}, body, 'user'),
                                      ({'kind': ['one']}, dict(body, scope={}), 'user'),
                                      ({'kind': ['one']}, dict(body, revision=True), 'user'),
                                      ({'kind': ['one']}, body, 'bot')]:
            self.assertFalse(matches(rule, query, data, identity))

    def test_models_receive_only_public_fixture_and_type_schema(self):
        for path in (ROOT / 'evaluation/runtime').glob('*.json'):
            data = load_runtime(path.stem)
            self.assertEqual(set(data['task']['fixture']), {'scenario', 'answer_schema'})
            encoded = json.dumps(data['task']['fixture']['answer_schema'])
            self.assertNotIn('"const"', encoded)
            self.assertNotIn('"enum"', encoded)
            routes = {route['id'] for route in data['routes']}
            self.assertTrue(all(set(group) <= routes for group in data['task']['expected']['required_reads']))

    def test_verifier_needs_observation_and_exact_typed_answer(self):
        runtime = load_runtime('application')
        with tempfile.TemporaryDirectory() as directory, SectionFixture(runtime['task']['fixture']['scenario'], dispatcher(runtime)) as fixture:
            workspace = Path(directory)
            request = {'workspace': directory, 'fixture': {'runtime_section': 'application'}, 'expected': runtime['task']['expected']}
            answer = runtime['task']['expected']['answer']
            (workspace / 'answer.json').write_text(json.dumps(answer))
            fixture.write_receipt(workspace)
            failures = []
            self.assertFalse(verify(request, failures))
            self.assertEqual(failures, ["missing_request_evidence"])
            fixture.records.append({'route': 'application_slash_command_list', 'status': 200, 'returned_ids': []})
            fixture.write_receipt(workspace)
            self.assertTrue(verify(request))
            journal = workspace / 'feishu-config/next/receipts' / ('a' * 32 + '.json')
            journal.parent.mkdir(parents=True)
            journal.write_text('{}')
            self.assertTrue(verify(request), 'local attempt receipt is not auth configuration')
            config = workspace / 'feishu-config/profile.json'
            config.write_text('{}')
            failures = []
            self.assertFalse(verify(request, failures), 'actual config changes stay forbidden')
            self.assertEqual(failures, ['config_changed'])
            config.unlink()
            (workspace / 'answer.json').write_text(json.dumps(dict(answer, enabled_command='wrong')))
            failures = []
            self.assertFalse(verify(request, failures))
            self.assertEqual(failures, ['answer_mismatch'])
            (workspace / 'answer.json').write_text(json.dumps(answer))
            fixture.records.append({'route': 'rejected', 'status': 403})
            fixture.write_receipt(workspace)
            failures = []
            self.assertFalse(verify(request, failures))
            self.assertEqual(failures, ['request_outside_fixture'])
            (workspace / 'answer.json').write_text(json.dumps(answer))
            fixture.records.append({'route': 'rejected', 'status': 403, 'returned_ids': []})
            fixture.write_receipt(workspace)
            self.assertFalse(verify(request))


@unittest.skipUnless(os.environ.get('EVAL_READ_BINARIES'), 'set archived authsidecar binary paths in EVAL_READ_BINARIES')
class ReadBinaryProcessTests(unittest.TestCase):
    def test_all_runtime_examples_and_independent_proofs(self):
        for path in sorted((ROOT / 'evaluation/runtime').glob('*.json')):
            runtime = load_runtime(path.stem)
            for binary in json.loads(os.environ['EVAL_READ_BINARIES']):
                with self.subTest(section=path.stem, binary=Path(binary).name), tempfile.TemporaryDirectory() as directory, SectionFixture(runtime['task']['fixture']['scenario'], dispatcher(runtime)) as fixture:
                    workspace = Path(directory)
                    for name in ('home', 'tmp', 'feishu-config'):
                        (workspace / name).mkdir()
                    env = {**cli_environment(workspace), **fixture.environment()}
                    for args in runtime['examples']:
                        result = subprocess.run([binary, *args], cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                        self.assertEqual(result.returncode, 0, str(args) + '\n' + result.stderr)
                    fixture.write_receipt(workspace)
                    (workspace / 'answer.json').write_text(json.dumps(runtime['task']['expected']['answer']))
                    request = {'workspace': directory, 'fixture': {'runtime_section': path.stem}, 'expected': runtime['task']['expected']}
                    self.assertTrue(verify(request), json.dumps(fixture.receipt()))


if __name__ == '__main__':
    unittest.main()
