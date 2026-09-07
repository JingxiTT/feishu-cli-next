import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
from urllib.request import Request, urlopen
from urllib.error import HTTPError

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
from scenario_protocol import Reply, ScenarioBase
from scenario_fixture import ScenarioFixture, load_scenario, observe, probe_isolation
from section_fixture import signed_headers
from codex_runner import run_agent, configuration


class ScenarioIntegrationTests(unittest.TestCase):
    def test_binary_reply_state_and_unsigned_attempt_ledger(self):
        class Fixture(ScenarioBase):
            def dispatch(self, method, uri, body, headers):
                return Reply('download', body=b'\x00\xff', content_type='application/octet-stream',
                             headers={'Content-Disposition': 'attachment; filename="fixture.bin"'}, proof={'bytes': 2})
            def snapshot(self):
                return {'independent': True}
        with ScenarioFixture('controlled', Fixture()) as server:
            uri = '/fixture?private=not-retained'
            with urlopen(Request(server.environment()['LARKSUITE_CLI_AUTH_PROXY'] + uri,
                                 headers=signed_headers('GET', uri))) as response:
                self.assertEqual(response.read(), b'\x00\xff')
                self.assertEqual(response.headers['Content-Length'], '2')
            try:
                urlopen(Request(server.environment()['LARKSUITE_CLI_AUTH_PROXY'] + uri, method='DELETE'))
                self.fail('unsigned delete accepted')
            except HTTPError as error:
                error.close()
            receipt = server.receipt()
            self.assertEqual(receipt['state'], {'independent': True})
            self.assertEqual(receipt['requests'][0]['proof'], {'bytes': 2})
            rejected = receipt['requests'][1]
            self.assertEqual((rejected['method'], rejected['uri'], rejected['signature_valid']), ('DELETE', '/fixture', False))
            self.assertNotIn('not-retained', json.dumps(receipt))

    def test_trusted_observer_ignores_model_text_and_fails_closed_on_hook_error(self):
        class Fixture(ScenarioBase):
            def __init__(self): self.items = []
            def observe_execution(self, item): self.items.append(item)
            def audit_output(self, raw): return 'forbidden-output' not in raw
        fixture = Fixture()
        raw = '\n'.join(json.dumps(event) for event in [
            {'type': 'item.started', 'item': {'type': 'command_execution'}},
            {'type': 'item.completed', 'item': {'type': 'agent_message', 'text': 'claimed command'}},
            {'type': 'item.completed', 'item': {'type': 'command_execution', 'exit_code': 0}}])
        self.assertTrue(observe(fixture, raw))
        self.assertEqual(len(fixture.items), 1)
        self.assertEqual(fixture._trusted_output_audit, {'checked': True, 'passed': True})
        self.assertFalse(observe(fixture, raw + '\nforbidden-output'))
        fixture.observe_execution = lambda item: 1 / 0
        self.assertFalse(observe(fixture, raw))

    def test_observation_failure_preserves_all_reported_usage(self):
        class Fixture(ScenarioBase):
            def setup(self, workspace):
                assert (workspace / 'guidance/SKILL.md').is_file()
                assert (workspace / 'bin').is_dir()
            def observe_execution(self, item): raise ValueError('unpublishable private content')
        class Child:
            returncode = 0
            def communicate(self, prompt, timeout):
                return '\n'.join(json.dumps(event) for event in [
                    {'type': 'turn.started'},
                    {'type': 'item.completed', 'item': {'type': 'command_execution', 'command': 'echo private', 'exit_code': 0}},
                    {'type': 'turn.completed', 'usage': {'input_tokens': 200, 'cached_input_tokens': 80, 'output_tokens': 30}}]), ''
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); workspace = root / 'workspace'; workspace.mkdir()
            skill = root / 'skill'; skill.mkdir(); (skill / 'SKILL.md').write_text('controlled')
            cli = root / 'cli'; cli.write_text('fixture')
            request = {'workspace': str(workspace), 'cli': str(cli), 'skill': str(skill / 'SKILL.md'),
                       'fixture': {'answer_schema': {'type': 'object'}}, 'prompt': 'controlled',
                       'settings': {'model': 'controlled', 'reasoning': 'high', 'tokenizer': 'provider-undisclosed',
                                    'budget': {'timeout_seconds': 5}}}
            with patch('codex_runner.subprocess.run') as version, patch('codex_runner.subprocess.Popen', return_value=Child()):
                version.return_value.stdout = 'controlled-version'
                response, code = run_agent(request, cli, root / 'auth', 'controlled-version', scenario=Fixture())
            self.assertEqual(code, 1)
            self.assertTrue(response['accounting_complete'])
            self.assertEqual(response['turns'], [{'input_tokens': 200, 'cached_input_tokens': 80, 'output_tokens': 30}])
            self.assertNotIn('private', json.dumps(response))

    def test_receipt_is_captured_after_transport_drains_and_scenario_closes(self):
        import threading
        import time
        from types import SimpleNamespace
        from scenario_fixture import run_scenario
        entered = threading.Event()
        clients = []
        class Fixture(ScenarioBase):
            def __init__(self): self.count, self.closed = 0, False
            def dispatch(self, method, uri, body, headers):
                entered.set(); time.sleep(0.1); self.count += 1
                return Reply('synthetic_change', json={'code': 0})
            def close(self): self.closed = True
            def snapshot(self): return {'count': self.count, 'closed': self.closed}
        def agent(request, codex, auth, version, env, **kwargs):
            def read():
                with urlopen(Request(env['LARKSUITE_CLI_AUTH_PROXY'] + '/fixture', headers=signed_headers('GET', '/fixture'))) as response:
                    response.read()
            thread = threading.Thread(target=read); clients.append(thread); thread.start()
            self.assertTrue(entered.wait(2))
            return {'turns': [{'input_tokens': 5, 'cached_input_tokens': 0, 'output_tokens': 1}]}, 0
        with tempfile.TemporaryDirectory() as directory:
            request = {'task_id': 'controlled', 'workspace': directory,
                       'fixture': {'scenario_section': 'base', 'scenario': 'controlled'}}
            module = SimpleNamespace(Scenario=Fixture, TASK={'id': 'controlled', 'fixture': {'scenario': 'controlled'}})
            with patch('scenario_fixture.load_scenario', return_value=module):
                response, code = run_scenario(request, None, None, None, agent)
            for thread in clients: thread.join(2)
            receipt = json.loads((Path(directory) / 'backend-receipt.json').read_text())
            self.assertEqual(code, 0)
            self.assertEqual(receipt['state'], {'count': 1, 'closed': True})
            self.assertEqual(response['api_calls'], 1)

    def test_partial_http_headers_cannot_block_final_transport_shutdown(self):
        import socket
        import time
        class Fixture(ScenarioBase):
            def dispatch(self, *args): raise AssertionError('partial request dispatched')
        server = ScenarioFixture('controlled', Fixture())
        server.__enter__()
        connection = socket.create_connection(('127.0.0.1', server.server.server_port))
        try:
            connection.sendall(b'GET /')
            time.sleep(0.05)
            start = time.monotonic()
            server.__exit__()
            self.assertLess(time.monotonic() - start, 4)
        finally:
            connection.close()

    def test_loader_and_protected_socket_profile_are_explicit(self):
        with self.assertRaises(ValueError): load_scenario('../arbitrary')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            profile = '\n'.join(configuration(root, root, 'high', unix_sockets=[root / 'bus.sock'], protect_home=True))
            self.assertIn('network={enabled=false,unix_sockets=', profile)
            self.assertIn('filesystem=', profile)
            self.assertNotIn('sandbox_workspace_write', profile)
            offline = '\n'.join(configuration(root, root, 'high', {'FIXTURE_PUBLIC_SETTING': 'value'}, protect_home=True))
            self.assertIn('network={enabled=false}', offline)
            self.assertNotIn('allow_local_binding', offline)
            self.assertNotIn('features.network_proxy=', offline)

    @unittest.skipUnless(os.environ.get('EVAL_NATIVE_CODEX'), 'set EVAL_NATIVE_CODEX for native protected-home enforcement')
    def test_native_packaging_permissions_reject_home_and_network_writes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('home', 'feishu-config', 'tmp', 'bin', 'codex-home'): (root / name).mkdir()
            self.assertEqual(probe_isolation(Path(os.environ['EVAL_NATIVE_CODEX']), root / 'codex-home', root, 'high'),
                             {'protected_home_writes_denied': True, 'network_denied': True})


if __name__ == '__main__':
    unittest.main()
