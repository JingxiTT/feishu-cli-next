"""Signed transport and lifecycle for bounded independently verified scenarios."""
import importlib
import json
import os
from pathlib import Path
import subprocess
import sys
from types import SimpleNamespace

from section_fixture import SectionFixture

SECTIONS = frozenset(('base', 'docs', 'drive', 'events', 'extensions', 'markdown', 'packaging', 'sheets', 'task'))
PAIRED_SECTIONS = frozenset(('discovery', 'contact', 'docs-read', 'docs-edit', 'markdown', 'sheets', 'base', 'drive'))


def load_scenario(section, row_count=None, case_id=None):
    if section.startswith('paired-') and section.removeprefix('paired-') in PAIRED_SECTIONS:
        return importlib.import_module('paired.' + section.removeprefix('paired-').replace('-', '_'))
    if section == "generated-business-v1":
        module = importlib.import_module("scenarios.generated_business")
        task = module.task(case_id)
        return SimpleNamespace(TASK=task, Scenario=lambda: module.make_scenario(task["id"]), verify=module.verify)
    if section in ("scaling-base", "scaling-sheets", "scaling-sheets-v2"):
        if type(row_count) is not int or row_count not in (100, 1000, 10000):
            raise ValueError("unsupported development row count")
        module = importlib.import_module("scaling.base" if section == "scaling-base" else "scaling.sheets")
        result = SimpleNamespace(TASK=module.task(row_count), Scenario=lambda: module.Scenario(row_count), verify=module.verify)
        if hasattr(module, 'verification_diagnostics'):
            result.verification_diagnostics = module.verification_diagnostics
        return result
    if section not in SECTIONS:
        raise ValueError('unknown executable section')
    return importlib.import_module('scenarios.' + section)


class ScenarioFixture(SectionFixture):
    def __init__(self, name, scenario):
        super().__init__(name, scenario.dispatch, scenario.snapshot)
        self.scenario_instance = scenario

    def environment(self):
        return {**super().environment(), **self.scenario_instance.environment()}


def observe(scenario, raw):
    """Consume trusted provider events before discarding content; keep costs on errors."""
    passed = True
    callback = getattr(scenario, 'observe_execution', None)
    if callback:
        for line in raw.splitlines():
            try:
                event = json.loads(line)
            except (ValueError, TypeError):
                continue
            if not isinstance(event, dict):
                continue
            item = event.get('item')
            if event.get('type') == 'item.completed' and isinstance(item, dict) and item.get('type') == 'command_execution':
                try:
                    callback(item)
                except Exception:
                    passed = False
    audit = getattr(scenario, 'audit_output', None)
    if audit:
        try:
            valid = audit(raw) is True
        except Exception:
            valid = False
        scenario._trusted_output_audit = {'checked': True, 'passed': valid}
        passed = passed and valid
    return passed


def probe_isolation(codex, auth_home, workspace, reasoning):
    """Check actual native enforcement before running the protected-home task."""
    from codex_runner import cli_environment, configuration
    command = [str(codex), 'sandbox', '--permission-profile', 'evaluation', '-C', str(workspace)]
    for setting in configuration(auth_home, workspace, reasoning, protect_home=True):
        command += ['-c', setting]
    script = '''import json, pathlib, socket
result = {}
for name in ('home','feishu-config'):
 p = pathlib.Path(name) / '.evaluator-write-probe'
 try:
  p.write_text('probe'); result[name] = False; p.unlink()
 except PermissionError:
  result[name] = True
p = pathlib.Path('workspace-write-probe'); p.write_text('probe'); p.unlink()
s = socket.socket(); s.settimeout(1)
try:
 s.connect(('192.0.2.1', 9)); result['network'] = False
except PermissionError:
 result['network'] = True
except OSError:
 result['network'] = False
finally:
 s.close()
print(json.dumps(result))
'''
    command += [sys.executable, '-c', script]
    result = subprocess.run(command, cwd=workspace, env={**cli_environment(workspace), 'CODEX_HOME': str(auth_home)},
                            capture_output=True, text=True, timeout=20)
    try:
        observed = json.loads(result.stdout)
    except ValueError:
        observed = None
    if result.returncode != 0 or observed != {'home': True, 'feishu-config': True, 'network': True}:
        raise ValueError('native protected-home/network enforcement unavailable')
    return {'protected_home_writes_denied': True, 'network_denied': True}


def run_scenario(request, codex, auth_home, expected_version, run_agent):
    section = request['fixture'].get('scenario_section', request['fixture']['scenario'])
    module = load_scenario(section, request['fixture'].get('row_count'), request['fixture'].get('case_id'))
    if request['task_id'] != module.TASK['id'] or request['fixture']['scenario'] != module.TASK['fixture']['scenario']:
        raise ValueError('scenario task mismatch')
    scenario = module.Scenario()
    fixture = ScenarioFixture(request['fixture']['scenario'], scenario)
    environment = None if section in ('packaging', 'extensions') else SectionFixture.environment(fixture)
    response, code = None, 1
    try:
        with fixture:
            response, code = run_agent(request, codex, auth_home, expected_version, environment,
                                       scenario=scenario, protect_home=section == 'packaging')
    except Exception:
        if response is None:
            raise
        code = 1
    finally:
        try:
            scenario.close()
        except Exception:
            if response is None:
                raise
            code = 1
    extra = {}
    if hasattr(scenario, '_trusted_output_audit'):
        extra['output_audit'] = scenario._trusted_output_audit
    try:
        fixture.write_receipt(request['workspace'], extra)
        response['api_calls'] = len(fixture.records)
    except Exception:
        # Completed provider usage remains available even when evidence capture fails.
        code = 1
    return response, code
