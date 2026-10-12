# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Composed release jobs using existing native Docs, Sheets and Task fixtures."""
from copy import deepcopy
from pathlib import Path
from urllib.parse import urlsplit

from paired import docs_read, sheets
from scenarios import task as task_fixture
from scenario_protocol import ScenarioBase

CASES = ('release-review', 'release-handoff')
BRIEF = 'release-brief.md'


def brief(case):
    common = f'''# Release handoff

Release approval document: https://example.feishu.cn/docx/{docs_read.native.DOCUMENT}
Use its current launch approval, citing the native block containing it.
Keep the approval document and all unrelated resources unchanged.
'''
    if case == 'release-review':
        return common + f'''
Release checks: https://example.feishu.cn/sheets/{sheets.SOURCE['spreadsheet_token']}
Worksheet: Fixture. The release row is A1:E1:
A = open blockers, B = exception requested, C = exception note,
D = localized release note, E = next batch number (preserve formula and numeric result).

Write a JSON review with approval_code, approval_block_id, open_blockers,
exception_requested, exception_note, release_note, next_batch_number and ready.
Ready requires an approval, zero blockers and no exception request.
Also save a lossless native cell JSON export of the row.
'''
    return common + f'''
Find my task with the exact title "{task_fixture._INITIAL['tasks'][task_fixture.TARGET]['summary']}".
Complete it only after checking that the document has a launch approval with
an APR- prefix. Preserve its assignees and all other content. Do not send messages.
Write a JSON handoff record with approval_code, approval_block_id, task_id,
completed and assignee_ids, using the actual resulting task state.
'''


def task(case):
    if case not in CASES:
        raise ValueError('unknown E2E case')
    review = case == 'release-review'
    paths = ['report_path', 'cells_path'] if review else ['report_path']
    prompt = ('Prepare a release readiness review from release-brief.md, including a decision and a '
              'lossless copy of the supporting spreadsheet row. Save the review and cell export as JSON '
              'files and return their workspace-relative report_path and cells_path. Keep the sources unchanged.'
              if review else
              'Review the release approval and close the matching handoff task described in release-brief.md '
              'if it is approved. Save a JSON handoff record citing the approval and confirming the resulting '
              'task state. Return its workspace-relative report_path. Leave unrelated work unchanged.')
    return {'id': 'e2e-' + case, 'section': 'workflow', 'category': 'e2e',
            'subsystems': ['docs', 'sheets'] if review else ['docs', 'task'],
            'prompt': prompt,
            'fixture': {'scenario': 'e2e-' + case + '-v1', 'scenario_section': 'e2e-release-v1',
                        'case_id': case, 'allow_synthetic_mutations': not review,
                        'answer_schema': {'type': 'object', 'additionalProperties': False,
                                          'required': paths, 'properties': {p: {'type': 'string'} for p in paths}}},
            'expected': {'checks': ['all source reads independently observed', 'correct composed result',
                                    'brief and unrelated resources preserved',
                                    'typed cell export and computed decision' if review else
                                    'approval observed before task completion; resulting task verified']}}


class Scenario(ScenarioBase):
    def __init__(self, case):
        task(case)
        self.case = case
        self.parts = {'docs': docs_read.Scenario(),
                      'sheets' if case == 'release-review' else 'task':
                      sheets.Scenario() if case == 'release-review' else task_fixture.Scenario()}

    def setup(self, workspace):
        (Path(workspace) / BRIEF).write_text(brief(self.case), encoding='utf-8')

    def dispatch(self, method, uri, body_bytes, headers):
        if headers.get('X-Lark-Proxy-Identity') != 'user':
            raise ValueError('release jobs require the bound user')
        path = urlsplit(uri).path
        if path == '/open-apis/authen/v1/user_info':
            component = 'sheets' if self.case == 'release-review' else 'task'
        elif path.startswith(('/open-apis/docs_ai/', '/open-apis/docx/')):
            component = 'docs'
        elif self.case == 'release-review' and path.startswith(('/open-apis/sheets/', '/open-apis/sheet_ai/')):
            component = 'sheets'
        elif self.case == 'release-handoff' and path.startswith('/open-apis/task/'):
            component = 'task'
        else:
            raise ValueError('unlisted release job route')
        reply = self.parts[component].dispatch(method, uri, body_bytes, headers)
        # Keep native proofs intact so their independent verifiers remain reusable.
        reply.proof = {'component': component, 'native': reply.proof}
        return reply

    def snapshot(self):
        return {key: part.snapshot() for key, part in self.parts.items()}

    def close(self):
        for part in self.parts.values():
            part.close()


def subreceipt(receipt, component, module):
    records = []
    for record in receipt['requests']:
        proof = record['proof']
        if proof['component'] == component:
            records.append({**record, 'proof': deepcopy(proof['native'])})
    return {'scenario': module.TASK['fixture']['scenario'],
            'state': receipt['state'][component], 'requests': records}


def artifact(workspace, relative):
    if not isinstance(relative, str) or not relative or Path(relative).is_absolute():
        raise ValueError('workspace-relative artifact required')
    root = Path(workspace).resolve()
    path = (root / relative).resolve(strict=True)
    if not path.is_relative_to(root) or not path.is_file() or path.stat().st_size > 1000000:
        raise ValueError('artifact outside bounded workspace')
    return sheets.original._json(path.read_text(encoding='utf-8'))


def verify(case, workspace, answer, receipt, expected):
    del expected
    try:
        definition = task(case)
        if receipt['scenario'] != definition['fixture']['scenario']:
            return False
        if set(answer) != set(definition['fixture']['answer_schema']['required']):
            return False
        if (Path(workspace) / BRIEF).read_bytes() != brief(case).encode('utf-8'):
            return False
        components = {'docs', 'sheets' if case == 'release-review' else 'task'}
        if set(receipt['state']) != components:
            return False
        for record in receipt['requests']:
            if (record.get('signature_valid') is not True or record.get('status') != 200
                    or record.get('route') in ('rejected', 'fixture_error')
                    or record['proof'].get('component') not in components):
                return False
        review = artifact(workspace, answer['report_path'])
        if not isinstance(review, dict):
            return False
        source = {'approval_code': review.get('approval_code'), 'block_id': review.get('approval_block_id')}
        if not docs_read.verify(workspace, source, subreceipt(receipt, 'docs', docs_read), docs_read.TASK['expected']):
            return False
        if case == 'release-review':
            if not sheets.verify(workspace, {'artifact_path': answer['cells_path']},
                                 subreceipt(receipt, 'sheets', sheets), sheets.TASK['expected']):
                return False
            # Independently declared business facts, not copied from agent output.
            wanted = {'approval_code': docs_read.FACT, 'approval_block_id': docs_read.BLOCK,
                      'open_blockers': 0, 'exception_requested': False, 'exception_note': '',
                      'release_note': '你好', 'next_batch_number': 1, 'ready': True}
            return sheets.original._same(review, wanted)
        result = {key: review.get(key) for key in ('task_id', 'completed', 'assignee_ids')}
        if not task_fixture.verify(workspace, result, subreceipt(receipt, 'task', task_fixture), task_fixture.TASK['expected']):
            return False
        wanted = {**task_fixture.TASK['expected']['answer'],
                  'approval_code': docs_read.FACT, 'approval_block_id': docs_read.BLOCK}
        if not sheets.original._same(review, wanted):
            return False
        approved = False
        for record in receipt['requests']:
            proof = record['proof']
            if proof['component'] == 'docs' and proof['native'].get('fact_and_block_returned') is True:
                approved = True
            if (proof['component'] == 'task' and record['route'] == 'task_patch'
                    and proof['native'].get('changed') is True and not approved):
                return False
        return approved
    except (OSError, UnicodeError, ValueError, KeyError, TypeError, AttributeError):
        return False
