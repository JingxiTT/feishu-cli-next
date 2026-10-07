# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

import importlib.util
from pathlib import Path
import subprocess
import unittest


ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("measure_cli", ROOT / "evaluation/measure_cli.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class MeasureCLIContractTests(unittest.TestCase):
    def test_request_comparison_retains_safety_and_guidance(self):
        full = b'{"name":"read","inputSchema":{},"outputSchema":{},"_meta":{"risk":"read","scopes":["read"],"affordance":{"tips":["keep"]}}}'
        inputs = full.replace(b',"outputSchema":{}', b'')
        self.assertEqual(MODULE.request_contract(full), MODULE.request_contract(inputs))
        for old, new in ((b'"read"]', b'"write"]'), (b'"keep"', b'"drop"')):
            self.assertNotEqual(MODULE.request_contract(full), MODULE.request_contract(inputs.replace(old, new)))
        self.assertEqual(MODULE.request_contract(b'[' + full + b']'), MODULE.request_contract(inputs))

    def test_failures_never_become_fast_success_measurements(self):
        result = subprocess.CompletedProcess([], 2, b'', b'private diagnostic')
        with self.assertRaisesRegex(ValueError, "comparison incomplete") as error:
            MODULE.require_success(result, "reference", ["schema", "unknown"])
        self.assertNotIn("private diagnostic", str(error.exception))


if __name__ == "__main__":
    unittest.main()
