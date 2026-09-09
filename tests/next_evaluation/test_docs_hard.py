"""Harder Docs contract controls: no models or live account access."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
from paired import docs_hard_v1 as hard
from scenarios import docs as native
from scenario_fixture import ScenarioFixture, load_scenario
from section_fixture import signed_headers
from codex_runner import cli_environment, validate_output_schema


class HardDocsTests(unittest.TestCase):
    def call(self, scenario, records, method, uri, body=None):
        raw = json.dumps(body or {}).encode()
        reply = scenario.dispatch(method, uri, raw, signed_headers(method, uri, raw))
        records.append({'route': reply.route, 'status': reply.status, 'signature_valid': True,
                        'proof': copy.deepcopy(reply.proof)})
        return reply.json['data']

    def receipt(self, scenario, records):
        return {'scenario': 'docs-hard-v1', 'state': scenario.snapshot(), 'requests': records}

    def check(self, case, scenario, records, answer=None):
        return hard.verify(case, None, hard.ANSWERS[case] if answer is None else answer,
                           self.receipt(scenario, records), {})

    def test_manifest_and_bounded_size_are_versioned_and_closed(self):
        manifest = json.loads((ROOT / 'evaluation/tasks.docs-hard.json').read_text())
        self.assertEqual(manifest['tasks'], [hard.task(case) for case in hard.CASES])
        for task in manifest['tasks']:
            validate_output_schema(task['fixture']['answer_schema'])
            self.assertEqual(load_scenario('docs-hard-v1', case_id=task['id']).TASK, task)
        state = hard.initial(hard.CASES[1])
        self.assertEqual(len(state['blocks']), 181)
        self.assertTrue(10000 < len(state['full_xml'].encode()) < 35000)

    def test_explicit_launcher_opt_in_keeps_original_guide_and_balanced_18(self):
        from unittest.mock import patch
        import run_read_batch
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory); binary = root / 'binary'; binary.touch()
            args = ['run_read_batch', '--task-kind', 'docs-hard', '--original-comparison', '--guidance-delivery', 'file',
                    '--model', 'gpt-5.6-luna', '--reasoning', 'medium']
            for key, value in {'official-source': ROOT, 'official-cli': binary, 'next-source': ROOT, 'next-cli': binary,
                               'codex': binary, 'auth-home': root / 'unused-auth', 'output-root': root / 'results'}.items():
                args += ['--' + key, str(value)]
            def inspect(command, **kwargs):
                config = json.loads(Path(command[command.index('--config') + 1]).read_text())
                self.assertEqual(set(config['arms']), {'official', 'next'})
                self.assertEqual(config['guidance_delivery'], 'file')
                self.assertEqual(Path(config['arms']['official']['skill']).read_bytes(), (ROOT / 'skills/lark-doc/SKILL.md').read_bytes())
                self.assertEqual(Path(command[command.index('--tasks') + 1]).name, 'tasks.docs-hard.json')
                self.assertEqual(command[command.index('--seed') + 1], '20260909')
                self.assertEqual(command[command.index('--repeats') + 1], '3')
                return subprocess.CompletedProcess(command, 0)
            with patch.object(sys, 'argv', args), patch.object(run_read_batch.subprocess, 'run', side_effect=inspect) as run, patch('builtins.print'):
                self.assertEqual(run_read_batch.main(), 0)
                run.assert_called_once()
            self.assertFalse((root / 'unused-auth').exists())

    def test_rich_native_hierarchy_and_semantic_answer(self):
        case = hard.CASES[0]; scenario = hard.Scenario(case); records = []
        data = self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks')
        blocks = {b['block_id']: b for b in data['items']}
        self.assertEqual(blocks[hard.DOCUMENT]['children'][:2], ['hardHeading', 'hardTable'])
        self.assertEqual(blocks['hardTable']['table']['property'], {'row_size': 3, 'column_size': 3})
        self.assertEqual(blocks['hardPrepare']['block_type'], 13)
        self.assertEqual(blocks['hardPrepare']['children'], ['hardSeal', 'hardLabel'])
        self.assertEqual(blocks['hardSeal']['parent_id'], 'hardPrepare')
        self.assertEqual(blocks['hardValue12']['text']['elements'][0]['text_run']['content'], '')
        self.assertTrue(self.check(case, scenario, records))
        for changed in ({'note': None}, {'sku': '1'}, {'ordered_steps': ['Release', 'Prepare']},
                        {'prepare_children': ['Read label', 'Inspect seal']}, {'table_block_id': 'hardPrepare'}):
            self.assertFalse(self.check(case, scenario, records, {**hard.ANSWERS[case], **changed}))
        scenario.state['blocks'][-1]['token'] = 'lost'
        self.assertFalse(self.check(case, scenario, records))

    def test_child_references_and_empty_text_do_not_prove_returned_blocks(self):
        scenario = hard.Scenario(hard.CASES[0]); records = []
        for identifier in [hard.DOCUMENT, 'hardValue10', 'hardValue11', 'hardCell12',
                           'hardPrepare', 'hardRelease', 'hardSeal', 'hardLabel']:
            self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks/' + identifier)
        observed = {identifier for record in records for identifier in record['proof']['observed_block_ids']}
        self.assertNotIn('hardTable', observed)
        self.assertNotIn('hardValue12', observed)
        self.assertFalse(self.check(scenario.case, scenario, records))
        scenario = hard.Scenario(hard.CASES[0]); records = []
        self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {
            'export_option': {'export_block_id': True}, 'read_option': {'read_mode': 'outline'}})
        self.assertFalse(self.check(scenario.case, scenario, records))
        scenario = hard.Scenario(hard.CASES[0]); records = []
        self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {'format': 'xml'})
        self.assertFalse(self.check(scenario.case, scenario, records))

    def test_full_scoped_and_raw_read_routes_without_forced_workflow(self):
        for option in ({}, {'read_mode': 'section', 'start_block_id': 'hardHeading'}):
            scenario = hard.Scenario(hard.CASES[0]); records = []
            data = self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {
                'format': 'xml', 'export_option': {'export_block_id': True}, 'read_option': option})
            self.assertIn('<table', data['document']['content'])
            self.assertIn('seq="2"', data['document']['content'])
            self.assertTrue(self.check(scenario.case, scenario, records))
        scenario = hard.Scenario(hard.CASES[0]); records = []
        self.call(scenario, records, 'GET', native.DOCX_PATH + '/raw_content')
        self.assertFalse(self.check(scenario.case, scenario, records))  # Missing native references.
        self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks')
        self.assertTrue(self.check(scenario.case, scenario, records))
        with self.assertRaisesRegex(ValueError, '^unsupported strategy:'):
            self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {
                'read_option': {'read_mode': 'range', 'start_block_id': 'hardTable', 'end_block_id': 'hardTable'}})

    def test_large_read_supports_pagination_and_focused_section(self):
        scenario = hard.Scenario(hard.CASES[1]); records = []
        page = self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks?page_size=100')
        self.assertTrue(page['has_more'])
        tail = self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks?page_size=100&page_token=' + page['page_token'])
        self.assertFalse(tail['has_more'])
        self.assertEqual(len(page['items']) + len(tail['items']), 181)
        self.assertTrue(self.check(scenario.case, scenario, records))
        for option in ({'read_mode': 'section', 'start_block_id': 'hardSection11'},
                       {'read_mode': 'keyword', 'keyword': 'Orion final approval'}):
            scenario = hard.Scenario(hard.CASES[1]); records = []
            self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {
                'format': 'xml', 'export_option': {'export_block_id': True}, 'read_option': option})
            self.assertTrue(self.check(scenario.case, scenario, records))
        scenario = hard.Scenario(hard.CASES[1]); records = []
        self.call(scenario, records, 'POST', native.AI_PATH + '/fetch', {
            'export_option': {'export_block_id': True}, 'read_option': {'read_mode': 'section', 'start_block_id': 'hardSection10'}})
        self.assertFalse(self.check(scenario.case, scenario, records))

    def complete_edits(self, scenario, records, strategy):
        if strategy == 'patch':
            for n, replacements in [(2, [('Draft', 'Final'), ('amber', 'green')]), (3, [('legacy', 'current')])]:
                block = scenario.state['blocks'][n]
                elements = native.wire_block(block)['text']['elements']
                for element in elements:
                    for old, new in replacements:
                        element['text_run']['content'] = element['text_run']['content'].replace(old, new)
                self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'],
                          {'update_text_elements': {'elements': elements}})
        elif strategy == 'block_replace':
            for identifier, content in [('hardEditA', '<p>Final status; <b>green</b> flag — 你好</p>'),
                ('hardEditB', '<p>Open <a href="https://example.com/reference">current portal</a> for résumé 😀.</p>')]:
                self.call(scenario, records, 'PUT', native.AI_PATH, {'command': 'block_replace', 'format': 'xml',
                    'block_id': identifier, 'content': content})
        else:
            for old, new in [('Draft', 'Final'), ('amber', '<b>green</b>'),
                             ('legacy', '<a href="https://example.com/reference">current</a>')]:
                self.call(scenario, records, 'PUT', native.AI_PATH, {'command': 'str_replace', 'format': 'xml',
                    'pattern': old, 'content': new})

    def test_original_rich_update_and_native_patch_preserve_everything(self):
        for strategy in ('patch', 'block_replace', 'str_replace'):
            with self.subTest(strategy=strategy):
                scenario = hard.Scenario(hard.CASES[2]); records = []
                self.complete_edits(scenario, records, strategy)
                self.assertTrue(self.check(scenario.case, scenario, records))
                self.assertEqual(hard.preserved(scenario.state), hard.preserved(hard.initial(scenario.case)))
                self.assertFalse(self.check(scenario.case, scenario, records[:-1]))

    def test_valid_signed_mutations_enforce_semantics_not_only_hashes(self):
        case = hard.CASES[2]
        # An unrelated change and restoration is still forbidden in the chain.
        scenario = hard.Scenario(case); records = []
        for content in ('changed', '你好'):
            self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/hardValue11',
                      {'update_text_elements': {'elements': [{'text_run': {'content': content, 'text_element_style': {}}}]}})
        self.complete_edits(scenario, records, 'patch')
        self.assertFalse(self.check(case, scenario, records))
        # Correct words with incorrect native marks fail despite authentic hashes.
        for wrong in ('bold', 'link'):
            scenario = hard.Scenario(case); records = []
            self.complete_edits(scenario, records, 'patch')
            block = scenario.state['blocks'][2 if wrong == 'bold' else 3]
            elements = hard.wire(block)['text']['elements']
            for element in elements:
                style = element['text_run']['text_element_style']
                if wrong == 'bold':
                    style.pop('bold', None)
                elif 'link' in style:
                    style['link']['url'] = 'https://example.com/wrong'
            self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'],
                      {'update_text_elements': {'elements': elements}})
            self.assertFalse(self.check(case, scenario, records))
        # Native run boundaries have no hidden identity requirement.
        scenario = hard.Scenario(case); records = []
        self.complete_edits(scenario, records, 'patch')
        block = scenario.state['blocks'][2]
        elements = []
        for element in hard.wire(block)['text']['elements']:
            for char in element['text_run']['content']:
                part = copy.deepcopy(element); part['text_run']['content'] = char; elements.append(part)
        self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'],
                  {'update_text_elements': {'elements': elements}})
        self.assertTrue(self.check(case, scenario, records))

    def test_unsupported_request_preserves_known_failures_but_not_missing_proof(self):
        from scenario_verifier import evaluate
        from urllib.error import HTTPError
        from urllib.request import Request, urlopen
        for mode in ('wrong_answer', 'complete_read', 'missing_edit', 'wrong_bold', 'repaired_bold', 'unrelated_restore'):
            case = hard.CASES[0] if mode in ('wrong_answer', 'complete_read') else hard.CASES[2]
            scenario = hard.Scenario(case); records = []
            answer = copy.deepcopy(hard.ANSWERS[case])
            if case == hard.CASES[0]:
                self.call(scenario, records, 'GET', native.DOCX_PATH + '/blocks')
                if mode == 'wrong_answer': answer['sku'] = 'wrong'
            elif mode != 'missing_edit':
                if mode == 'unrelated_restore':
                    for content in ('changed', '你好'):
                        self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/hardValue11',
                            {'update_text_elements': {'elements': [{'text_run': {'content': content, 'text_element_style': {}}}]}})
                self.complete_edits(scenario, records, 'patch')
                if mode in ('wrong_bold', 'repaired_bold'):
                    block = scenario.state['blocks'][2]
                    elements = hard.wire(block)['text']['elements']
                    for element in elements: element['text_run']['text_element_style'].pop('bold', None)
                    self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'],
                              {'update_text_elements': {'elements': elements}})
                    if mode == 'repaired_bold':
                        elements = hard.wire(hard.desired_edit_state()['blocks'][2])['text']['elements']
                        self.call(scenario, records, 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'],
                                  {'update_text_elements': {'elements': elements}})
            with tempfile.TemporaryDirectory() as directory:
                workspace = Path(directory)
                request = {'task_id': case, 'workspace': directory, 'fixture': hard.task(case)['fixture'], 'expected': hard.task(case)['expected']}
                (workspace / 'answer.json').write_text(json.dumps(answer))
                (workspace / 'backend-receipt.json').write_text(json.dumps(self.receipt(scenario, records)))
                before = evaluate(request)
                with ScenarioFixture('docs-hard-v1', scenario) as fixture:
                    uri = native.DOCX_PATH + '/blocks?document_revision_id=' + str(scenario.state['revision_id'] - 1)
                    raw = b'{}'
                    with self.assertRaises(HTTPError) as caught:
                        urlopen(Request(fixture.environment()['LARKSUITE_CLI_AUTH_PROXY'] + uri, data=raw, method='GET', headers=signed_headers('GET', uri, raw)))
                    caught.exception.close()
                rejected = fixture.receipt()['requests']
                self.assertEqual(rejected[0]['rejection'], 'unsupported_strategy')
                receipt = self.receipt(scenario, records + rejected)
                (workspace / 'backend-receipt.json').write_text(json.dumps(receipt))
                after = evaluate(request)
                self.assertFalse(after['success'])
                self.assertEqual(after['fixture_unsupported'], mode in ('complete_read', 'missing_edit', 'repaired_bold'))
                if mode == 'wrong_answer':
                    self.assertFalse(before['success'])
                    self.assertIn('paired_docs_answer_mismatch', after['verification_failures'])
                elif mode == 'wrong_bold':
                    self.assertIn('paired_docs_bold_mismatch', after['verification_failures'])
                elif mode == 'unrelated_restore':
                    self.assertIn('paired_docs_unrelated_state_changed', after['verification_failures'])
                elif mode in ('complete_read', 'repaired_bold'):
                    self.assertTrue(before['success'])

    def test_mutation_fidelity_and_signed_chain_negative_controls(self):
        scenario = hard.Scenario(hard.CASES[2]); records = []
        self.complete_edits(scenario, records, 'patch')
        receipt = self.receipt(scenario, records)
        corruptions = [lambda r: r['requests'][0].update(signature_valid=False),
            lambda r: r['state']['final']['blocks'][2]['runs'][1]['style'].update(bold=False),
            lambda r: r['state']['final']['blocks'][3]['runs'][1]['style']['link'].update(url='https://example.com/wrong'),
            lambda r: r['state']['mutations'][0]['after']['blocks'][-1].update(token='lost'),
            lambda r: r['state']['mutations'][0].update(before_hash='forged'),
            lambda r: r['state']['final']['blocks'][0]['children'].reverse(),
            lambda r: r['state']['final']['permissions'].update(public=True)]
        for corrupt in corruptions:
            bad = copy.deepcopy(receipt); corrupt(bad)
            self.assertFalse(hard.verify(scenario.case, None, hard.ANSWERS[scenario.case], bad, {}))
        untouched = hard.Scenario(hard.CASES[0])
        with self.assertRaises(ValueError):
            self.call(untouched, [], 'PATCH', native.DOCX_PATH + '/blocks/hardHeading', {})
        self.assertEqual(untouched.state, hard.initial(untouched.case))


@unittest.skipUnless(os.environ.get('EVAL_PAIRED_BINARIES'), 'set EVAL_PAIRED_BINARIES JSON paths for real CLI loopback controls')
class HardDocsBinaryTests(unittest.TestCase):
    def test_existing_fetch_and_updates_on_each_binary(self):
        for binary in json.loads(os.environ['EVAL_PAIRED_BINARIES']):
            for case in hard.CASES:
                scenario = hard.Scenario(case)
                with self.subTest(binary=Path(binary).name, case=case), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ('home', 'tmp', 'feishu-config'): (workspace / name).mkdir()
                    with ScenarioFixture('docs-hard-v1', scenario) as fixture:
                        env = {**cli_environment(workspace), **fixture.environment()}
                        def run(args):
                            result = subprocess.run([binary, *args, '--as', 'user'], cwd=workspace, env=env,
                                                    capture_output=True, text=True, timeout=20)
                            self.assertEqual(result.returncode, 0, result.stderr)
                        if case != hard.CASES[2]:
                            run(['docs', '+fetch', '--doc', hard.DOCUMENT, '--doc-format', 'xml', '--detail', 'full'])
                        else:
                            for old, new in [('Draft', 'Final'), ('amber', '<b>green</b>'),
                                             ('legacy', '<a href="https://example.com/reference">current</a>')]:
                                run(['docs', '+update', '--doc', hard.DOCUMENT, '--command', 'str_replace',
                                     '--doc-format', 'xml', '--pattern', old, '--content', new])
                    self.assertTrue(hard.verify(case, workspace, hard.ANSWERS[case], fixture.receipt(), {}))

    def test_native_api_read_and_styled_patch_on_each_binary(self):
        for binary in json.loads(os.environ['EVAL_PAIRED_BINARIES']):
            for case in (hard.CASES[0], hard.CASES[2]):
                scenario = hard.Scenario(case)
                with self.subTest(binary=Path(binary).name, case=case), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ('home', 'tmp', 'feishu-config'): (workspace / name).mkdir()
                    with ScenarioFixture('docs-hard-v1', scenario) as fixture:
                        env = {**cli_environment(workspace), **fixture.environment()}
                        def run(args):
                            result = subprocess.run([binary, *args, '--as', 'user'], cwd=workspace, env=env,
                                                    capture_output=True, text=True, timeout=20)
                            self.assertEqual(result.returncode, 0, result.stderr)
                        run(['api', 'GET', native.DOCX_PATH + '/blocks'])
                        if case == hard.CASES[2]:
                            for n, replacements in [(2, [('Draft', 'Final'), ('amber', 'green')]), (3, [('legacy', 'current')])]:
                                block = scenario.state['blocks'][n]
                                elements = hard.wire(block)['text']['elements']
                                for element in elements:
                                    for old, new in replacements:
                                        element['text_run']['content'] = element['text_run']['content'].replace(old, new)
                                run(['api', 'PATCH', native.DOCX_PATH + '/blocks/' + block['block_id'], '--data',
                                     json.dumps({'update_text_elements': {'elements': elements}})])
                    self.assertTrue(hard.verify(case, workspace, hard.ANSWERS[case], fixture.receipt(), {}))

    @unittest.skipUnless(os.environ.get('EVAL_DOCS_HARD_NEXT'), 'set EVAL_DOCS_HARD_NEXT for new candidate pull/push controls')
    def test_candidate_read_views_focused_selection_and_multi_edit_push(self):
        binary = os.environ['EVAL_DOCS_HARD_NEXT']
        for case in hard.CASES:
            scenario = hard.Scenario(case)
            with self.subTest(case=case), tempfile.TemporaryDirectory() as directory:
                workspace = Path(directory)
                for name in ('home', 'tmp', 'feishu-config'): (workspace / name).mkdir()
                with ScenarioFixture('docs-hard-v1', scenario) as fixture:
                    env = {**cli_environment(workspace), **fixture.environment()}
                    def run(args):
                        result = subprocess.run([binary, *args, '--as', 'user'], cwd=workspace, env=env,
                                                capture_output=True, text=True, timeout=20)
                        self.assertEqual(result.returncode, 0, result.stderr)
                        return json.loads(result.stdout)['data']
                    if case == hard.CASES[0]:
                        data = run(['docs', '+pull', '--doc', hard.DOCUMENT])
                        self.assertIn('001', data['markdown'])
                        self.assertIn('你好', data['markdown'])
                        self.assertIn('Inspect seal', data['markdown'])
                    elif case == hard.CASES[1]:
                        data = run(['docs', '+pull', '--doc', hard.DOCUMENT, '--scope', 'section', '--start-block-id', 'hardSection11'])
                        self.assertIn(r'OR\-0731', data['markdown'])
                        self.assertNotIn('Archived launch gate Orion archive', data['markdown'])
                        self.assertFalse(data['complete'])
                        self.assertLess(data['returned_blocks'], data['total_blocks'])
                    else:
                        run(['docs', '+pull', '--doc', hard.DOCUMENT, '--output', './note.md'])
                        path = workspace / 'note.md'
                        content = path.read_text().replace('Draft status', 'Final status').replace('amber', 'green').replace('legacy portal', 'current portal')
                        path.write_text(content)
                        data = run(['docs', '+push', '--file', './note.md'])
                        self.assertTrue(data['verified'])
                        self.assertEqual([b['text'] for b in data['changed_blocks']], hard.ANSWERS[case]['paragraphs'])
                self.assertTrue(hard.verify(case, workspace, hard.ANSWERS[case], fixture.receipt(), {}))


if __name__ == '__main__':
    unittest.main()
