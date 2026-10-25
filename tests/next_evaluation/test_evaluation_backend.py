# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""The current evaluator must not ship the retired fake Feishu backend."""
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[2]


class EvaluationBackendTests(unittest.TestCase):
    def test_fake_endpoint_and_legacy_launchers_are_removed(self):
        for name in ('section_fixture.py', 'scenario_fixture.py', 'read_fixture.py',
                     'scenario_protocol.py', 'run_read_batch.py', 'run_matrix.py',
                     'runtime', 'scenarios', 'paired', 'contracts'):
            with self.subTest(path=name):
                self.assertFalse((ROOT / 'evaluation' / name).exists())


if __name__ == '__main__':
    unittest.main()
