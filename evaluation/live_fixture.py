#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Trusted real-Feishu fixture lifecycle; never a simulated backend."""
from contextlib import contextmanager
import json
import os
from pathlib import Path
import stat
import sys
import tempfile

from harness import atomic_json, process

BACKEND = 'live-feishu-v1'


def load_recipient(config):
    """Load a private test destination; absence never authorizes delivery."""
    if 'recipient_file' not in config:
        return None
    path = Path(config['recipient_file'])
    info = path.stat()
    if not path.is_absolute() or not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077:
        raise ValueError('recipient_file must be an absolute owner-only file')
    recipient = json.loads(path.read_text())
    if (not isinstance(recipient, dict) or set(recipient) != {'open_id', 'tenant_key'}
            or not all(isinstance(value, str) and value.strip() for value in recipient.values())):
        raise ValueError('recipient_file requires nonempty open_id and tenant_key strings')
    return recipient


def validate_live_config(config):
    required = {'driver', 'cli_config_dir', 'cli_data_dir'}
    if not isinstance(config, dict) or not required <= set(config) or set(config) - required - {'recipient_file'}:
        raise ValueError('live requires driver, cli_config_dir and cli_data_dir, with optional recipient_file')
    command = config['driver']
    if (not isinstance(command, list) or not command or not all(isinstance(s, str) for s in command)
            or not Path(command[0]).is_absolute() or not Path(command[0]).is_file()):
        raise ValueError('live driver requires an absolute executable argv array')
    for key in ('cli_config_dir', 'cli_data_dir'):
        directory = Path(config[key])
        if not directory.is_absolute() or not directory.is_dir():
            raise ValueError('live credential directories must be absolute existing protected directories')
    load_recipient(config)


def driver_call(config, phase, state, workspace, task, *, recipient=None, **extra):
    return process(config['live']['driver'], {
        'protocol': 1, 'backend': BACKEND, 'phase': phase,
        'state_dir': str(state), 'workspace': str(workspace), 'task': task,
        'cli_config_dir': config['live']['cli_config_dir'],
        'cli_data_dir': config['live']['cli_data_dir'], **extra,
        'recipient': recipient,
    }, state, config.get('timeout_seconds', 210))


@contextmanager
def attempt(config, task, trial, arm):
    """Keep private recovery state outside the model workspace and public report."""
    if os.environ.get('CI'):
        raise ValueError('real Feishu evaluation cannot run in CI')
    validate_live_config(config['live'])
    recipient = load_recipient(config['live'])
    requires_recipient = task['fixture'].get('requires_recipient', False)
    if type(requires_recipient) is not bool or requires_recipient and recipient is None:
        raise ValueError('delivery tasks require a private recipient_file before setup')
    root = Path(config['live_state_dir'])
    root.mkdir(parents=True, exist_ok=True, mode=0o700)
    root.chmod(0o700)
    blocked = root / 'RECONCILE'
    if blocked.exists():
        raise ValueError('live fixture requires reconciliation before another attempt')
    state = Path(tempfile.mkdtemp(prefix='attempt-', dir=root))
    workspace = state / 'workspace'
    workspace.mkdir(mode=0o700)
    # Persist before the first remote call. A crash must never replay setup.
    atomic_json(blocked, {'state_dir': str(state), 'task_id': task['id'], 'trial': trial, 'arm': arm})
    prepared, code, _ = driver_call(config, 'setup', state, workspace, task,
                                   recipient=recipient, trial=trial, arm=arm)
    verified_recipient = prepared.get('verified_recipient')
    if (code != 0 or prepared.get('backend') != BACKEND or prepared.get('ready') is not True
            or prepared.get('mutation_outcome_unknown') is True
            or prepared.get('account_verified') is not True or prepared.get('privacy_verified') is not True
            or (verified_recipient is not None and (recipient is None or verified_recipient != recipient))
            or (requires_recipient and verified_recipient is None)
            or not isinstance(prepared.get('prompt'), str) or not prepared['prompt'].strip()):
        raise ValueError('live setup blocked; private recovery state retained; no replay')
    bound = {**task, 'prompt': prepared['prompt'], 'fixture': {
        'answer_schema': task['fixture']['answer_schema'],
        'cli_config_dir': config['live']['cli_config_dir'],
        'cli_data_dir': config['live']['cli_data_dir']}}
    if verified_recipient is not None:
        bound['fixture']['authorized_recipient'] = verified_recipient
    # The verifier alone receives the trusted driver command and state path.
    verification = {'live': config['live'], 'live_state_dir': str(state),
                    'recipient': recipient,
                    'live_task': task, 'timeout_seconds': config.get('timeout_seconds', 210)}
    atomic_json(state / 'verifier.json', verification)
    status = {}
    try:
        yield workspace, bound, status
    finally:
        cleaned, clean_code, _ = driver_call(config, 'cleanup', state, workspace, task,
                                            recipient=recipient, trial=trial, arm=arm)
        if clean_code == 0 and cleaned.get('cleaned') is True and cleaned.get('mutation_outcome_unknown') is not True:
            blocked.unlink()
        else:
            status['cleanup_unconfirmed'] = True
        # A failed cleanup leaves RECONCILE and all known accounting intact.


def main():
    request = json.load(sys.stdin)
    workspace = Path(request['workspace'])
    # Read controller data from the private state directory, never model files.
    config = json.loads((workspace.parent / 'verifier.json').read_text())
    payload, code, _ = driver_call(config, 'verify', Path(config['live_state_dir']), workspace,
                                   config['live_task'], recipient=config.get('recipient'))
    print(json.dumps({'success': code == 0 and payload.get('success') is True
                      and payload.get('backend') == BACKEND,
                      'fixture_unsupported': payload.get('fixture_unsupported') is True}))
    return 0 if code == 0 else 1


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError):
        print(json.dumps({'success': False}))
        sys.exit(1)
