# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Composed job outcomes, data handoffs and independently observed effects."""
from copy import deepcopy
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
from codex_runner import validate_output_schema
from e2e import release
from scenario_fixture import ScenarioFixture, load_scenario
from section_fixture import signed_headers


class ReleaseE2ETests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.workspace = Path(self.temp.name)

    def start(self, case):
        self.case = case
        self.definition = release.task(case)
        self.scenario = release.Scenario(case)
        self.scenario.setup(self.workspace)
        self.records = []
        self.send = self.direct

    def direct(self, method, uri, body=None):
        reply = self.scenario.dispatch(method, uri, b'' if body is None else json.dumps(body).encode(),
                                       {'X-Lark-Proxy-Identity': 'user'})
        self.records.append({'method': method, 'uri': uri.split('?')[0], 'status': reply.status,
                             'route': reply.route, 'proof': deepcopy(reply.proof), 'signature_valid': True})
        return reply.json['data']

    def approval(self):
        data = self.send('GET', release.docs_read.native.DOCX_PATH + '/blocks/' + release.docs_read.BLOCK)
        return {'approval_code': re.search(r'APR-\d+', json.dumps(data)).group(),
                'approval_block_id': data['block']['block_id']}

    def cells(self, mode):
        data = {'excel_id': release.sheets.SOURCE['spreadsheet_token'], 'sheet_id': 'shdev01',
                'ranges': ['A1:E1'], 'value_render_option': mode, 'include_styles': False}
        response = self.send('POST', release.sheets.READ_PATH,
                             {'tool_name': 'get_cell_ranges', 'input': json.dumps(data)})
        return json.loads(response['output'])

    def review(self):
        report = self.approval()
        raw, formula = self.cells('raw_value'), self.cells('formula')
        cells = raw['ranges'][0]['cells'][0]
        formula_cell = formula['ranges'][0]['cells'][0][-1]
        self.assertEqual(cells[-1]['value'], formula_cell['value'])
        cells[-1]['formula'] = formula_cell['formula']
        (self.workspace / 'cells.json').write_text(json.dumps(raw, ensure_ascii=False))
        report.update(open_blockers=cells[0]['value'], exception_requested=cells[1]['value'],
                      exception_note=cells[2]['value'], release_note=cells[3]['value'],
                      next_batch_number=cells[4]['value'])
        report['ready'] = bool(report['approval_code']) and report['open_blockers'] == 0 and report['exception_requested'] is False
        (self.workspace / 'review.json').write_text(json.dumps(report, ensure_ascii=False))
        return {'report_path': 'review.json', 'cells_path': 'cells.json'}

    def complete(self, guid):
        return self.send('PATCH', release.task_fixture._BASE + '/' + guid,
                         {'task': {'completed_at': '1893546000000'}, 'update_fields': ['completed_at']})['task']

    def handoff(self, approved_first=True):
        source = self.approval() if approved_first else None
        tasks = self.send('GET', release.task_fixture._BASE)['items']
        chosen = next(t for t in tasks if t['summary'] == 'Private completion fixture')
        result = self.complete(chosen['guid'])
        if source is None:
            source = self.approval()
        source.update(task_id=result['guid'], completed=result['status'] == 'done',
                      assignee_ids=[m['id'] for m in result['members'] if m['role'] == 'assignee'])
        (self.workspace / 'handoff.json').write_text(json.dumps(source))
        return {'report_path': 'handoff.json'}

    def receipt(self):
        return {'scenario': self.definition['fixture']['scenario'],
                'requests': deepcopy(self.records), 'state': self.scenario.snapshot()}

    def verify(self, answer, receipt=None):
        return release.verify(self.case, self.workspace, answer, self.receipt() if receipt is None else receipt,
                              self.definition['expected'])

    def test_manifest_has_real_compositions_and_matches_executable_contracts(self):
        manifest = json.loads((ROOT / 'evaluation/tasks.e2e.json').read_text())
        self.assertEqual(manifest['category'], 'e2e')
        self.assertEqual(manifest['tasks'], [release.task(case) for case in release.CASES])
        for task in manifest['tasks']:
            self.assertEqual(task['category'], 'e2e')
            self.assertGreaterEqual(len(set(task['subsystems'])), 2)
            self.assertEqual(load_scenario('e2e-release-v1', case_id=task['fixture']['case_id']).TASK, task)
            validate_output_schema(task['fixture']['answer_schema'])
            for hidden in ('APR-0042', release.docs_read.BLOCK, release.task_fixture.TARGET, '你好'):
                self.assertNotIn(hidden, json.dumps(task['fixture'], ensure_ascii=False))
        with self.assertRaises(ValueError):
            load_scenario('e2e-release-v1', case_id='unknown')

    def test_review_requires_both_subsystems_correct_decision_and_typed_export(self):
        self.start('release-review')
        answer = self.review()
        self.assertTrue(self.verify(answer))
        for missing in ('docs', 'sheets'):
            receipt = self.receipt()
            receipt['requests'] = [r for r in receipt['requests'] if r['proof']['component'] != missing]
            self.assertFalse(self.verify(answer, receipt))
        path = self.workspace / 'review.json'
        original = path.read_text()
        for key, value in (('ready', False), ('open_blockers', False), ('exception_requested', 0),
                           ('approval_code', 'wrong'), ('release_note', 'wrong')):
            report = json.loads(original)
            report[key] = value
            path.write_text(json.dumps(report))
            self.assertFalse(self.verify(answer))
        path.write_text(original)
        cells = json.loads((self.workspace / 'cells.json').read_text())
        cells['ranges'][0]['cells'][0][-1].pop('formula')
        (self.workspace / 'cells.json').write_text(json.dumps(cells))
        self.assertFalse(self.verify(answer))

    def test_handoff_must_observe_approval_before_mutation(self):
        self.start('release-handoff')
        answer = self.handoff()
        self.assertTrue(self.verify(answer))
        receipt = self.receipt()
        receipt['requests'] = [r for r in receipt['requests'] if r['proof']['component'] != 'docs']
        self.assertFalse(self.verify(answer, receipt))
        self.start('release-handoff')
        self.assertFalse(self.verify(self.handoff(approved_first=False)))

    def test_unrelated_changes_brief_changes_and_unsigned_evidence_fail(self):
        self.start('release-handoff')
        answer = self.handoff()
        receipt = self.receipt()
        receipt['requests'][0]['signature_valid'] = False
        self.assertFalse(self.verify(answer, receipt))
        (self.workspace / release.BRIEF).write_text('changed brief')
        self.assertFalse(self.verify(answer))
        (self.workspace / release.BRIEF).write_text(release.brief(self.case))
        unrelated = next(guid for guid in self.scenario.parts['task'].tasks if guid != release.task_fixture.TARGET)
        self.complete(unrelated)
        self.assertFalse(self.verify(answer))

    def test_artifacts_cannot_escape_workspace_or_forge_source_proof(self):
        self.start('release-review')
        answer = self.review()
        for invalid in ('../outside.json', '/tmp/outside.json', '', 1):
            self.assertFalse(self.verify({**answer, 'report_path': invalid}))
        receipt = self.receipt()
        receipt['state']['docs']['final']['blocks'][2]['text'] = 'changed source'
        self.assertFalse(self.verify(answer, receipt))
        with self.assertRaises(ValueError):
            self.scenario.dispatch('GET', '/open-apis/authen/v1/user_info', b'', {'X-Lark-Proxy-Identity': 'bot'})
        with self.assertRaises(ValueError):
            self.scenario.dispatch('POST', '/open-apis/im/v1/messages', b'{}', {'X-Lark-Proxy-Identity': 'user'})

    def test_signed_transport_and_separate_verifier_process(self):
        for case in release.CASES:
            with self.subTest(case=case):
                self.start(case)
                with ScenarioFixture(self.definition['fixture']['scenario'], self.scenario) as server:
                    def send(method, uri, body=None):
                        data = b'' if body is None else json.dumps(body).encode()
                        request = Request(server.environment()['LARKSUITE_CLI_AUTH_PROXY'] + uri,
                                          data=None if body is None else data, method=method,
                                          headers=signed_headers(method, uri, data))
                        with urlopen(request, timeout=5) as response:
                            return json.load(response)['data']
                    self.send = send
                    answer = self.review() if case == 'release-review' else self.handoff()
                self.scenario.close()
                # An agent-written receipt is overwritten with post-run server observations.
                (self.workspace / 'backend-receipt.json').write_text('{}')
                server.write_receipt(self.workspace)
                (self.workspace / 'answer.json').write_text(json.dumps(answer))
                request = {'workspace': str(self.workspace), 'task_id': self.definition['id'],
                           'fixture': self.definition['fixture'], 'expected': self.definition['expected']}
                result = subprocess.run([sys.executable, str(ROOT / 'evaluation/scenario_verifier.py')],
                                        input=json.dumps(request), text=True, capture_output=True, timeout=10)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertTrue(json.loads(result.stdout)['success'])


if __name__ == '__main__':
    unittest.main()
