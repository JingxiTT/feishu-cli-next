# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Private recipient configuration and lifecycle checks; no tenant calls."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
import live_fixture


class RecipientTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.recipient = {'open_id': 'test-recipient', 'tenant_key': 'test-tenant'}
        self.path = self.root / 'recipient.json'
        self.path.write_text(json.dumps(self.recipient))
        self.path.chmod(0o600)
        self.config = {'live': {'driver': [sys.executable], 'cli_config_dir': str(self.root),
                               'cli_data_dir': str(self.root), 'recipient_file': str(self.path)},
                       'live_state_dir': str(self.root / 'private')}
        self.task = {'id': 'test-delivery', 'prompt': 'Synthetic delivery goal',
                     'fixture': {'answer_schema': {}, 'requires_recipient': True}, 'expected': {}}
        self.setup = {'backend': 'live-feishu-v1', 'ready': True, 'account_verified': True,
                      'privacy_verified': True, 'prompt': self.task['prompt']}
        environment = patch.dict(os.environ, {}, clear=True)
        environment.start()
        self.addCleanup(environment.stop)

    def test_private_file_is_optional_and_rejects_unsafe_or_invalid_input(self):
        self.assertIsNone(live_fixture.load_recipient({}))
        self.assertEqual(live_fixture.load_recipient(self.config['live']), self.recipient)
        self.path.chmod(0o644)
        with self.assertRaises(ValueError):
            live_fixture.validate_live_config(self.config['live'])
        self.path.chmod(0o600)
        for value in ([], {}, {'open_id': 'test'}, {**self.recipient, 'name': 'unnecessary'},
                      {**self.recipient, 'tenant_key': ' '}, {**self.recipient, 'open_id': False}):
            self.path.write_text(json.dumps(value))
            with self.subTest(value=value), self.assertRaises(ValueError):
                live_fixture.validate_live_config(self.config['live'])
        with self.assertRaises(ValueError):
            live_fixture.load_recipient({'recipient_file': 'evaluation/tasks.live-contact.json'})

    def test_reusable_recipient_config_is_ignored_and_template_has_only_placeholders(self):
        repo = Path(__file__).resolve().parents[2]
        for name in ['recipient.json', 'live.local.json', 'verification.private.json']:
            path = '.tmp/evaluation-private/' + name
            ignored = subprocess.run(['git', 'check-ignore', '--no-index', '-q', path],
                                     cwd=repo, capture_output=True, check=False)
            self.assertEqual(ignored.returncode, 0)
            tracked = subprocess.run(['git', 'ls-files', '--error-unmatch', '--', path],
                                     cwd=repo, capture_output=True, check=False)
            self.assertNotEqual(tracked.returncode, 0)
        template = json.loads((repo / 'evaluation/recipient.example.json').read_text())
        self.assertEqual(set(template), {'open_id', 'tenant_key'})
        self.assertTrue(all(value.startswith('<') and value.endswith('>') for value in template.values()))

    def test_delivery_without_config_is_blocked_before_setup(self):
        del self.config['live']['recipient_file']
        with patch.object(live_fixture, 'driver_call') as call:
            with self.assertRaises(ValueError):
                with live_fixture.attempt(self.config, self.task, 0, 'official'):
                    self.fail('unconfigured delivery reached model')
            call.assert_not_called()

    def test_unverified_or_mismatched_destination_blocks_before_model_and_retains_marker(self):
        for recipient in (None, {**self.recipient, 'open_id': 'wrong'},
                          {**self.recipient, 'tenant_key': 'wrong'}):
            with self.subTest(recipient=recipient):
                config = copy.deepcopy(self.config)
                config['live_state_dir'] = str(self.root / ('private-' + str(recipient)))
                prepared = {**self.setup, 'verified_recipient': recipient}
                with patch.object(live_fixture, 'driver_call', return_value=(prepared, 0, 0)) as call:
                    with self.assertRaises(ValueError):
                        with live_fixture.attempt(config, self.task, 0, 'official'):
                            self.fail('unverified delivery reached model')
                    self.assertEqual(call.call_count, 1)
                self.assertTrue((Path(config['live_state_dir']) / 'RECONCILE').exists())

    def test_verified_recipient_reaches_runner_and_is_pinned_for_cleanup(self):
        prepared = {**self.setup, 'verified_recipient': self.recipient}
        with patch.object(live_fixture, 'driver_call', side_effect=[
                (prepared, 0, 0), ({'cleaned': True}, 0, 0)]) as call:
            with live_fixture.attempt(self.config, self.task, 0, 'next') as (workspace, bound, _):
                self.assertEqual(bound['fixture']['authorized_recipient'], self.recipient)
                self.assertNotIn('expected', bound['fixture'])
                verifier = json.loads((workspace.parent / 'verifier.json').read_text())
                self.assertEqual(verifier['recipient'], self.recipient)
                self.assertEqual(call.call_args.kwargs['recipient'], self.recipient)
                self.path.write_text(json.dumps({**self.recipient, 'open_id': 'changed'}))
            self.assertEqual(call.call_args.kwargs['recipient'], self.recipient)
        self.assertFalse((Path(self.config['live_state_dir']) / 'RECONCILE').exists())

    def test_read_only_task_never_inherits_unverified_or_manifest_recipient(self):
        self.task['fixture'].update(requires_recipient=False, authorized_recipient=self.recipient)
        with patch.object(live_fixture, 'driver_call', side_effect=[
                (self.setup, 0, 0), ({'cleaned': True}, 0, 0)]):
            with live_fixture.attempt(self.config, self.task, 0, 'official') as (_, bound, _):
                self.assertNotIn('authorized_recipient', bound['fixture'])


if __name__ == '__main__':
    unittest.main()
