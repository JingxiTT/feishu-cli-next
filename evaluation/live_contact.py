#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Real Feishu current-account fixture driver; read-only, no mock transport."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

from live_fixture import BACKEND


def current_account(cli, config_dir, data_dir, state):
    env = {'PATH': os.defpath, 'HOME': str(state), 'TMPDIR': str(state),
           'LARKSUITE_CLI_CONFIG_DIR': config_dir, 'LARKSUITE_CLI_DATA_DIR': data_dir,
           'LARKSUITE_CLI_REMOTE_META': 'off', 'LARKSUITE_CLI_NO_UPDATE_NOTIFIER': '1',
           'LARKSUITE_CLI_NO_SKILLS_NOTIFIER': '1'}

    def call(arguments):
        result = subprocess.run([str(cli), *arguments], env=env, cwd=state,
                                capture_output=True, text=True, timeout=30, check=False)
        # Do not leave expected identity data in files the model can read.
        with tempfile.NamedTemporaryFile(mode='w', dir=state, prefix='cli-', suffix='.json', delete=False) as stream:
            json.dump({'exit_code': result.returncode, 'command': arguments[:2]}, stream)
        if result.returncode != 0:
            raise ValueError('current-account read failed')
        return json.loads(result.stdout)

    auth = call(['auth', 'status', '--verify', '--json'])
    user = call(['contact', '+get-user', '--as', 'user'])
    identity = user['data']['user']
    account = {'open_id': identity['open_id'], 'tenant_key': identity['tenant_key']}
    if (auth.get('brand') != 'feishu' or auth.get('verified') is not True or auth['identities']['user'].get('verified') is not True
            or auth['identities']['user'].get('openId') != account['open_id']
            or not all(isinstance(v, str) and v for v in account.values())):
        raise ValueError('effective account and tenant could not be verified')
    return account


def run(request, cli, binding):
    if request.get('backend') != BACKEND or request['task']['id'] != 'live-contact-current-account':
        raise ValueError('unsupported live task')
    state = Path(request['state_dir'])
    if request['phase'] == 'cleanup':
        # This fixture only reads the current account and creates no remote objects.
        return {'cleaned': True}
    account = current_account(cli, request['cli_config_dir'], request['cli_data_dir'], state)
    proof = json.loads(binding.read_text())
    fingerprint = hashlib.sha256(('contact-current-account-v1\0' + account['open_id'] + '\0'
                                  + account['tenant_key']).encode()).hexdigest()
    if proof.get('algorithm') != 'sha256' or proof.get('digest') != fingerprint:
        raise ValueError('independent account binding mismatch')
    if request['phase'] == 'setup':
        return {'backend': BACKEND, 'ready': True, 'account_verified': True, 'privacy_verified': True,
                'prompt': request['task']['prompt']}
    if request['phase'] != 'verify':
        raise ValueError('unsupported lifecycle phase')
    # Re-read Feishu independently of the agent and compare the complete answer.
    answer = json.loads((Path(request['workspace']) / 'answer.json').read_text())
    return {'backend': BACKEND, 'success': account == answer}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cli', type=Path, required=True)
    parser.add_argument('--binding', type=Path, required=True,
                        help='Owner-only independent account/tenant SHA-256 binding proof')
    args = parser.parse_args()
    if (not args.cli.is_absolute() or not args.cli.is_file() or not args.binding.is_absolute()
            or args.binding.stat().st_mode & 0o077):
        raise ValueError('absolute CLI and owner-only binding proof required')
    print(json.dumps(run(json.load(sys.stdin), args.cli, args.binding)))
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError):
        print(json.dumps({'ready': False, 'success': False, 'category': 'live_contact_blocked'}))
        sys.exit(2)
