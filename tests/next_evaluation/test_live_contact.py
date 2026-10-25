# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline tests of the live driver; no credentials or network calls."""
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
import live_contact


class LiveContactTests(unittest.TestCase):
    def test_bound_current_account_setup_readback_and_no_remote_cleanup(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            workspace = root / 'workspace'
            workspace.mkdir()
            account = {'open_id': 'test-account', 'tenant_key': 'test-tenant'}
            binding = root / 'binding.json'
            binding.write_text(json.dumps({'algorithm': 'sha256', 'digest': hashlib.sha256(
                b'contact-current-account-v1\0test-account\0test-tenant').hexdigest()}))
            request = {'backend': 'live-feishu-v1', 'state_dir': directory,
                       'workspace': str(workspace), 'cli_config_dir': '/unused-in-test',
                       'cli_data_dir': '/unused-in-test',
                       'task': {'id': 'live-contact-current-account', 'prompt': 'Find the current account'},
                       'phase': 'setup'}
            with patch.object(live_contact, 'current_account', return_value=account) as read:
                setup = live_contact.run(request, '/unused-in-test', binding)
                self.assertTrue(setup['ready'])
                self.assertNotIn('test-account', json.dumps(setup))
                self.assertFalse((root / 'account.json').exists())
                (workspace / 'answer.json').write_text(json.dumps(account))
                request['phase'] = 'verify'
                self.assertTrue(live_contact.run(request, '/unused-in-test', binding)['success'])
                (workspace / 'answer.json').write_text(json.dumps({**account, 'tenant_key': 'wrong'}))
                self.assertFalse(live_contact.run(request, '/unused-in-test', binding)['success'])
                request['phase'] = 'cleanup'
                self.assertEqual(live_contact.run(request, '/unused-in-test', binding), {'cleaned': True})
                self.assertEqual(read.call_count, 3)
                binding.write_text('{"algorithm":"sha256","digest":"wrong"}')
                request['phase'] = 'setup'
                with self.assertRaises(ValueError):
                    live_contact.run(request, '/unused-in-test', binding)

    def test_real_cli_reads_strip_ambient_proxy_and_verify_brand_and_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            state = Path(directory)
            user = {'data': {'user': {'open_id': 'test-account', 'tenant_key': 'test-tenant'}}}
            auth = {'brand': 'feishu', 'verified': True,
                    'identities': {'user': {'verified': True, 'openId': 'test-account'}}}
            def response(value):
                from subprocess import CompletedProcess
                return CompletedProcess([], 0, json.dumps(value), '')
            with patch.object(live_contact.subprocess, 'run', side_effect=[response(auth), response(user)]) as run:
                account = live_contact.current_account('/test-cli', '/test-config', '/test-data', state)
                self.assertEqual(account, user['data']['user'])
                self.assertEqual(run.call_count, 2)
                env = run.call_args.kwargs['env']
                self.assertEqual(env['LARKSUITE_CLI_CONFIG_DIR'], '/test-config')
                self.assertEqual(env['LARKSUITE_CLI_DATA_DIR'], '/test-data')
                self.assertNotIn('LARKSUITE_CLI_AUTH_PROXY', env)
                self.assertNotIn('HTTPS_PROXY', env)
                for diagnostic in state.glob('cli-*.json'):
                    saved = diagnostic.read_text()
                    self.assertNotIn('test-account', saved)
                    self.assertNotIn('test-tenant', saved)
                    self.assertNotIn('stdout', saved)
                    self.assertNotIn('stderr', saved)
            for invalid in ({**auth, 'brand': 'lark'}, {**auth, 'verified': False}):
                with patch.object(live_contact.subprocess, 'run', side_effect=[response(invalid), response(user)]):
                    with self.assertRaises(ValueError):
                        live_contact.current_account('/test-cli', '/test-config', '/test-data', state)


if __name__ == '__main__':
    unittest.main()
