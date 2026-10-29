# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Business reconciliation goals; native domain readers remain the verifier."""
import copy
from decimal import Decimal, ROUND_HALF_EVEN
import json
import random

from live_domains import base, docs, sheets
from live_domains.session import exact, require


def journal():
    events = []
    for i in range(24):
        events.append({'event_id': f'E{i:03d}', 'revision': 1, 'Code': f'{i % 6 + 1:03d}',
                       'currency': 'USD' if i % 3 == 0 else 'CNY',
                       'amount': str(i % 9 - 4) + '.25', 'void': i % 7 == 0})
    events += [{**events[i], 'revision': 2, 'amount': '0' if i == 3 else '-2.50',
                'void': i == 9} for i in [0, 3, 6, 9, 12, 15]]
    events += [copy.deepcopy(events[i]) for i in [1, 4, 8, 26]]
    random.Random(43).shuffle(events)
    return events


def journal_result(before):
    latest, copies = {}, {}
    for event in journal():
        key = (event['event_id'], event['revision'])
        copies[key] = copies.get(key, 0) + 1
        if event['revision'] > latest.get(event['event_id'], {}).get('revision', 0):
            latest[event['event_id']] = event
    amounts = {row[0]['value']: Decimal(str(row[1]['value'])) for row in before['raw'][1:5]}
    counts = {key: 0 for key in amounts}
    applied, skipped, voided = [], [], []
    for key, event in sorted(latest.items()):
        code = event['Code']
        if event['void']:
            voided.append(key)
        elif code not in amounts or before['raw'][int(code)][2]['value'] is not True:
            skipped.append(key)
        else:
            rate = Decimal('7.25') if event['currency'] == 'USD' else Decimal(1)
            amounts[code] += (Decimal(event['amount']) * rate).quantize(Decimal('.01'), rounding=ROUND_HALF_EVEN)
            counts[code] += 1
            applied.append(key)
    changes = {}
    for code, count in counts.items():
        if count:
            changes[int(code), 1] = float(amounts[code])
            changes[int(code), 3] = f'R43 journal: {count}'
    report = {'applied_event_ids': applied, 'skipped_event_ids': skipped, 'voided_event_ids': voided,
              'duplicate_revisions': sorted(f'{key[0]}:{key[1]}' for key, count in copies.items() if count > 1),
              'closing_amounts': {key: float(value) for key, value in amounts.items()},
              'approved_total': float(sum(value for key, value in amounts.items()
                                         if before['raw'][int(key)][2]['value'] is True))}
    return changes, report


def adjustments():
    rows = [{'request_id': f'R{i:03d}', 'revision': 1, 'Title': f'Item-{i:03d}',
             'expected_amount': i % 7 + (1 if i % 11 == 0 else 0),
             'new_amount': 0 if i % 4 == 0 else -(i % 9),
             'note': None if i % 8 == 0 else 'R43 crédit 你好 😀'} for i in range(0, 205, 3)]
    rows += [{**rows[i], 'revision': 2, 'new_amount': 1.25} for i in [2, 12, 22, 32, 42, 52, 62]]
    rows.append({'request_id': 'R999', 'revision': 1, 'Title': 'Item-999', 'expected_amount': 0,
                 'new_amount': 0, 'note': None})
    rows += [copy.deepcopy(rows[0]), copy.deepcopy(rows[-2])]
    random.Random(44).shuffle(rows)
    return rows


def adjustment_result(before):
    latest = {}
    for row in adjustments():
        if row['revision'] > latest.get(row['request_id'], {}).get('revision', 0):
            latest[row['request_id']] = row
    by_title = {row['Title']: key for key, row in before.items()}
    after, report = copy.deepcopy(before), {k: [] for k in ['applied', 'conflict', 'unapproved', 'missing']}
    for request_id, request in sorted(latest.items()):
        key = by_title.get(request['Title'])
        if key is None:
            category = 'missing'
        elif before[key]['Amount'] != request['expected_amount']:
            category = 'conflict'
        elif before[key]['Checked'] is not True:
            category = 'unapproved'
        else:
            category = 'applied'
            after[key].update(Amount=request['new_amount'], Note=request['note'], Checked=False)
        report[category].append(request_id)
    report['changed_record_ids'] = sorted(key for key in before if not exact(before[key], after[key]))
    report['closing_total'] = sum(row['Amount'] for row in after.values())
    return after, report


def closeout_result(before):
    selected = docs.audit_report(before)
    chosen = set(selected['block_ids'])
    expected = copy.deepcopy(before)
    for block in expected:
        if block['block_id'] in chosen:
            for element in block['text']['elements']:
                if 'text_run' in element:
                    element['text_run']['content'] = element['text_run']['content'].replace('APPROVED', 'SETTLED')
    line = f"R43 closeout: {selected['record_count']} invoices; total {selected['total_amount']}."
    return expected, selected, line


def coalesced_text(block):
    """Run boundaries are not formatting; merge only identical adjacent metadata."""
    result = copy.deepcopy(block)
    elements = result.get('text', {}).get('elements')
    if not isinstance(elements, list):
        return result
    merged = []
    for element in elements:
        current = element.get('text_run')
        previous = merged[-1].get('text_run') if merged else None
        if (element.keys() == {'text_run'} and merged and merged[-1].keys() == {'text_run'}
                and isinstance(current, dict) and isinstance(previous, dict)
                and isinstance(current.get('content'), str) and isinstance(previous.get('content'), str)
                and exact({k: v for k, v in current.items() if k != 'content'},
                          {k: v for k, v in previous.items() if k != 'content'})):
            previous['content'] += current['content']
        else:
            merged.append(element)
    result['text']['elements'] = merged
    return result


def closeout_preserved(before, after):
    from live_workspace import unchanged_blocks
    expected, audit, line = closeout_result(before)
    chosen = set(audit['block_ids'])
    normalize = lambda blocks: [coalesced_text(b) if b['block_id'] in chosen else b for b in blocks]
    return unchanged_blocks(normalize(expected), normalize(after), line)


def setup(s, domain):
    module = {'sheets': sheets, 'base': base, 'docs': docs}[domain]
    original = s.request['task']['id']
    # Reuse calibrated domain seeding and native readers rather than another backend.
    seed = {'sheets': 'common-sheets-intake', 'base': 'common-base-batch', 'docs': 'common-docs-audit-large'}[domain]
    s.request = {**s.request, 'task': {**s.request['task'], 'id': seed}}
    try:
        module.setup(s)
    finally:
        s.request = {**s.request, 'task': {**s.request['task'], 'id': original}}
    if domain == 'sheets':
        (s.workspace / 'journal.json').write_text(json.dumps(journal()))
        return ('Reconcile journal.json into workbook ' + s.data['spreadsheet_token'] + ', sheet '
                + s.data['sheet_id'] + ', A1:F6. For each event_id keep only its highest revision; '
                'identical event_id/revision copies count once. Then exclude void=true; skip missing '
                'Codes and rows whose existing Approved is not boolean true. Keep Codes as strings. '
                'Convert each retained amount to CNY at USD=7.25/CNY=1, rounding EACH converted event '
                'to two decimals using decimal ROUND_HALF_EVEN. Add converted signed amounts to original '
                'Amount. Update only B and D of rows with applied events; D must be "R43 journal: N" '
                '(N is their unique applied event count). Preserve every other cell, style, formula and '
                'GUARD; computed formulas may recalculate. Write report.json with sorted '
                'applied_event_ids, skipped_event_ids, voided_event_ids; sorted duplicate_revisions '
                '("event_id:revision" for input pairs occurring more than once), closing_amounts '
                '(all four existing data Codes mapped to numeric closing amounts), and numeric '
                'approved_total. Skip/void classification uses latest revisions only, void first. '
                'Return exactly {"path":"report.json"}. Do not create resources or change permissions.')
    if domain == 'base':
        (s.workspace / 'adjustments.json').write_text(json.dumps(adjustments(), ensure_ascii=False))
        return ('Apply adjustments.json to Base ' + s.data['base_token'] + ', table ' + s.data['table_id']
                + '. Read the complete table first. Deduplicate each request_id to its highest revision '
                '(identical copies count once). Match exact string Title. Classify in priority order: '
                'missing if no Title; conflict if ORIGINAL Amount differs from expected_amount; '
                'unapproved if ORIGINAL Checked is not boolean true; otherwise applied. Apply only '
                'Amount=new_amount, Note=note (JSON null clears it), Checked=false on applied records. '
                'Preserve all other fields, all skipped records and schema; never create/delete rows. '
                'Write report.json with sorted request-ID arrays applied, conflict, unapproved, missing, '
                'sorted changed_record_ids and numeric closing_total over ALL rows after reconciliation. '
                'Return exactly {"path":"report.json"}. Do not change permissions or send messages.')
    return ('Close out invoices in native document ' + s.data['document_id']
            + ', including all continuation pages beyond 500 blocks. Select paragraphs with release '
            'R42, status APPROVED, and strictly positive Amount. Change only APPROVED to SETTLED in '
            'each selected paragraph. Preserve their native IDs, all other characters and formatting; '
            'preserve every unrelated native block, style, link and order. Append exactly one plain '
            'paragraph "R43 closeout: X invoices; total Y." with original selected count/sum, whole '
            'numbers rendered as integers. Write report.json with sorted original native block_ids, '
            'sorted invoice_codes (leading zeros intact), record_count, total_amount. Return exactly '
            '{"path":"report.json"}. Do not recreate the document, create resources, change '
            'permissions or send messages. Do not replay a mutation with unknown outcome.')


def verify(s, answer, domain):
    require(answer.keys() == {'path'}, 'invalid_answer')
    report = json.loads(s.artifact(answer['path']).read_text())
    if domain == 'sheets':
        s.private_drive(s.data['spreadsheet_token'], 'sheet')
        changes, expected = journal_result(s.data['before'])
        return exact(report, expected) and exact(sheets.snapshot(s), sheets.expected_snapshot(s.data['before'], changes))
    if domain == 'base':
        s.private_drive(s.data['base_token'], 'bitable')
        expected, audit = adjustment_result(s.data['before'])
        return exact(report, audit) and exact(base.records(s), expected)
    s.private_drive(s.data['document_id'], 'docx')
    _, audit, _ = closeout_result(s.data['before'])
    return exact(report, audit) and closeout_preserved(s.data['before'], docs.blocks(s, s.data['document_id']))
