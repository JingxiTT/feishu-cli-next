# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('next_layout', Path(__file__).with_name('next_layout.py'))
layout = importlib.util.module_from_spec(spec)
spec.loader.exec_module(layout)


class LayoutTest(unittest.TestCase):
    def tree(self, root, files):
        for name, content in files.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(content)
        return sorted(files)

    def sealed(self, root, names):
        layout.seal(root, names)
        return names + [layout.MANIFEST]

    def test_repository_layout_holds(self):
        self.assertEqual(layout.check(layout.ROOT), [])

    def test_repository_keeps_records_and_baselines_out_of_docs(self):
        names = layout.tracked(layout.ROOT)
        self.assertFalse([n for n in names if n.startswith(('docs/reference/', 'evaluation/evidence/'))])
        for required in ('baseline/v1.0.97/next-interface-mapping.json.gz', 'baseline/sources.json',
                         'evidence/reference/p3-review-assignments.json', layout.MANIFEST,
                         layout.UPSTREAM_SOURCES, layout.FOOTPRINT):
            self.assertIn(required, names)
        self.assertGreater(len(layout.sealed_files(names)), 400)

    def test_new_root_entry_and_non_markdown_docs_are_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = self.sealed(root, self.tree(root, {'README.md': b'x', 'docs/a.md': b'x'}))
            self.assertEqual(layout.check(root, names), [])
            names += self.tree(root, {'notes/plan.md': b'x', 'docs/sections/run.json': b'{}'})
            problems = layout.check(root, names)
            self.assertEqual(len(problems), 2)
            self.assertTrue(problems[0].startswith('notes: new root entry'))
            self.assertTrue(problems[1].startswith('docs/sections/run.json: docs/ holds markdown only'))

    def test_sealed_evidence_cannot_change_move_or_stay_unsealed(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = self.tree(root, {'evidence/run/a.json': b'{"ok":true}\n', 'evidence/README.md': b'index'})
            self.assertEqual(layout.check(root, names), [layout.MANIFEST + ': missing',
                                                         'evidence/run/a.json: not sealed; run python3 scripts/next_layout.py seal'])
            names = self.sealed(root, names)
            self.assertEqual(layout.check(root, names), [])
            self.assertEqual((root / layout.MANIFEST).read_text(),
                             hashlib.sha256(b'{"ok":true}\n').hexdigest() + '  evidence/run/a.json\n')
            # The index may change; a record may not.
            (root / 'evidence/README.md').write_bytes(b'updated index')
            self.assertEqual(layout.check(root, names), [])
            (root / 'evidence/run/a.json').write_bytes(b'{"ok":false}\n')
            self.assertEqual(layout.check(root, names), ['evidence/run/a.json: sealed evidence changed; add a new file instead of editing a record'])
            # Sealing again must not launder the edit.
            self.assertEqual(layout.seal(root, names), [])
            self.assertEqual(len(layout.check(root, names)), 1)
            names.remove('evidence/run/a.json')
            self.assertEqual(layout.check(root, names), ['evidence/run/a.json: sealed evidence removed or renamed'])

    def test_seal_appends_new_records_only(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = self.sealed(root, self.tree(root, {'evidence/run/a.json': b'a'}))
            before = (root / layout.MANIFEST).read_text()
            names += self.tree(root, {'evidence/run/b.json': b'b'})
            self.assertEqual(layout.seal(root, names), ['evidence/run/b.json'])
            after = (root / layout.MANIFEST).read_text()
            self.assertTrue(after.startswith(before))
            self.assertEqual(after.count('\n'), 2)
            self.assertEqual(layout.check(root, names), [])

    def test_footprint_lists_exactly_the_edited_upstream_files(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            pinned = {'cmd/a.go': b'package cmd\n', 'cmd/b.go': b'package cmd\n', 'main.go': b'package main\n'}
            names = self.sealed(root, self.tree(root, pinned))
            sources = {'files': {name: hashlib.sha256(content).hexdigest() for name, content in pinned.items()}}
            names += self.tree(root, {layout.UPSTREAM_SOURCES: json.dumps(sources).encode()})
            self.assertEqual(layout.check(root, names), [layout.FOOTPRINT + ': out of date; run python3 scripts/next_layout.py footprint'])
            (root / 'docs').mkdir()
            write = lambda: (root / layout.FOOTPRINT).write_text(layout.render_footprint(*layout.footprint(root)))
            write()
            names.append(layout.FOOTPRINT)
            self.assertEqual(layout.check(root, names), [])
            self.assertIn('0 of 3 pinned upstream files.', (root / layout.FOOTPRINT).read_text())
            # An edit to an upstream file must show up in the committed list.
            (root / 'cmd/b.go').write_bytes(b'package cmd\n\nvar fork = true\n')
            (root / 'main.go').unlink()
            self.assertEqual(layout.check(root, names), [layout.FOOTPRINT + ': out of date; run python3 scripts/next_layout.py footprint (newly edited upstream files: cmd/b.go, main.go)'])
            write()
            text = (root / layout.FOOTPRINT).read_text()
            self.assertIn('2 of 3 pinned upstream files.', text)
            self.assertIn('## (root) (1)\n\n- `main.go`\n', text)
            self.assertIn('## cmd (1)\n\n- `cmd/b.go`\n', text)
            self.assertEqual(layout.check(root, names), [])
            # Restoring upstream bytes is reported too, so the list only ever matches reality.
            (root / 'cmd/b.go').write_bytes(pinned['cmd/b.go'])
            self.assertEqual(layout.check(root, names), [layout.FOOTPRINT + ': out of date; run python3 scripts/next_layout.py footprint (restored to upstream: cmd/b.go)'])

    def test_footprint_names_upstream_paths_without_pinned_hashes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            pinned = {'cmd/a.go': b'package cmd\n'}
            self.tree(root, dict(pinned, **{'skills/lark-im/SKILL.md': b'x', 'Makefile': b'x', 'docs/a.md': b'x'}))
            sources = {'files': {name: hashlib.sha256(content).hexdigest() for name, content in pinned.items()}}
            self.tree(root, {layout.UPSTREAM_SOURCES: json.dumps(sources).encode()})
            edited, count, uncovered = layout.footprint(root)
            self.assertEqual(uncovered, ['Makefile', 'skills/'])
            text = layout.render_footprint(edited, count, uncovered)
            self.assertIn('## Not covered\n', text)
            self.assertTrue(text.endswith('- `Makefile`\n- `skills/`\n'))
            # The uncovered list is not mistaken for edited files when the footprint drifts.
            (root / 'cmd/a.go').write_bytes(b'package cmd\n\nvar fork = true\n')
            (root / layout.FOOTPRINT).write_text(text)
            self.assertEqual(layout.footprint_problems(root), [layout.FOOTPRINT + ': out of date; run python3 scripts/next_layout.py footprint (newly edited upstream files: cmd/a.go)'])

    def test_fork_go_files_in_upstream_packages_carry_next_suffix(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            pinned = {'cmd/root.go': b'package cmd\n'}
            sources = {'files': {name: hashlib.sha256(content).hexdigest() for name, content in pinned.items()}}
            self.tree(root, dict(pinned, **{layout.UPSTREAM_SOURCES: json.dumps(sources).encode()}))
            names = ['cmd/root.go', 'cmd/agent.go', 'cmd/agent_test.go', 'cmd/hook_next.go',
                     'cmd/hook_next_test.go', 'cmd/receipts/receipts.go', 'main_next_helper.go']
            self.assertEqual(layout.unmarked_fork_files(root, names), [
                'cmd/agent.go: fork file in upstream package; name it <name>_next.go or <name>_next_test.go',
                'cmd/agent_test.go: fork file in upstream package; name it <name>_next.go or <name>_next_test.go'])

    def test_malformed_or_duplicate_manifest_lines_fail(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            names = self.sealed(root, self.tree(root, {'evidence/run/a.json': b'a'}))
            line = (root / layout.MANIFEST).read_text()
            (root / layout.MANIFEST).write_text(line + line + 'not-a-hash  evidence/run/c.json\n')
            self.assertEqual(layout.check(root, names), [layout.MANIFEST + ':2: malformed or duplicate entry',
                                                         layout.MANIFEST + ':3: malformed or duplicate entry'])
            with self.assertRaises(ValueError):
                layout.seal(root, names)


if __name__ == '__main__':
    unittest.main()
