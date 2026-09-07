#!/usr/bin/env python3
"""Independent local state/artifact verification; never invokes the model."""
import json
from pathlib import Path
import sys
from scenario_fixture import load_scenario


def verify(request):
    workspace = Path(request['workspace'])
    module = load_scenario(request['fixture'].get('scenario_section', request['fixture']['scenario']), request['fixture'].get('row_count'), request['fixture'].get('case_id'))
    if request['task_id'] != module.TASK['id']:
        return False
    receipt = json.loads((workspace / 'backend-receipt.json').read_text())
    if any(item.get('route') in ('rejected', 'fixture_error') or item.get('signature_valid') is False
           for item in receipt.get('requests', [])):
        return False
    answer = json.loads((workspace / 'answer.json').read_text())
    return module.verify(workspace, answer, receipt, request['expected']) is True


if __name__ == '__main__':
    request = json.load(sys.stdin)
    limitation = False
    try:
        receipt = json.loads((Path(request['workspace']) / 'backend-receipt.json').read_text())
        limitation = any(item.get('rejection') == 'unsupported_strategy' for item in receipt.get('requests', []))
        success = verify(request)
    except (ValueError, KeyError, TypeError, AttributeError, OSError):
        success = False
    print(json.dumps({'success': success, 'fixture_unsupported': limitation}))
