# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Replay local Docs workflows through a real CLI and the unchanged fidelity gates."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / 'evaluation'))
from paired import docs_edit, docs_read
from scenarios import docs as native


@unittest.skipUnless(os.environ.get('EVAL_DOCS_WORKSPACE_BINARY'), 'requires the candidate native CLI')
class DocsWorkspaceBinaryTests(unittest.TestCase):
    def test_local_workflows_preserve_existing_native_verification(self):
        from codex_runner import cli_environment
        from scenario_fixture import ScenarioFixture

        for kind in ('read', 'edit', 'noop', 'unsupported'):
            module = docs_read if kind == 'read' else docs_edit
            scenario = module.Scenario()
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as directory, ScenarioFixture(module.TASK['fixture']['scenario'], scenario) as server:
                work = Path(directory)
                for name in ('home', 'tmp', 'feishu-config'):
                    (work / name).mkdir()
                env = {**cli_environment(work), **server.environment()}

                def call(*args, success=True):
                    result = subprocess.run([os.environ['EVAL_DOCS_WORKSPACE_BINARY'], *args, '--as', 'user'],
                                            cwd=work, env=env, capture_output=True, text=True, timeout=30)
                    if success:
                        self.assertEqual(result.returncode, 0, result.stderr)
                        return json.loads(result.stdout)['data']
                    self.assertNotEqual(result.returncode, 0)

                args = ['docs', '+pull', '--doc', native.DOCUMENT]
                if kind == 'read':
                    data = call(*args)
                    # CommonMark backslash escapes affect presentation, not the fact.
                    self.assertIn(docs_read.FACT, re.sub(r'\\([!"#$%&\'()*+,\-./:;<=>?@\[\]\\^_`{|}~])', r'\1', data['markdown']))
                    self.assertIn(docs_read.BLOCK, data['markdown'])
                    self.assertTrue(docs_read.verify(work, docs_read.ANSWER, server.receipt(), {}))
                    continue
                call(*args, '--output', './note.md')
                local = work / 'note.md'
                initial = local.read_text()
                if kind == 'edit':
                    self.assertEqual(initial.count('Old '), 1)
                    local.write_text(initial.replace('Old ', 'New ', 1))
                    result = call('docs', '+push', '--file', './note.md')
                    self.assertTrue(result['verified'])
                    changed = result['changed_blocks']
                    self.assertEqual(len(changed), 1)
                    self.assertEqual(changed[0]['block_id'], native.TARGET)
                    answer = {'document_id': result['document_id'], 'paragraph_text': changed[0]['text']}
                    self.assertTrue(docs_edit.verify(work, answer, server.receipt(), {}))
                elif kind == 'noop':
                    result = call('docs', '+push', '--file', './note.md')
                    self.assertTrue(result['verified'])
                    self.assertTrue(result['unchanged'])
                    self.assertEqual(result['changed_blocks'], [])
                    self.assertEqual(scenario.state, native.INITIAL)
                    self.assertEqual(scenario.mutations, [])
                else:
                    local.write_text(initial.replace('<!-- block:', '<!-- changed:', 1))
                    call('docs', '+push', '--file', './note.md', success=False)
                    self.assertEqual(scenario.state, native.INITIAL)
                    self.assertEqual(scenario.mutations, [])


if __name__ == '__main__':
    unittest.main()
