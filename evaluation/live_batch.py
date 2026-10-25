#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Internal real-Feishu batch adapter for pipeline.py."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys

from harness import atomic_json, validate_config
from live_fixture import BACKEND, validate_live_config

ROOT = Path(__file__).resolve().parents[1]


def load_runner_settings(path):
    settings = json.loads(path.read_text())
    required = {'runner', 'accounting_method', 'tokenizer', 'tools', 'permissions', 'supported_backends', 'mode'}
    if not isinstance(settings, dict) or set(settings) != required or settings['mode'] not in ('real', 'controlled'):
        raise ValueError('runner config requires exactly the documented protocol settings')
    command = settings['runner']
    if (not isinstance(command, list) or not command or not all(isinstance(arg, str) for arg in command)
            or not Path(command[0]).is_absolute() or not Path(command[0]).is_file()):
        raise ValueError('runner must be an argv array with an absolute executable')
    if (not isinstance(settings['supported_backends'], list) or not settings['supported_backends']
            or not all(isinstance(backend, str) for backend in settings['supported_backends'])):
        raise ValueError('declare supported fixture backends')
    for key in ('accounting_method', 'tokenizer', 'tools', 'permissions'):
        if not settings[key]:
            raise ValueError('pin runner measurement settings')
    return settings


def batch_config(config, experiment, output):
    validate_live_config(config['live'])
    cell = next(c for c in config['experiments'] if c['id'] == experiment)
    budget = config.get('budget_seconds', 180)
    settings = {'mode': 'real', 'backend': BACKEND, 'model': cell['model'],
                'reasoning': cell['reasoning'], 'tokenizer': 'provider-undisclosed',
                'tools': ['shell', 'files'],
                'permissions': 'workspace-write; real Feishu HTTPS; private run-owned resources only',
                'budget': {'timeout_seconds': budget}, 'timeout_seconds': budget + 30,
                'guidance_delivery': config.get('guidance_delivery', 'file'),
                'accounting_method': 'codex-cli 0.153.4 provider turn.completed; cached input included',
                'runner': [sys.executable, str(ROOT / 'evaluation/codex_runner.py'),
                           '--codex', cell.get('codex', ''), '--auth-home', cell.get('auth_home', ''),
                           '--codex-version', 'codex-cli 0.153.4'],
                'verifier': [sys.executable, str(ROOT / 'evaluation/live_fixture.py')],
                'live': config['live'], 'live_state_dir': str(output / '.private'),
                'arms': {'official': {'cli': config['official_cli'], 'skill': str(
                    Path(config['official_source']) / 'skills/lark-shared/SKILL.md')},
                         'next': {'cli': config['next_cli'], 'skill': str(
                    Path(config['next_source']) / 'agent-skills/feishu-cli-next/SKILL.md')}}}
    if 'runner_config' in cell:
        custom = load_runner_settings(Path(cell['runner_config']))
        if custom['mode'] != 'real' or BACKEND not in custom['supported_backends']:
            raise ValueError('real evaluation requires a live-feishu-v1 model adapter')
        settings.update({k: v for k, v in custom.items() if k != 'supported_backends'})
    validate_config(settings)
    return settings


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--config', type=Path, required=True)
    parser.add_argument('--experiment', required=True)
    parser.add_argument('--suite', required=True)
    parser.add_argument('--section', action='append', required=True)
    parser.add_argument('--output-root', type=Path, required=True)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument('--prepare-only', action='store_true')
    mode.add_argument('--execute', action='store_true')
    parser.add_argument('--progress', action='store_true')
    args = parser.parse_args()
    config = json.loads(args.config.read_text())
    settings = batch_config(config, args.experiment, args.output_root)
    if args.prepare_only:
        return 0
    if os.environ.get('CI'):
        raise ValueError('real Feishu evaluation cannot run in CI')
    if (Path(settings['live_state_dir']) / 'RECONCILE').exists():
        raise ValueError('live fixture requires reconciliation before execution can resume')
    failed = False
    for section in args.section:
        if (args.output_root / 'STOP').exists():
            break
        directory = args.output_root / section
        directory.mkdir(parents=True, exist_ok=True, mode=0o700)
        path = directory / 'config.local.json'
        atomic_json(path, settings)
        code = subprocess.run([
            sys.executable, str(ROOT / 'evaluation/harness.py'),
            '--config', str(path), '--tasks', config['live_manifests'][args.suite],
            '--section', section, '--output', str(directory / 'batch-01.json'),
            '--original-comparison', '--resume', '--repeats', str(config.get('repeats', 3)),
            '--seed', str(config.get('seed', 20261012)),
            '--stop-file', str(args.output_root / 'STOP'),
            *(['--progress'] if args.progress else []),
        ], check=False).returncode
        if code not in (0, 1):
            return 2
        failed |= code != 0
    return int(failed)


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, ValueError, KeyError, TypeError):
        print(json.dumps({'status': 'blocked', 'category': 'invalid_live_evaluation_setup'}), file=sys.stderr)
        sys.exit(2)
