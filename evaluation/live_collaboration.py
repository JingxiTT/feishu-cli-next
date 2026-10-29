#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Permission-bounded real Feishu collaboration evaluation lifecycle."""
import argparse
import json
from pathlib import Path
import subprocess
import sys

from live_domains import collaboration
from live_domains.session import Blocked, Session, require
from live_fixture import BACKEND

MEDIA = {'collaboration-document-create', 'collaboration-image-upload', 'collaboration-file-upload',
         'collaboration-mermaid-create', 'collaboration-drawing-create'}


def run(request, cli, binding, resources):
    identifier, phase = request['task']['id'], request['phase']
    require(request.get('backend') == BACKEND and identifier in MEDIA, 'unsupported_live_task')
    s = Session(request, cli, binding, resources)
    if phase == 'cleanup':
        require(s.data['pending'] is None, 'mutation_requires_reconciliation')
        if not s.data.get('cleanup_confirmed'):
            collaboration.cleanup(s)
            s.data['cleanup_confirmed'] = True
            s.save()
        return {'cleaned': True}
    if phase == 'preflight':
        s.parent()
        return {'backend': BACKEND, 'prerequisites_verified': True, 'model_execution': 'not_run'}
    if phase == 'setup':
        require(not s.data.get('setup_started'), 'fixture_setup_already_started')
        s.data['setup_started'] = True
        s.save()
        prompt = collaboration.setup(s)
        s.data['setup_complete'] = True
        s.save()
        return {'backend': BACKEND, 'ready': True, 'account_verified': True,
                'privacy_verified': True, 'prompt': prompt}
    require(phase == 'verify', 'unsupported_fixture_phase')
    try:
        answer = json.loads(s.artifact('answer.json').read_text())
        success = collaboration.verify(s, answer)
    except (OSError, ValueError, KeyError, TypeError, IndexError):
        success = False
    return {'backend': BACKEND, 'success': success}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--cli', type=Path, required=True)
    p.add_argument('--binding', type=Path, required=True)
    p.add_argument('--resources', type=Path, required=True)
    a = p.parse_args()
    try:
        print(json.dumps(run(json.load(sys.stdin), a.cli, a.binding, a.resources)))
        return 0
    except Blocked as e:
        print(json.dumps({'ready': False, 'success': False, 'category': str(e)}))
        return 2
    except (OSError, ValueError, KeyError, TypeError, IndexError, subprocess.SubprocessError):
        print(json.dumps({'ready': False, 'success': False, 'category': 'live_collaboration_blocked'}))
        return 2


if __name__ == '__main__':
    sys.exit(main())
