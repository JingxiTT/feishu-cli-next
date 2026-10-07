# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Independent binary/artifact checks; optional signed clean-archive CLI probes."""
import base64
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scenarios import drive as scenario
from section_fixture import signed_headers
from scenario_fixture import ScenarioFixture
from codex_runner import cli_environment


class DriveScenarioTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.workspace = Path(temporary.name)
        self.backend = scenario.Scenario()
        self.backend.setup(self.workspace)
        self.requests = []
        self.answer = {"path": scenario.ARTIFACT, "sha256": scenario.SHA256}

    def dispatch(self, method, uri, body=None, identity="user"):
        raw = json.dumps(body).encode() if body is not None else b""
        reply = self.backend.dispatch(method, uri, raw, signed_headers(method, uri, raw, identity=identity))
        self.requests.append({"route": reply.route, "status": reply.status, "returned_ids": [], "proof": reply.proof})
        return reply

    def receipt(self):
        return {"scenario": scenario.SCENARIO, "state": self.backend.snapshot(),
                "requests": self.requests, "output_audit": {"checked": True, "passed": True}}

    def success(self):
        reply = self.dispatch("GET", scenario.DOWNLOAD)
        path = self.workspace / scenario.ARTIFACT
        path.parent.mkdir()
        path.write_bytes(reply.body)
        return self.receipt()

    def check(self, receipt=None, answer=None):
        return scenario.verify(self.workspace, self.answer if answer is None else answer,
                               self.receipt() if receipt is None else receipt, scenario.TASK["expected"])

    def test_public_contract_and_exact_binary_headers(self):
        self.assertEqual({"scenario", "answer_schema"}, set(scenario.TASK["fixture"]))
        self.assertNotIn(scenario.SHA256, json.dumps(scenario.TASK))
        reply = self.dispatch("GET", scenario.DOWNLOAD)
        self.assertIsNone(reply.json)
        self.assertEqual("application/octet-stream", reply.content_type)
        self.assertEqual(str(1048576), reply.headers["Content-Length"])
        self.assertIn('filename="drive-dev-01.bin"', reply.headers["Content-Disposition"])
        self.assertEqual(bytes(i % 251 for i in range(1048576)), reply.body)
        self.assertEqual(scenario.SHA256, hashlib.sha256(reply.body).hexdigest())

    def test_equivalent_read_order_identities_and_source_preservation(self):
        self.dispatch("POST", scenario.META, {"request_docs": [{"doc_token": scenario.TOKEN, "doc_type": "file"}], "with_url": True}, identity="bot")
        self.dispatch("GET", scenario.AUTH + "?type=file&action=export", identity="bot")
        self.success()
        self.assertTrue(self.check())
        self.assertEqual(self.backend.snapshot()["initial_source"], self.backend.snapshot()["final_source"])
        self.answer["path"] = str(self.workspace / scenario.ARTIFACT)
        self.answer["sha256"] = scenario.SHA256.upper()
        self.assertTrue(self.check())

    def test_foreign_selectors_mutations_and_ambiguous_values_rejected(self):
        bad = [("DELETE", scenario.DOWNLOAD, None), ("POST", scenario.DOWNLOAD, {}),
               ("GET", scenario.DOWNLOAD.replace(scenario.TOKEN, "foreign"), None),
               ("GET", scenario.DOWNLOAD + "?version=7", None),
               ("GET", scenario.AUTH + "?action=export&type=file&type=file", None),
               ("GET", scenario.AUTH + "?action=edit&type=file", None),
               ("POST", scenario.META, {"request_docs": [{"doc_token": "foreign", "doc_type": "file"}]}),
               ("POST", scenario.META, {"request_docs": [{"doc_token": scenario.TOKEN, "doc_type": "file"}], "with_url": 1})]
        for method, uri, body in bad:
            with self.subTest(uri=uri, body=body), self.assertRaises(ValueError):
                self.dispatch(method, uri, body)
        with self.assertRaises(ValueError):
            self.dispatch("GET", scenario.DOWNLOAD, identity="other")
        with self.assertRaises(ValueError):
            self.backend.dispatch("POST", scenario.META, b'{"request_docs":[],"request_docs":[]}', {"X-Lark-Proxy-Identity": "user"})
        self.assertEqual(0, self.backend.downloads)

    def test_claim_and_generated_file_without_download_observation_fail(self):
        self.assertFalse(self.check())
        path = self.workspace / scenario.ARTIFACT
        path.parent.mkdir(); path.write_bytes(scenario.PAYLOAD)
        self.assertFalse(self.check())

    def test_corrupt_truncated_and_json_artifacts_fail(self):
        self.success()
        path = self.workspace / scenario.ARTIFACT
        for contents in (scenario.PAYLOAD[:-1], scenario.PAYLOAD[:-1] + b"X", json.dumps({"ok": True, "data": self.answer}).encode()):
            path.write_bytes(contents)
            self.assertFalse(self.check())

    def test_missing_audit_altered_receipt_and_wrong_answer_fail(self):
        receipt = self.success()
        self.assertTrue(self.check(receipt))
        for key, replacement in (("requests", []), ("state", {}), ("output_audit", {}), ("scenario", "foreign")):
            changed = copy.deepcopy(receipt); changed[key] = replacement
            self.assertFalse(self.check(changed))
        for key, value in (("checked", 1), ("passed", 1), ("passed", False)):
            changed = copy.deepcopy(receipt); changed["output_audit"][key] = value
            self.assertFalse(self.check(changed))
        for change in ({"route": "drive_upload"}, {"status": True}, {"proof": {}}, {"status": 403}):
            changed = copy.deepcopy(receipt); changed["requests"][0].update(change)
            self.assertFalse(self.check(changed))
        for answer in ({}, {**self.answer, "sha256": "a" * 64}, {**self.answer, "path": "elsewhere.bin"}, {**self.answer, "extra": "ignored"}):
            self.assertFalse(self.check(receipt, answer))

    def test_backend_source_and_unrelated_file_changes_fail(self):
        self.success()
        self.backend.payload = self.backend.payload[:-1] + b"X"
        self.assertFalse(self.check())
        self.backend.payload = scenario.PAYLOAD
        self.backend.metadata["acl"] = "public"
        self.assertFalse(self.check())
        self.backend.metadata["acl"] = "self_only"
        (self.workspace / scenario.SENTINEL).write_text("changed")
        self.assertFalse(self.check())

    def test_symlink_and_outside_artifacts_fail(self):
        receipt = self.success()
        path = self.workspace / scenario.ARTIFACT
        original = self.workspace / "other.bin"
        path.rename(original); path.symlink_to(original)
        self.assertFalse(self.check(receipt))
        self.assertFalse(self.check(receipt, {**self.answer, "path": str(self.workspace.parent / "foreign.bin")}))

    def test_output_audit_decodes_protocol_json_and_detects_common_dumps(self):
        normal = {"type": "item.completed", "item": {"type": "command_execution", "aggregated_output": json.dumps(self.answer)}}
        self.assertTrue(scenario.audit_output(json.dumps(normal)))
        chunk = scenario.PAYLOAD[:512]
        for dump in (chunk.decode("latin1"), chunk.decode("utf-8", "replace"), chunk.hex(),
                     base64.b64encode(chunk).decode(), " ".join(f"{b:02x}" for b in chunk),
                     "\n".join(base64.b64encode(chunk).decode()[i:i+76] for i in range(0, len(base64.b64encode(chunk)), 76)),
                     repr(chunk), json.dumps(list(chunk))):
            event = {"type": "item.completed", "item": {"type": "command_execution", "aggregated_output": dump}}
            self.assertFalse(scenario.audit_output(json.dumps(event)), dump[:40])
        self.assertTrue(scenario.audit_output(json.dumps({"type": "agent_message", "text": "Saved " + scenario.ARTIFACT + " SHA-256 " + scenario.SHA256})))


class DriveArchivedCLITests(unittest.TestCase):
    def test_real_download_strategies(self):
        paths = ROOT / ".tmp/evaluation-paths.json"
        if not paths.is_file():
            self.skipTest("optional clean archived CLI paths unavailable")
        binaries = json.loads(paths.read_text())
        for arm in ("official_cli", "next_cli"):
            binary = binaries.get(arm)
            if not binary or not Path(binary).is_file():
                self.skipTest("optional clean archived CLI unavailable")
            for strategy in ("explicit_user", "explicit_bot", "url", "default", "inspect"):
                with self.subTest(arm=arm, strategy=strategy), tempfile.TemporaryDirectory() as temp:
                    workspace = Path(temp)
                    backend = scenario.Scenario(); backend.setup(workspace)
                    with ScenarioFixture(scenario.SCENARIO, backend) as server:
                        env = {**cli_environment(workspace), **server.environment()}
                        cwd = workspace
                        args = [binary, "drive", "+download"]
                        args += ["--url", "https://example.feishu.cn/file/" + scenario.TOKEN] if strategy == "url" else ["--file-token", scenario.TOKEN]
                        identity = "bot" if strategy == "explicit_bot" else "user"
                        if strategy == "default":
                            cwd = workspace / "downloads"; cwd.mkdir()
                        else:
                            args += ["--output", scenario.ARTIFACT]
                        args += ["--as", identity]
                        outputs = []
                        if strategy == "inspect":
                            inspected = subprocess.run([binary, "drive", "+inspect", "--url", "https://example.feishu.cn/file/" + scenario.TOKEN, "--as", identity], cwd=cwd, env=env, capture_output=True, text=True, timeout=20)
                            self.assertEqual(0, inspected.returncode, inspected.stderr)
                            outputs += [inspected.stdout, inspected.stderr]
                        result = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True, timeout=30)
                        self.assertEqual(0, result.returncode, result.stderr)
                        self.assertTrue(json.loads(result.stdout)["ok"])
                        outputs += [result.stdout, result.stderr]
                        receipt = server.receipt()
                        receipt["output_audit"] = {"checked": True, "passed": scenario.audit_output("\n".join(json.dumps({"aggregated_output": x}) for x in outputs))}
                        answer = {"path": scenario.ARTIFACT, "sha256": hashlib.sha256((workspace / scenario.ARTIFACT).read_bytes()).hexdigest()}
                        self.assertTrue(scenario.verify(workspace, answer, receipt, scenario.TASK["expected"]))


if __name__ == "__main__":
    unittest.main()
