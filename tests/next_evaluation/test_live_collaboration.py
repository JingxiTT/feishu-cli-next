# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline regression checks; not live task-success evidence."""
import copy
import json
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
import live_collaboration
import live_workspace
from live_domains import collaboration as c
from live_domains.session import Blocked


class CollaborationTests(unittest.TestCase):
    def test_manifest_registers_only_the_five_permitted_document_cases(self):
        p = Path(__file__).resolve().parents[2] / 'evaluation/tasks.live-collaboration.json'
        d = json.loads(p.read_text())
        self.assertEqual({v['id'] for v in d['tasks']}, live_collaboration.MEDIA)
        self.assertEqual(d['backend'], 'live-feishu-v1')
        self.assertEqual(len(d['tasks']), 5)

    def test_unauthorized_cases_are_absent_and_rejected_before_tenant_access(self):
        directory = Path(__file__).resolve().parents[2] / 'evaluation'
        self.assertFalse((directory / 'live_domains/im_read.py').exists())
        self.assertFalse((directory / 'tasks.live-im.json').exists())
        registered = {task['id'] for path in directory.glob('tasks.live-*.json')
                      for task in json.loads(path.read_text())['tasks']}
        supported = live_collaboration.MEDIA | live_workspace.SINGLE.keys() | live_workspace.CROSS
        self.assertEqual(registered, supported | {'live-contact-current-account'})
        for identifier in ['collaboration-im-read', 'collaboration-group-read',
                           'business-file-handoff', 'business-task-create',
                           'business-calendar-create', 'business-mail-send']:
            self.assertNotIn(identifier, registered)
            request = {'backend': 'live-feishu-v1', 'phase': 'setup', 'task': {'id': identifier}}
            with patch.object(live_collaboration, 'Session') as collaboration_session, \
                 patch.object(live_workspace, 'session') as workspace_session:
                for driver in [live_collaboration.run, live_workspace.run]:
                    with self.assertRaisesRegex(Blocked, 'unsupported_live_task'):
                        driver(request, None, None, None)
                collaboration_session.assert_not_called()
                workspace_session.assert_not_called()

    def test_pngs_have_same_basename_but_distinct_dimensions_and_bytes(self):
        blobs = list(c.IMAGES.values())
        self.assertNotEqual(blobs[0], blobs[1])
        self.assertEqual([struct.unpack('>II', b[16:24]) for b in blobs], [(96, 64), (64, 96)])
        self.assertTrue(all(b.startswith(b'\x89PNG\r\n\x1a\n') for b in blobs))
        self.assertIn(b'\x00', c.FILES['archive/交接.bin'])

    def test_append_rejects_replacement_reorder_and_unrelated_native_style_change(self):
        before = [{'block_id': 'root', 'block_type': 1, 'children': ['guard']},
                  {'block_id': 'guard', 'block_type': 2, 'text': {'elements': []}}]
        after = copy.deepcopy(before)
        added = {'block_id': 'image', 'block_type': 27, 'image': {'token': 'test'}}
        after[0]['children'].append('image')
        after.append(added)
        self.assertEqual(c.appended(before, after, 27, 1), [added])
        after[0]['children'].reverse()
        self.assertIsNone(c.appended(before, after, 27, 1))
        after[0]['children'].reverse()
        after[1]['text']['style'] = {'bold': True}
        self.assertIsNone(c.appended(before, after, 27, 1))

    def test_board_requires_native_syntax_and_exact_source(self):
        for source, syntax in [(c.MERMAID, 2), (c.DRAWING, 3)]:
            nodes = [{'id': 'node', 'syntax': {'code': source, 'syntax_type': syntax}}]
            self.assertTrue(c.valid_board(nodes, source, syntax))
            self.assertFalse(c.valid_board(nodes, source + 'extra', syntax))
            self.assertFalse(c.valid_board(nodes, source, 99))
            self.assertFalse(c.valid_board(nodes * 2, source, syntax))
            self.assertFalse(c.valid_board([{'text': source}], source, syntax))

    def test_creation_cleanup_does_not_accept_an_unrelated_owned_document(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory) / 'answer.json'
            p.write_text(json.dumps({'document_id': 'a' * 20}))
            s = Mock(workspace=Path(directory), data={'tag': '[TEST] unique', 'creation_started_at': 200})
            s.artifact.return_value = p
            for meta in [{'title': '[TEST] other', 'create_time': '201'},
                         {'title': '[TEST] unique', 'create_time': '100'}]:
                s.api.return_value = {'metas': [meta]}
                with self.assertRaisesRegex(Blocked, 'created_document_not_run_owned'):
                    c.created_document(s)
                s.save.assert_not_called()

    def test_creation_journal_allows_cleanup_when_the_final_answer_is_wrong(self):
        with tempfile.TemporaryDirectory() as directory:
            p = Path(directory)
            (p / 'creation.json').write_text(json.dumps({'document_id': 'a' * 20}))
            (p / 'answer.json').write_text(json.dumps({'document_id': 'b' * 20}))
            s = Mock(workspace=p, data={'tag': '[TEST] unique', 'creation_started_at': 200, 'owned': []})
            s.artifact.side_effect = lambda name: p / name
            s.api.return_value = {'metas': [{'title': '[TEST] unique', 'create_time': '201'}]}
            self.assertEqual(c.created_document(s), 'a' * 20)
            self.assertEqual(s.data['owned'][0]['id'], 'a' * 20)

    def test_files_require_view_parentage_order_bytes_names_and_preservation(self):
        before = [{'block_id': 'root', 'block_type': 1, 'children': ['guard']},
                  {'block_id': 'guard', 'block_type': 2, 'text': {'elements': []}}]
        native = copy.deepcopy(before)
        native[0]['children'] += ['v1', 'v2']
        for i, name in enumerate(c.FILES, 1):
            native += [{'block_id': 'v' + str(i), 'block_type': 33, 'parent_id': 'root',
                        'children': ['f' + str(i)]},
                       {'block_id': 'f' + str(i), 'block_type': 23, 'parent_id': 'v' + str(i),
                        'file': {'name': name.split('/')[-1], 'token': 'token-' + str(i)}}]
        s = Mock(data={'document_id': 'doc', 'before': before},
                 request={'task': {'id': 'collaboration-file-upload'}})
        assets = dict(zip(['token-1', 'token-2'], c.FILES.values()))
        with patch.object(c.docs, 'blocks', return_value=native), \
             patch.object(c, 'download', side_effect=lambda _, token, suffix: assets[token]):
            self.assertTrue(c.verify(s, {'document_id': 'doc'}))
            assets['token-2'] += b'changed'
            self.assertFalse(c.verify(s, {'document_id': 'doc'}))
            assets['token-2'] = list(c.FILES.values())[1]
            native[3]['parent_id'] = 'root'
            self.assertFalse(c.verify(s, {'document_id': 'doc'}))

    def test_drawing_verifies_geometry_labels_vector_and_connector_endpoints(self):
        nodes = []
        for kind, x, y, w, h, fill, border in [
                ('ellipse', 30, 60, 80, 80, '#dbeafe', '#2563eb'),
                ('round_rect', 440, 65, 160, 70, '#dcfce7', '#16a34a')]:
            nodes.append({'type': 'composite_shape', 'angle': 0, 'x': x, 'y': y, 'width': w, 'height': h,
                          'composite_shape': {'type': kind}, 'style': {'fill_color': fill,
                          'border_color': border, 'border_radius': dict.fromkeys(
                              ['top_left', 'top_right', 'bottom_left', 'bottom_right'], 16)}})
        for label, x in [('Intake 001', 70), ('Approved?', 280), ('Ship 你好', 520)]:
            nodes.append({'type': 'text_shape', 'angle': 0, 'x': x - 20, 'y': 90, 'width': 40, 'height': 20,
                          'text': {'text': label, 'font_size': 16, 'text_color': '#000000'}})
        for start, end in [(110, 215), (345, 440)]:
            nodes.append({'type': 'connector', 'angle': 0, 'style': {'border_color': '#334155'},
                          'connector': {'start': {'position': {'x': start, 'y': 100}},
                                        'end': {'position': {'x': end, 'y': 100}}}})
        nodes.append({'type': 'image', 'angle': 0, 'image': {'token':
            '<svg xmlns="http://www.w3.org/2000/svg"><polygon points="280,45 345,100 280,155 215,100" '
            'fill="#fef3c7" stroke="#d97706"/></svg>'}})
        self.assertTrue(c.valid_drawing(nodes))
        self.assertFalse(c.valid_drawing(nodes * 2))
        for index, key, value in [(0, 'width', 79), (0, 'angle', 90), (2, 'x', 0)]:
            changed = copy.deepcopy(nodes)
            changed[index][key] = value
            self.assertFalse(c.valid_drawing(changed))
        changed = copy.deepcopy(nodes)
        changed[-2]['connector']['end']['position']['x'] = 500
        self.assertFalse(c.valid_drawing(changed))
        nodes[-1]['image']['token'] = '<svg>truncated'
        self.assertFalse(c.valid_drawing(nodes))


if __name__ == '__main__':
    unittest.main()
