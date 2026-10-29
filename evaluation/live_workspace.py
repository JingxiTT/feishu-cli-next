#!/usr/bin/env python3
# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Live document, spreadsheet and Base business-workflow fixtures."""
import argparse
import copy
import json
from pathlib import Path
import subprocess
import sys

from live_domains import base, complex, docs, sheets
from live_domains.session import Blocked, Session, exact, require
from live_fixture import BACKEND

MODULES = {'docs': docs, 'sheets': sheets, 'base': base}
IDS = {'docs': 'common-docs-edit', 'sheets': 'common-sheets-analyze',
       'base': 'common-base-filter'}
SINGLE = {'business-document-edit': ('docs', 'common-docs-edit'),
          'business-sheet-reconcile': ('sheets', 'common-sheets-update'),
          'business-base-reconcile': ('base', 'common-base-batch'),
          'business-base-audit-large': ('base', 'common-base-audit-large'),
          'business-base-conditional-close': ('base', 'common-base-conditional-close'),
          'business-sheet-intake': ('sheets', 'common-sheets-intake'),
          'business-sheet-export': ('sheets', 'common-sheets-export'),
          'business-document-audit-large': ('docs', 'common-docs-audit-large'),
          'business-sheet-journal': ('sheets', 'common-complex-sheets'),
          'business-base-conflict-reconcile': ('base', 'common-complex-base'),
          'business-document-closeout': ('docs', 'common-complex-docs')}
CROSS = {'business-cross-source-report', 'business-budget-close'}


def session(request, name, cli, binding, resources):
    root = Path(request['state_dir']) / name
    root.mkdir(mode=0o700, exist_ok=True)
    workspace = root / 'workspace'
    workspace.mkdir(mode=0o700, exist_ok=True)
    task = {**request['task'], 'id': SINGLE.get(request['task']['id'], (name, IDS[name]))[1]}
    return Session({**request, 'task': task, 'state_dir': str(root), 'workspace': str(workspace)},
                   cli, binding, resources)


def totals(sessions):
    base_rows = base.records(sessions['base'])
    sheet = sheets.snapshot(sessions['sheets'])
    require(exact(base_rows, sessions['base'].data['before'])
            and exact(sheet, sessions['sheets'].data['before']), 'source_changed')
    selected = {key: row for key, row in base_rows.items() if row['Checked'] is True and row['Amount'] >= 4}
    raw = sheet['raw']
    approved = sum(raw[i][1]['value'] for i in range(1, 5) if raw[i][2]['value'] is True)
    return {'approved_sheet_total': approved, 'selected_base_total': sum(row['Amount'] for row in selected.values()),
            'selected_record_count': len(selected), 'record_ids': sorted(selected)}


def summary_line(values):
    return (f"R42 approved sheet amount: {values['approved_sheet_total']}; "
            f"checked Base amount: {values['selected_base_total']}; "
            f"qualifying records: {values['selected_record_count']}.")


def settlement_report(before):
    selected = base.selected_records(before)
    return {'record_ids': sorted(selected), 'record_count': len(selected),
            'total_amount': sum(row['Amount'] for row in selected.values())}


def settlement_line(values):
    return f"R42 settlement: {values['total_amount']} across {values['record_count']} records."


def verify_settlement(sessions, report):
    d, sh, b = (sessions[name] for name in ['docs', 'sheets', 'base'])
    expected = settlement_report(b.data['before'])
    grid = sheets.expected_snapshot(sh.data['before'], {
        (1, 1): expected['total_amount'], (1, 3): 'Closed R42'})
    return (exact(report, expected) and exact(base.records(b), base.closed_records(b.data['before']))
            and exact(sheets.snapshot(sh), grid) and unchanged_blocks(d.data['before'],
                docs.blocks(d, d.data['document_id']), settlement_line(expected)))


def unchanged_blocks(before, after, line):
    old = {b['block_id']: b for b in before}
    now = {b['block_id']: b for b in after}
    added = [b for b in after if b['block_id'] not in old]
    if (len(old) != len(before) or len(now) != len(after) or not old.keys() <= now.keys()
            or len(added) != 1 or added[0].get('block_type') != 2 or docs.text(added[0]) != line):
        return False
    roots = [key for key, block in old.items() if block.get('block_type') == 1]
    if len(roots) != 1:
        return False
    expected = copy.deepcopy(old)
    root = roots[0]
    expected[root]['children'] = old[root].get('children', []) + [added[0]['block_id']]
    return all(exact(value, now[key]) for key, value in expected.items())


def run(request, cli, binding, resources):
    identifier = request['task']['id']
    require(identifier in CROSS | SINGLE.keys()
            and request.get('backend') == BACKEND, 'unsupported_live_task')
    names = (['docs', 'sheets', 'base'] if identifier in CROSS
             else [SINGLE[identifier][0]])
    phase = request['phase']
    if phase == 'cleanup':
        for name in reversed(names):
            root = Path(request['state_dir']) / name
            if not (root / 'common-state.json').exists():
                continue
            s = session(request, name, cli, binding, resources)
            require(s.data['pending'] is None, 'mutation_requires_reconciliation')
            if not s.data.get('cleanup_confirmed'):
                MODULES[name].cleanup(s)
                s.data['cleanup_confirmed'] = True
                s.save()
        return {'cleaned': True}
    sessions = {}
    prompts = {}
    for name in names:
        s = session(request, name, cli, binding, resources)
        sessions[name] = s
        if phase == 'preflight':
            s.parent()
        elif phase == 'setup':
            require(not s.data.get('setup_started'), 'fixture_setup_already_started')
            s.data['setup_started'] = True
            s.save()
            prompts[name] = (complex.setup(s, name) if s.request['task']['id'].startswith('common-complex-')
                             else MODULES[name].setup(s))
            s.data['setup_complete'] = True
            s.save()
    if phase == 'preflight':
        return {'backend': BACKEND, 'prerequisites_verified': True, 'model_execution': 'not_run'}
    if phase == 'setup':
        if identifier in SINGLE:
            s = sessions[names[0]]
            for source in s.workspace.iterdir():
                require(source.is_file() and not source.is_symlink(), 'unexpected_fixture_input')
                (Path(request['workspace']) / source.name).write_bytes(source.read_bytes())
            prompt = prompts[names[0]]
        else:
            d, sh, b = (sessions[name] for name in names)
            sources = ('native document ' + d.data['document_id'] + '; workbook '
                + sh.data['spreadsheet_token'] + ', sheet ' + sh.data['sheet_id']
                + ', range A1:F6; and Base ' + b.data['base_token'] + ', table ' + b.data['table_id'])
            if identifier == 'business-budget-close':
                prompt = ('Close the R42 budget across these private live sources: ' + sources
                    + '. First read the complete Base and select Checked=true and Amount>=4. Save their '
                    'sorted record_ids, record_count and total_amount in report.json before changing '
                    'selection fields. Set only Checked=false and Note="Closed R42" on those Base '
                    'records, preserving Title, Amount and all unselected records. Set only Sheet B2 '
                    'to the numeric total_amount and D2 to "Closed R42", preserving every other cell, '
                    'style and formula; formula results may recalculate. Append exactly one plain '
                    'paragraph to the document: "R42 settlement: X across Y records." Replace X/Y with '
                    'the original selected total/count, rendering whole numbers as integers. Preserve '
                    'all existing native blocks, IDs, formatting, links and Unicode. Do not alter '
                    'permissions/memberships, send messages, create other resources, or replay an '
                    'uncertain mutation. Return exactly {"report_path":"report.json"}.')
            else:
                prompt = ('Prepare an R42 financial readiness note from three live sources: native document '
                + d.data['document_id'] + '; workbook ' + sh.data['spreadsheet_token'] + ', sheet '
                + sh.data['sheet_id'] + ', range A1:F6; and Base ' + b.data['base_token']
                + ', table ' + b.data['table_id'] + '. In the sheet, sum Amount only for Approved=true '
                'data rows; exclude the header and GUARD row. In Base, select every record with '
                'Checked=true and Amount>=4 across the entire table, including all pages; sum Amount '
                'and count matching records. Write report.json with approved_sheet_total, '
                'selected_base_total, selected_record_count and sorted record_ids. Append exactly one '
                'plain paragraph to the native document: "R42 approved sheet amount: X; checked Base '
                'amount: Y; qualifying records: Z." Replace X/Y/Z with the computed numbers, '
                'rendering whole-number totals as integers. '
                'Preserve every existing native block, ID, style, link, Unicode character and unrelated '
                'text. Do not change the workbook, Base, permissions or memberships, and do not send '
                'messages or share anything. Return exactly {"report_path":"report.json"}.')
        return {'backend': BACKEND, 'ready': True, 'account_verified': True,
                'privacy_verified': True, 'prompt': prompt}
    require(phase == 'verify', 'unsupported_fixture_phase')
    try:
        workspace = Path(request['workspace'])
        require(not (workspace / 'answer.json').is_symlink(), 'invalid_answer')
        answer = json.loads((workspace / 'answer.json').read_text())
        if identifier in SINGLE:
            s = sessions[names[0]]
            s.workspace = workspace
            success = (complex.verify(s, answer, names[0])
                       if s.request['task']['id'].startswith('common-complex-')
                       else MODULES[names[0]].verify(s, answer))
        else:
            require(answer.keys() == {'report_path'}, 'invalid_answer')
            d = sessions['docs']
            d.workspace = workspace
            report = json.loads(d.artifact(answer['report_path']).read_text())
            for name, kind, key in [('docs', 'docx', 'document_id'), ('sheets', 'sheet', 'spreadsheet_token'),
                                    ('base', 'bitable', 'base_token')]:
                s = sessions[name]
                s.private_drive(s.data[key], kind)
            if identifier == 'business-budget-close':
                success = verify_settlement(sessions, report)
            else:
                expected = totals(sessions)
                success = (exact(report, expected) and unchanged_blocks(d.data['before'],
                    docs.blocks(d, d.data['document_id']), summary_line(expected)))
    except (OSError, ValueError, KeyError, TypeError):
        success = False
    return {'backend': BACKEND, 'success': success}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--cli', type=Path, required=True)
    parser.add_argument('--binding', type=Path, required=True)
    parser.add_argument('--resources', type=Path, required=True)
    args = parser.parse_args()
    try:
        print(json.dumps(run(json.load(sys.stdin), args.cli, args.binding, args.resources)))
        return 0
    except Blocked as error:
        print(json.dumps({'ready': False, 'success': False, 'category': str(error)}))
        return 2
    except (OSError, ValueError, KeyError, TypeError, IndexError, AttributeError, subprocess.SubprocessError):
        print(json.dumps({'ready': False, 'success': False, 'category': 'live_business_blocked'}))
        return 2


if __name__ == '__main__':
    sys.exit(main())
