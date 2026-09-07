#!/usr/bin/env python3
"""Independent local state/artifact verification; never invokes the model."""
import json
from pathlib import Path
import sys
from scenario_fixture import load_scenario
from section_fixture import UNSUPPORTED_REASON_CODES

SCENARIO_DIAGNOSTICS = frozenset(('execution_proof_unavailable', 'packaging_receipt_invalid',
    'packaging_isolation_missing', 'packaging_answer_mismatch', 'packaging_answer_path_mismatch',
    'packaging_protected_changed', 'packaging_project_changed', 'packaging_install_invalid',
    'base_scaling_answer_invalid', 'base_scaling_report_path', 'base_scaling_report_format',
    'base_scaling_report_ids', 'base_scaling_missing_rows', 'base_scaling_typed_cells',
    'base_scaling_metadata_changed', 'base_scaling_input_changed', 'base_scaling_api_rejected',
    'base_scaling_unsupported_strategy', 'base_scaling_bad_mutation_proof', 'base_scaling_fixture_state_invalid'))


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


def unsupported_diagnostics(receipt):
    return sorted({item.get('unsupported_reason') if isinstance(item.get('unsupported_reason'), str) and item.get('unsupported_reason') in UNSUPPORTED_REASON_CODES
                   else 'unsupported_strategy_other' for item in receipt.get('requests', [])
                   if item.get('rejection') == 'unsupported_strategy'})


def outcome_diagnostics(request, receipt):
    module = load_scenario(request['fixture'].get('scenario_section', request['fixture']['scenario']),
        request['fixture'].get('row_count'), request['fixture'].get('case_id'))
    # A missing scenario-specific proof must not hide a known shared rejection.
    if request['task_id'] != module.TASK['id']:
        return ['scenario_mismatch']
    if any(item.get('route') in ('rejected', 'fixture_error') or item.get('signature_valid') is False
           for item in receipt.get('requests', [])):
        return ['request_outside_fixture']
    diagnose = getattr(module, 'verification_diagnostics', None)
    if diagnose is None:
        return []
    answer = json.loads((Path(request['workspace']) / 'answer.json').read_text())
    reasons = diagnose(Path(request['workspace']), answer, receipt, request['expected'])
    return sorted({code for code in reasons if isinstance(code, str) and code in SCENARIO_DIAGNOSTICS}) if isinstance(reasons, list) else []


def evaluate(request):
    limitation = False
    reasons = []
    try:
        receipt = json.loads((Path(request['workspace']) / 'backend-receipt.json').read_text())
        reasons = unsupported_diagnostics(receipt)
        # A rejected route can short-circuit verify(); still reject malformed input.
        json.loads((Path(request['workspace']) / 'answer.json').read_text())
        success = verify(request)
        if not success:
            reasons = sorted(set(reasons + outcome_diagnostics(request, receipt)))
            known_failure = 'scenario_mismatch' in reasons or any(
                item.get('signature_valid') is False or item.get('route') == 'fixture_error' or
                (item.get('route') == 'rejected' and item.get('rejection') != 'unsupported_strategy')
                for item in receipt.get('requests', []))
            limitation = not known_failure and (bool(unsupported_diagnostics(receipt)) or
                                                'execution_proof_unavailable' in reasons)
    except (ValueError, KeyError, TypeError, AttributeError, OSError):
        success = False
        limitation = False
        reasons = sorted(set(reasons + ['invalid_verification_input']))
    return {'success': success, 'fixture_unsupported': limitation, 'verification_failures': reasons}


if __name__ == '__main__':
    print(json.dumps(evaluate(json.load(sys.stdin))))
