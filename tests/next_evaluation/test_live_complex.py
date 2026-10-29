# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Account-free regressions for business invariants."""
import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
from live_domains import base, complex, docs, sheets


class ComplexTests(unittest.TestCase):
    def test_journal_deduplicates_versions_before_void_approval_and_decimal_rounding(self):
        before = {'raw': [[{'value': value} for value in row] for row in sheets.VALUES]}
        changes, report = complex.journal_result(before)
        self.assertEqual(report['duplicate_revisions'], ['E001:1', 'E004:1', 'E006:2', 'E008:1'])
        self.assertIn('E009', report['voided_event_ids'])
        self.assertNotIn('E009', report['applied_event_ids'])
        self.assertNotIn('E000', report['voided_event_ids'])
        self.assertIn('E000', report['applied_event_ids'])
        self.assertTrue(all(row in [1, 3] and col in [1, 3] for row, col in changes))
        self.assertEqual(report['closing_amounts']['002'], 0)
        self.assertEqual(report['closing_amounts']['004'], 5)
        # 2 - 3*18.12 - 30.81; each USD event rounds before summation.
        self.assertEqual(report['closing_amounts']['001'], -83.17)
        self.assertEqual(report['closing_amounts']['003'], 2.75)

    def test_base_conflict_precedence_latest_revision_zero_null_and_late_page(self):
        before = {f'id{i}': copy.deepcopy(row) for i, row in enumerate(base.ROWS)}
        after, report = complex.adjustment_result(before)
        self.assertIn('R000', report['conflict'])
        self.assertIn('R033', report['conflict'])  # Conflict wins over unapproved.
        self.assertIn('R003', report['unapproved'])
        self.assertEqual(report['missing'], ['R999'])
        self.assertEqual(after['id6']['Amount'], 1.25)  # Latest, rather than revision 1's -6.
        self.assertEqual(after['id24']['Amount'], 0)
        self.assertIsNone(after['id24']['Note'])
        self.assertIn('id204', report['changed_record_ids'])
        self.assertEqual(before['id24']['Note'], base.ROWS[24]['Note'])
        self.assertEqual(after['id33'], before['id33'])
        self.assertEqual(set(after), set(before))

    def test_doc_closeout_requires_all_pages_and_preserves_ids_and_unrelated_rich_runs(self):
        native = [{'block_id': str(i), 'block_type': 2, 'text': {'elements': [
            {'text_run': {'content': line, 'text_element_style': {'bold': False}}}]}}
                  for i, line in enumerate(docs.audit_lines())]
        expected, report, line = complex.closeout_result(native)
        self.assertIn('519', report['block_ids'])
        self.assertEqual(report['record_count'], 107)
        self.assertEqual(report['total_amount'], 1006)
        self.assertNotIn('27', report['block_ids'])  # Approved R42 invoice with zero amount.
        self.assertNotIn('3', report['block_ids'])  # Approved R42 credit.
        self.assertEqual(expected[27], native[27])
        self.assertIn('SETTLED', docs.text(expected[519]))
        self.assertEqual(expected[0], native[0])
        self.assertEqual(expected[519]['block_id'], '519')
        self.assertEqual(expected[519]['text']['elements'][0]['text_run']['text_element_style'], {'bold': False})
        self.assertIn(str(report['total_amount']), line)
        self.assertIn('APPROVED', docs.text(native[519]))

    def test_sheet_verifier_rejects_correct_report_with_wrong_type_formula_or_guard(self):
        before = {mode: [[{'value': value} for value in row] for row in sheets.VALUES]
                  for mode in ['raw', 'formula']}
        for row in range(1, 5):
            before['formula'][row][4]['formula'] = f'=B{row+1}*2'
        changes, report = complex.journal_result(before)
        after = sheets.expected_snapshot(before, changes)
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'report.json'
            artifact.write_text(json.dumps(report))
            s = Mock(data={'before': before, 'spreadsheet_token': 'sheet'})
            s.artifact.return_value = artifact
            with patch.object(sheets, 'snapshot', return_value=after):
                self.assertTrue(complex.verify(s, {'path': 'report.json'}, 'sheets'))
                for row, col, value in [(1, 1, '-83.17'), (5, 3, 'changed'), (2, 2, 0)]:
                    old = after['raw'][row][col]['value']
                    after['raw'][row][col]['value'] = value
                    self.assertFalse(complex.verify(s, {'path': 'report.json'}, 'sheets'))
                    after['raw'][row][col]['value'] = old
                after['formula'][1][4]['formula'] = '=-166.34'
                self.assertFalse(complex.verify(s, {'path': 'report.json'}, 'sheets'))

    def test_base_verifier_rejects_nonselected_edit_and_false_report_even_when_totals_match(self):
        before = {f'id{i}': copy.deepcopy(row) for i, row in enumerate(base.ROWS)}
        after, report = complex.adjustment_result(before)
        with tempfile.TemporaryDirectory() as directory:
            artifact = Path(directory) / 'report.json'
            artifact.write_text(json.dumps(report))
            s = Mock(data={'before': before, 'base_token': 'base'})
            s.artifact.return_value = artifact
            with patch.object(base, 'records', return_value=after):
                self.assertTrue(complex.verify(s, {'path': 'report.json'}, 'base'))
                after['id33']['Note'] = 'unauthorized change'
                self.assertFalse(complex.verify(s, {'path': 'report.json'}, 'base'))
                after['id33'] = before['id33']
                artifact.write_text(json.dumps({**report, 'closing_total': str(report['closing_total'])}))
                self.assertFalse(complex.verify(s, {'path': 'report.json'}, 'base'))

    def test_closeout_accepts_identical_adjacent_runs_but_preserves_every_style_link_id_and_guard(self):
        text = 'R42 | APPROVED | INV-00519 | Amount=10 | résumé 你好 😀'
        before = [{'block_id': 'root', 'block_type': 1, 'children': ['invoice', 'guard']},
                  {'block_id': 'invoice', 'block_type': 2, 'text': {'elements': [
                      {'text_run': {'content': text, 'text_element_style': {'bold': False}}}]}},
                  {'block_id': 'guard', 'block_type': 2, 'text': {'elements': [
                      {'text_run': {'content': 'Keep linked status', 'text_element_style': {
                          'bold': True, 'link': {'url': docs.LINK}}}}]}}]
        after, _, line = complex.closeout_result(before)
        run = after[1]['text']['elements'][0]['text_run']
        run['content'], tail = run['content'][:10], run['content'][10:]
        after[1]['text']['elements'].append({'text_run': {**copy.deepcopy(run), 'content': tail}})
        after[0]['children'].append('summary')
        after.append({'block_id': 'summary', 'block_type': 2, 'text': {'elements': [
            {'text_run': {'content': line}}]}})
        self.assertTrue(complex.closeout_preserved(before, after))
        for mode in ['style', 'link', 'id', 'guard', 'duplicate-summary', 'root-order']:
            wrong = copy.deepcopy(after)
            if mode == 'style': wrong[1]['text']['elements'][1]['text_run']['text_element_style']['bold'] = True
            if mode == 'link': wrong[1]['text']['elements'][1]['text_run']['text_element_style']['link'] = {'url': docs.LINK}
            if mode == 'id': wrong[1]['block_id'] = 'replacement'
            if mode == 'guard': wrong[2]['text']['elements'][0]['text_run']['content'] = 'changed'
            if mode == 'duplicate-summary': wrong.append(copy.deepcopy(wrong[-1]))
            if mode == 'root-order': wrong[0]['children'].reverse()
            self.assertFalse(complex.closeout_preserved(before, wrong), mode)


if __name__ == '__main__':
    unittest.main()
