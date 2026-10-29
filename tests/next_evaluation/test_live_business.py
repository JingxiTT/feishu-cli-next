# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline verifier checks, never model-efficiency or tenant evidence."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
import live_workspace
from live_domains import base, docs, sheets
from live_domains.session import Blocked, Session, exact


class BusinessTests(unittest.TestCase):
    def test_unverifiable_handoff_is_rejected_before_any_live_setup(self):
        with patch.object(live_workspace, 'session') as setup:
            with self.assertRaises(Blocked):
                live_workspace.run({'backend': 'live-feishu-v1',
                                    'task': {'id': 'business-file-handoff'}}, None, None, None)
            setup.assert_not_called()

    def test_only_existing_scope_workflows_are_registered_and_sheet_seed_is_supported(self):
        directory = Path(__file__).resolve().parents[2] / 'evaluation'
        manifest = json.loads((directory / 'tasks.live-business.json').read_text())
        extended = json.loads((directory / 'tasks.live-business-extended.json').read_text())
        complex_tasks = json.loads((directory / 'tasks.live-business-complex.json').read_text())
        self.assertEqual(manifest['backend'], 'live-feishu-v1')
        self.assertEqual(extended['backend'], 'live-feishu-v1')
        identifiers = [t['id'] for t in manifest['tasks'] + extended['tasks'] + complex_tasks['tasks']]
        self.assertEqual(len(identifiers), 13)
        self.assertEqual(set(identifiers), set(live_workspace.SINGLE) | live_workspace.CROSS)
        self.assertNotIn('task', json.dumps(identifiers))
        self.assertTrue(all(value is not None for row in sheets.VALUES for value in row))
        self.assertEqual(len(base.ROWS), 205)

    def test_large_base_audit_crosses_cli_page_limit_and_retains_zero_false_and_credit_rows(self):
        rows = base.seed_rows('common-base-audit-large')
        self.assertEqual(len(rows), 2505)
        self.assertEqual(rows[-1]['Title'], 'INV-02504')
        self.assertTrue(any(row['Amount'] < 0 for row in rows))
        before = {'large-' + str(i): row for i, row in enumerate(rows)}
        result = base.audit_report(before)
        self.assertTrue(any(int(key[6:]) >= 2000 for key in result['record_ids']))
        self.assertTrue(result['zero_amount_record_ids'])
        self.assertEqual(result['record_count'], len(result['record_ids']))
        self.assertTrue(all(before[key]['Checked'] is True and before[key]['Amount'] >= 4
                            for key in result['record_ids']))

    def test_conditional_close_preserves_false_zero_and_every_unselected_value(self):
        before = {'selected': {'Title': '001', 'Amount': 4, 'Checked': True, 'Note': 'old'},
                  'zero': {'Title': '002', 'Amount': 0, 'Checked': True, 'Note': 'guard'},
                  'false': {'Title': '003', 'Amount': 9, 'Checked': False, 'Note': 'guard'}}
        after = base.closed_records(before)
        self.assertEqual(after['selected'], {**before['selected'], 'Checked': False, 'Note': 'Closed R42'})
        self.assertEqual(after['zero'], before['zero'])
        self.assertEqual(after['false'], before['false'])
        self.assertEqual(before['selected']['Note'], 'old')

    def test_long_document_audit_includes_last_page_and_excludes_decoys_and_credits(self):
        lines = docs.audit_lines()
        self.assertEqual(len(lines), 520)
        native = [{'block_id': 'block-' + str(i), 'block_type': 2,
                   'text': {'elements': [{'text_run': {'content': line}}]}}
                  for i, line in enumerate(lines)]
        report = docs.audit_report(native)
        self.assertIn('block-519', report['block_ids'])
        self.assertIn('INV-00519', report['invoice_codes'])
        self.assertNotIn('block-0', report['block_ids'])
        self.assertGreater(report['record_count'], 1)
        self.assertGreater(report['total_amount'], 0)

    def test_document_audit_rejects_correct_magnitude_as_string_and_source_changes(self):
        before = [{'block_id': 'invoice', 'block_type': 2, 'text': {'elements': [
            {'text_run': {'content': 'R42 | APPROVED | INV-00519 | Amount=10 | résumé 你好 😀'}}]}}]
        report = docs.audit_report(before)
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'audit.json'
            s = Mock(data={'document_id': 'test-doc', 'before': before},
                     request={'task': {'id': 'common-docs-audit-large'}})
            s.artifact.return_value = artifact
            now = copy.deepcopy(before)
            with patch.object(docs, 'blocks', return_value=now):
                artifact.write_text(json.dumps(report))
                self.assertTrue(docs.verify(s, {'path': 'audit.json'}))
                for key in ['total_amount', 'record_count']:
                    artifact.write_text(json.dumps({**report, key: str(report[key])}))
                    self.assertFalse(docs.verify(s, {'path': 'audit.json'}))
                artifact.write_text(json.dumps(report))
                now[0]['text']['elements'][0]['text_run']['text_element_style'] = {'bold': True}
                self.assertFalse(docs.verify(s, {'path': 'audit.json'}))

    def test_settlement_must_preserve_both_sources_and_native_document_blocks(self):
        before = {'selected': {'Title': '001', 'Amount': 4, 'Checked': True, 'Note': 'old'},
                  'guard': {'Title': '002', 'Amount': 0, 'Checked': False, 'Note': 'keep'}}
        grid = {mode: [[{'value': 'guard'} for _ in range(6)] for _ in range(6)]
                for mode in ['raw', 'formula']}
        doc_before = [{'block_id': 'root', 'block_type': 1, 'children': ['guard']},
                      {'block_id': 'guard', 'block_type': 2, 'text': {'elements': []}}]
        report = live_workspace.settlement_report(before)
        doc_after = copy.deepcopy(doc_before)
        doc_after[0]['children'].append('new')
        doc_after.append({'block_id': 'new', 'block_type': 2, 'text': {'elements': [
            {'text_run': {'content': live_workspace.settlement_line(report)}}]}})
        sessions = {'base': Mock(data={'before': before}), 'sheets': Mock(data={'before': grid}),
                    'docs': Mock(data={'before': doc_before, 'document_id': 'test-doc'})}
        closed = base.closed_records(before)
        changed_grid = sheets.expected_snapshot(grid, {(1, 1): 4, (1, 3): 'Closed R42'})
        with patch.object(base, 'records', return_value=closed), \
             patch.object(sheets, 'snapshot', return_value=changed_grid), \
             patch.object(docs, 'blocks', return_value=doc_after):
            self.assertTrue(live_workspace.verify_settlement(sessions, report))
            closed['guard']['Note'] = 'changed'
            self.assertFalse(live_workspace.verify_settlement(sessions, report))

    def test_unknown_mutation_is_journaled_and_cannot_be_replayed(self):
        s = Session.__new__(Session)
        s.data = {'pending': None, 'owned': []}
        s.save = Mock()
        operation = Mock(side_effect=TimeoutError())
        with self.assertRaises(TimeoutError):
            s.mutation('create', operation)
        self.assertEqual(s.data['pending'], 'create')
        with self.assertRaises(Blocked):
            s.mutation('create', operation)
        operation.assert_called_once()

    def test_all_pages_required_and_repeated_cursor_rejected(self):
        s = Session.__new__(Session)
        s.api = Mock(side_effect=[{'items': [1], 'has_more': True, 'page_token': 'a'},
                                  {'items': [2], 'has_more': False}])
        self.assertEqual(s.pages('/test'), [1, 2])
        self.assertEqual(s.api.call_args.args[2]['page_token'], 'a')
        for values in [[{'items': []}], [{'items': [], 'has_more': True, 'page_token': 'a'}] * 2]:
            s.api = Mock(side_effect=values)
            with self.assertRaises(Blocked):
                s.pages('/test')

    def test_private_parent_requires_user_authorization_binding_and_acl(self):
        s = Session.__new__(Session)
        s.resources = {'private_parent': {'creation_authorized_by_user': True, 'label': '[TEST]',
                                         'account_binding': 'verified-binding', 'folder_token': 'test-folder'}}
        s.data = {'binding': 'verified-binding'}
        s.private_drive = Mock()
        self.assertEqual(s.parent(), 'test-folder')
        s.private_drive.assert_called_once_with('test-folder', 'folder')
        s.resources['private_parent']['account_binding'] = 'other-account'
        with self.assertRaises(Blocked):
            s.parent()

    def test_private_acl_rejects_other_members_open_links_and_wrong_owner(self):
        s = Session.__new__(Session)
        s.account = {'open_id': 'test-owner'}
        settings = {'permission_public': {'link_share_entity': 'closed'}}
        members = {'items': [{'member_type': 'openid', 'member_id': 'test-owner',
                              'perm': 'full_access', 'perm_type': 'container'}]}
        metadata = {'metas': [{'doc_token': 'test-folder', 'doc_type': 'folder', 'owner_id': 'test-owner'}]}
        for mode in ['valid', 'open-link', 'other-member', 'wrong-owner']:
            a, b, c = copy.deepcopy((settings, members, metadata))
            if mode == 'open-link':
                a['permission_public']['link_share_entity'] = 'tenant_readable'
            if mode == 'other-member':
                b['items'].append({**b['items'][0], 'member_id': 'other-person'})
            if mode == 'wrong-owner':
                c['metas'][0]['owner_id'] = 'other-person'
            s.call = Mock(side_effect=[a, b])
            s.api = Mock(return_value=c)
            if mode == 'valid':
                s.private_drive('test-folder', 'folder')
            else:
                with self.assertRaises(Blocked):
                    s.private_drive('test-folder', 'folder')

    def test_native_document_append_preserves_block_ids_styles_and_order(self):
        before = [{'block_id': 'root', 'block_type': 1, 'children': ['guard']},
                  {'block_id': 'guard', 'block_type': 2, 'text': {'elements': [
                      {'text_run': {'content': 'Résumé 😀', 'text_element_style': {'bold': True}}}]}}]
        after = copy.deepcopy(before)
        after[0]['children'].append('summary')
        after.append({'block_id': 'summary', 'block_type': 2, 'text': {'elements': [
            {'text_run': {'content': 'Totals'}}]}})
        self.assertTrue(live_workspace.unchanged_blocks(before, after, 'Totals'))
        after[1]['text']['elements'][0]['text_run']['text_element_style']['bold'] = False
        self.assertFalse(live_workspace.unchanged_blocks(before, after, 'Totals'))

    def test_type_preservation_does_not_equate_false_zero_null_and_omission(self):
        self.assertFalse(exact(False, 0))
        self.assertFalse(exact({'value': None}, {}))
        self.assertFalse(exact('001', 1))
        self.assertFalse(exact('', None))

    def test_native_base_offset_continues_by_returned_count_and_keeps_every_record(self):
        s = Mock(data={'base_token': 'test-base', 'table_id': 'test-table'})
        shared = {'fields': ['Amount'], 'field_id_list': ['field'], 'field_type_list': ['number'],
                  'timezone': 'UTC', 'rev': 1}
        s.api.side_effect = [
            {**shared, 'record_id_list': ['a', 'b'], 'data': [[0], [2]], 'has_more': True},
            {**shared, 'record_id_list': ['c'], 'data': [[3]], 'has_more': False}]
        self.assertEqual(base.records(s), {'a': {'Amount': 0}, 'b': {'Amount': 2}, 'c': {'Amount': 3}})
        self.assertEqual(s.api.call_args.args[2]['offset'], 2)

    def test_base_revision_change_during_pagination_rejects_the_snapshot(self):
        s = Mock(data={'base_token': 'test-base', 'table_id': 'test-table'})
        shared = {'fields': ['Amount'], 'field_id_list': ['field'],
                  'field_type_list': ['number'], 'timezone': 'UTC'}
        s.api.side_effect = [
            {**shared, 'record_id_list': ['a'], 'data': [[0]], 'has_more': True, 'rev': 1},
            {**shared, 'record_id_list': ['b'], 'data': [[3]], 'has_more': False, 'rev': 2}]
        with self.assertRaisesRegex(Blocked, '^base_revision_changed$'):
            base.records(s)
        self.assertEqual(s.api.call_count, 2)
        s.mutation.assert_not_called()

    def test_sheet_update_keeps_raw_types_distinct_from_native_formula_rendering(self):
        before = {mode: [[{'value': 'guard'} for _ in range(6)] for _ in range(6)]
                  for mode in ['raw', 'formula']}
        before['raw'][1][1]['value'] = 2
        before['raw'][1][3]['value'] = '你好 😀'
        before['raw'][1][4]['value'] = 4
        before['formula'][1][1]['value'] = '2'
        before['formula'][1][4] = {'formula': '=B2*2', 'value': 4}
        after = copy.deepcopy(before)
        after['raw'][1][1]['value'] = 0
        after['raw'][1][3]['value'] = ''
        after['raw'][1][4]['value'] = 0
        after['formula'][1][1]['value'] = '0'
        after['formula'][1][3].pop('value')
        after['formula'][1][4]['value'] = 0
        s = Mock(data={'spreadsheet_token': 'test-sheet', 'before': before},
                 request={'task': {'id': 'common-sheets-update'}})
        with patch.object(sheets, 'snapshot', return_value=after):
            self.assertTrue(sheets.verify(s, {'spreadsheet_token': 'test-sheet'}))
            after['raw'][1][1]['value'] = '0'
            self.assertFalse(sheets.verify(s, {'spreadsheet_token': 'test-sheet'}))

    def test_csv_intake_requires_exact_notes_types_and_unchanged_guards(self):
        before = {mode: [[{'value': 'guard', 'cell_styles': {'font_size': 10}}
                          for _ in range(6)] for _ in range(6)] for mode in ['raw', 'formula']}
        after = sheets.expected_snapshot(before, sheets.INTAKE_CHANGES)
        answer = {'updated_codes': ['001', '003'], 'skipped_codes': ['002', '004']}
        s = Mock(data={'spreadsheet_token': 'test-sheet', 'before': before},
                 request={'task': {'id': 'common-sheets-intake'}})
        with patch.object(sheets, 'snapshot', return_value=after):
            self.assertTrue(sheets.verify(s, answer))
            for row, col, value in [(3, 3, 'Approved, résumé 你好 😀'), (3, 1, '7.5'),
                                    (1, 3, None), (4, 1, 99), (1, 0, 1)]:
                original = after['raw'][row][col]['value']
                after['raw'][row][col]['value'] = value
                self.assertFalse(sheets.verify(s, answer))
                after['raw'][row][col]['value'] = original
            after['formula'][3][4]['formula'] = '=15'
            self.assertFalse(sheets.verify(s, answer))

    def test_export_requires_computed_values_and_formulas_and_no_remote_changes(self):
        before = {mode: [[{'value': sheets.VALUES[r][c]} for c in range(6)]
                          for r in range(6)] for mode in ['raw', 'formula']}
        cells = copy.deepcopy(before['raw'])
        for row in range(1, 5):
            cells[row][4] = {'value': row * 2, 'formula': '=B' + str(row + 1) + '*2'}
            before['raw'][row][4]['value'] = row * 2
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'export.json'
            s = Mock(data={'spreadsheet_token': 'test-sheet', 'before': before},
                     request={'task': {'id': 'common-sheets-export'}})
            s.artifact.return_value = artifact
            with patch.object(sheets, 'snapshot', return_value=before):
                artifact.write_text(json.dumps({'range': sheets.EXTENT, 'cells': cells}))
                self.assertTrue(sheets.verify(s, {'path': 'export.json'}))
                for row, col, value in [(1, 0, 1), (2, 2, 0), (2, 2, 'FALSE'),
                                        (1, 2, 'TRUE'), (2, 1, '0'), (2, 3, None)]:
                    original = cells[row][col]['value']
                    cells[row][col]['value'] = value
                    artifact.write_text(json.dumps({'range': sheets.EXTENT, 'cells': cells}))
                    self.assertFalse(sheets.verify(s, {'path': 'export.json'}))
                    cells[row][col]['value'] = original
                cells[1][4].pop('formula')
                artifact.write_text(json.dumps({'range': sheets.EXTENT, 'cells': cells}))
                self.assertFalse(sheets.verify(s, {'path': 'export.json'}))

    def test_cleanup_uses_only_journaled_ids_without_drive_listing(self):
        s = Session.__new__(Session)
        s.data = {'pending': None, 'folder': 'test-folder', 'owned': [
            {'kind': 'folder', 'id': 'test-folder'}, {'kind': 'docx', 'id': 'test-doc'},
            {'kind': 'sheet', 'id': 'already-deleted', 'deleted': True}]}
        s.private_drive = Mock()
        s.mutation = Mock(return_value={'deleted': True})
        s.api = Mock(side_effect=AssertionError('broad listing must not run'))
        s.cleanup_drive()
        self.assertEqual([call.kwargs['target']['id'] for call in s.mutation.call_args_list], ['test-doc', 'test-folder'])
        s.data['pending'] = 'unresolved'
        with self.assertRaises(Blocked):
            s.cleanup_drive()

    def test_cleanup_cannot_succeed_when_a_created_folder_coordinate_was_not_saved(self):
        s = Session.__new__(Session)
        s.data = {'pending': None, 'owned': [{'kind': 'folder', 'id': 'test-folder'}]}
        with self.assertRaises(Blocked):
            s.cleanup_drive()

    def test_artifact_cannot_escape_workspace_through_parent_or_symlink(self):
        with tempfile.TemporaryDirectory() as directory:
            s = Session.__new__(Session)
            s.workspace = Path(directory) / 'workspace'
            s.workspace.mkdir()
            outside = Path(directory) / 'private.json'
            outside.write_text('{}')
            (s.workspace / 'link.json').symlink_to(outside)
            for name in ['../private.json', str(outside), 'link.json']:
                with self.assertRaises(Blocked):
                    s.artifact(name)


if __name__ == '__main__':
    unittest.main()
