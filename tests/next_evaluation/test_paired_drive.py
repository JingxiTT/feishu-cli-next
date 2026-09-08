"""Independent paired Drive byte/state/proof verification and clean CLI checks."""
import base64
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from paired import drive as scenario
from codex_runner import cli_environment
from scenario_fixture import ScenarioFixture
from section_fixture import signed_headers


def multipart(fields=None, payload=None):
    values = {"file_name": scenario.NAME, "parent_type": "explorer", "parent_node": scenario.PARENT,
              "size": str(scenario.SIZE), "file": scenario.PAYLOAD if payload is None else payload}
    if fields:
        values.update(fields)
    raw = b""
    for key, value in values.items():
        raw += b"--paired-boundary\r\nContent-Disposition: form-data; name=\"" + key.encode() + b'"'
        if key == "file":
            raw += b'; filename="source.bin"\r\nContent-Type: application/octet-stream'
        raw += b"\r\n\r\n" + (value if isinstance(value, bytes) else str(value).encode()) + b"\r\n"
    return raw + b"--paired-boundary--\r\n", "multipart/form-data; boundary=paired-boundary"


class PairedDriveTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory(); self.addCleanup(temp.cleanup)
        self.workspace = Path(temp.name)
        self.backend = scenario.Scenario(); self.backend.setup(self.workspace)
        self.requests = []
        self.answer = {"file_token": scenario.TOKEN, "path": scenario.ARTIFACT, "sha256": scenario.SHA256}

    def dispatch(self, method, uri, body=None, raw=None, content_type=None, identity="user"):
        raw = raw if raw is not None else json.dumps(body).encode() if body is not None else b""
        headers = signed_headers(method, uri, raw, identity=identity)
        if content_type:
            headers["Content-Type"] = content_type
        reply = self.backend.dispatch(method, uri, raw, headers)
        self.requests.append({"route": reply.route, "status": reply.status, "proof": reply.proof, "signature_valid": True})
        return reply

    def upload(self, **kwargs):
        raw, content_type = multipart(**kwargs)
        return self.dispatch("POST", scenario.UPLOAD, raw=raw, content_type=content_type)

    def receipt(self):
        return {"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": self.requests,
                "output_audit": {"checked": True, "passed": True}}

    def check(self, receipt=None, answer=None):
        return scenario.verify(self.workspace, self.answer if answer is None else answer,
                               self.receipt() if receipt is None else receipt, {})

    def roundtrip(self):
        self.upload()
        reply = self.dispatch("GET", scenario.DOWNLOAD)
        path = self.workspace / scenario.ARTIFACT; path.parent.mkdir(); path.write_bytes(reply.body)
        return self.receipt()

    def test_public_task_and_real_binary_roundtrip(self):
        self.assertEqual("paired-drive", scenario.TASK["id"])
        self.assertEqual("drive", scenario.TASK["section"])
        self.assertEqual("paired-drive", scenario.TASK["fixture"]["scenario_section"])
        self.assertNotIn(scenario.SHA256, json.dumps(scenario.TASK))
        self.assertNotIn(scenario.TOKEN, json.dumps(scenario.TASK))
        self.roundtrip()
        self.assertTrue(self.check())
        self.assertEqual(hashlib.sha256(bytes(i % 251 for i in range(4096))).hexdigest(), self.answer["sha256"])
        self.answer["path"] = str(self.workspace / scenario.ARTIFACT)
        self.answer["sha256"] = scenario.SHA256.upper()
        self.assertTrue(self.check())

    def test_optional_equivalent_read_routes(self):
        self.dispatch("GET", "/open-apis/authen/v1/user_info", identity="bot")
        self.dispatch("GET", "/open-apis/drive/v1/files?folder_token=" + scenario.PARENT + "&page_size=200")
        self.dispatch("POST", scenario.META, {"request_docs": [{"doc_token": scenario.PARENT, "doc_type": "folder"}]})
        self.roundtrip()
        for token, kind in ((scenario.PARENT, "folder"), (scenario.TOKEN, "file")):
            for suffix in ("/members", "/public"):
                self.dispatch("GET", "/open-apis/drive/v1/permissions/" + token + suffix + "?type=" + kind)
        self.dispatch("GET", "/open-apis/drive/v1/permissions/" + scenario.TOKEN + "/members/auth?type=file&action=export")
        self.assertTrue(self.check())

    def test_exact_bot_self_grant_effect_and_foreign_grant_rejected(self):
        raw, ct = multipart()
        self.dispatch("POST", scenario.UPLOAD, raw=raw, content_type=ct, identity="bot")
        path = "/open-apis/drive/v1/permissions/" + scenario.TOKEN + "/members?type=file&need_notification=false"
        body = {"member_type": "openid", "member_id": scenario.SELF, "perm": "full_access", "type": "user"}
        with self.assertRaises(ValueError):
            self.dispatch("POST", path, {**body, "member_id": "foreign"}, identity="bot")
        self.dispatch("POST", path, body, identity="bot")
        with self.assertRaises(ValueError):
            self.dispatch("POST", path, body, identity="bot")
        self.assertEqual([{"token": scenario.TOKEN, "identity": "bot", "recipient": scenario.SELF, "perm": "full_access", "notification": False}], self.backend.snapshot()["grants"])
        reply = self.dispatch("GET", scenario.DOWNLOAD, identity="bot")
        path = self.workspace / scenario.ARTIFACT; path.parent.mkdir(); path.write_bytes(reply.body)
        self.assertTrue(self.check())
        changed = self.receipt(); changed["state"]["grants"][0]["notification"] = 0
        self.assertFalse(self.check(changed))
        changed = copy.deepcopy(self.receipt()); changed["requests"][1]["proof"]["recipient"] = "foreign"
        self.assertFalse(self.check(changed))

    def test_folder_page_size_and_continuation(self):
        self.upload()
        path = "/open-apis/drive/v1/files?folder_token=" + scenario.PARENT + "&page_size=1"
        first = self.dispatch("GET", path).json["data"]
        self.assertEqual(1, len(first["files"]))
        self.assertTrue(first["has_more"])
        second = self.dispatch("GET", path + "&page_token=" + first["next_page_token"]).json["data"]
        self.assertEqual(1, len(second["files"]))
        self.assertFalse(second["has_more"])
        self.assertNotEqual(first["files"][0]["token"], second["files"][0]["token"])
        with self.assertRaises(ValueError):
            self.dispatch("GET", path + "&page_token=foreign")

    def test_wrong_upload_duplicate_creation_and_early_download_rejected(self):
        with self.assertRaises(ValueError): self.dispatch("GET", scenario.DOWNLOAD)
        for fields in ({"parent_node": "foreign"}, {"file_name": "wrong.bin"}, {"size": "4095"},
                       {"parent_type": "wiki"}, {"file_token": "existing"}, {"checksum": "incorrect"}):
            with self.subTest(fields=fields), self.assertRaises(ValueError): self.upload(fields=fields)
        with self.assertRaises(ValueError): self.upload(payload=scenario.PAYLOAD[:-1] + b"X")
        self.upload()
        with self.assertRaises(ValueError): self.upload()
        self.assertEqual(1, self.backend.uploads)

    def test_foreign_write_selector_duplicate_and_malformed_fields_rejected(self):
        for method, uri, body in (("DELETE", "/open-apis/drive/v1/files/" + scenario.UNRELATED["token"], None),
                                  ("PATCH", "/open-apis/drive/v1/permissions/" + scenario.PARENT + "/public", {"link_share_entity": "anyone_readable"}),
                                  ("GET", "/open-apis/drive/v1/files?folder_token=" + scenario.PARENT + "&folder_token=foreign", None),
                                  ("POST", scenario.META, {"request_docs": [{"doc_token": "foreign", "doc_type": "file"}]})):
            with self.subTest(uri=uri), self.assertRaises(ValueError): self.dispatch(method, uri, body)
        with self.assertRaises(ValueError): self.dispatch("POST", scenario.META, raw=b'{"request_docs":[],"request_docs":[]}')
        raw, ct = multipart()
        raw = raw.replace(b"--paired-boundary--\r\n", b'--paired-boundary\r\nContent-Disposition: form-data; name="size"\r\n\r\n4096\r\n--paired-boundary--\r\n')
        with self.assertRaises(ValueError): self.dispatch("POST", scenario.UPLOAD, raw=raw, content_type=ct)

    def test_claim_or_copied_source_without_signed_roundtrip_fails(self):
        path = self.workspace / scenario.ARTIFACT; path.parent.mkdir(); path.write_bytes(scenario.PAYLOAD)
        self.assertFalse(self.check())
        self.upload()
        self.assertFalse(self.check())
        self.dispatch("GET", scenario.DOWNLOAD)
        self.assertTrue(self.check())
        receipt = self.receipt(); receipt["requests"][0]["signature_valid"] = False
        self.assertFalse(self.check(receipt))

    def test_corrupt_bytes_symlinks_and_local_preservation(self):
        self.roundtrip(); path = self.workspace / scenario.ARTIFACT
        for raw in (scenario.PAYLOAD[:-1], scenario.PAYLOAD[:-1] + b"X", b'{"ok":true}'):
            path.write_bytes(raw); self.assertFalse(self.check())
        path.write_bytes(scenario.PAYLOAD)
        for relative in (scenario.SOURCE, scenario.SENTINEL):
            item = self.workspace / relative; before = item.read_bytes()
            item.write_bytes(b"changed"); self.assertFalse(self.check()); item.write_bytes(before)
        path.unlink(); path.symlink_to(self.workspace / scenario.SOURCE)
        self.assertFalse(self.check())

    def test_backend_and_signed_proof_tampering_fails(self):
        receipt = self.roundtrip()
        for key in ("initial_parent", "final_parent", "initial_unrelated", "final_unrelated", "uploaded", "payload"):
            changed = copy.deepcopy(receipt); changed["state"][key] = {}; self.assertFalse(self.check(changed))
        for key, value in (("uploads", True), ("uploads", 2), ("downloads", 0), ("reports", True), ("source_unchanged", False)):
            changed = copy.deepcopy(receipt); changed["state"][key] = value; self.assertFalse(self.check(changed))
        for key, value in (("proof", {}), ("status", True), ("status", 403), ("signature_valid", 1), ("route", "foreign")):
            changed = copy.deepcopy(receipt); changed["requests"][0][key] = value; self.assertFalse(self.check(changed))
        changed = copy.deepcopy(receipt); changed["requests"].reverse(); self.assertFalse(self.check(changed))
        for answer in ({}, {**self.answer, "file_token": "foreign"}, {**self.answer, "path": "elsewhere"}, {**self.answer, "sha256": "wrong"}):
            self.assertFalse(self.check(answer=answer))
        self.backend.payload = b"changed"; self.assertFalse(self.check())

    def test_no_dump_audit_required_and_common_encoding_detection(self):
        receipt = self.roundtrip()
        for value in ({}, {"checked": True, "passed": False}, {"checked": 1, "passed": True}):
            changed = copy.deepcopy(receipt); changed["output_audit"] = value; self.assertFalse(self.check(changed))
        self.assertTrue(scenario.audit_output(json.dumps({"aggregated_output": json.dumps(self.answer)})))
        for value in (scenario.PAYLOAD[:512].hex(), base64.b64encode(scenario.PAYLOAD[:512]).decode(), json.dumps(list(scenario.PAYLOAD[:512]))):
            self.assertFalse(scenario.audit_output(json.dumps({"aggregated_output": value})))


class PairedDriveArchivedCLITests(unittest.TestCase):
    def test_both_archived_cli_strategies(self):
        directory = ROOT / ".tmp/luna-medium-preparation"
        binaries = [directory / "official-cli", directory / "next-cli"]
        if os.environ.get("EVAL_PAIRED_DRIVE_BINARIES"):
            binaries = [Path(path) for path in json.loads(os.environ["EVAL_PAIRED_DRIVE_BINARIES"])]
        if not all(path.is_file() for path in binaries):
            self.skipTest("fresh controlled paired binaries unavailable")
        for binary in binaries:
            for strategy in ("shortcut_user", "shortcut_bot", "raw", "generated_reads"):
                with self.subTest(binary=binary.name, strategy=strategy), tempfile.TemporaryDirectory() as temp:
                    workspace = Path(temp); backend = scenario.Scenario(); backend.setup(workspace)
                    outputs = []
                    with ScenarioFixture(scenario.SCENARIO, backend) as server:
                        env = {**cli_environment(workspace), **server.environment()}
                        identity = "bot" if strategy == "shortcut_bot" else "user"
                        def run(args):
                            result = subprocess.run([str(binary), *args, "--as", identity], cwd=workspace, env=env, capture_output=True, text=True, timeout=30)
                            self.assertEqual(0, result.returncode, result.stderr)
                            outputs.extend([result.stdout, result.stderr])
                            return json.loads(result.stdout)
                        if strategy == "generated_reads":
                            run(["drive", "files", "list", "--folder-token", scenario.PARENT, "--page-size", "200"])
                            run(["drive", "metas", "batch_query", "--data", json.dumps({"request_docs": [{"doc_token": scenario.PARENT, "doc_type": "folder"}]})])
                        if strategy == "raw":
                            uploaded = run(["api", "POST", scenario.UPLOAD, "--file", scenario.SOURCE, "--data", json.dumps({"file_name": scenario.NAME, "parent_type": "explorer", "parent_node": scenario.PARENT, "size": scenario.SIZE})])
                        else:
                            uploaded = run(["drive", "+upload", "--file", scenario.SOURCE, "--folder-token", scenario.PARENT])
                        self.assertEqual(scenario.TOKEN, uploaded["data"]["file_token"])
                        (workspace / "downloads").mkdir()
                        if strategy == "raw":
                            run(["api", "GET", scenario.DOWNLOAD, "--output", scenario.ARTIFACT])
                        else:
                            run(["drive", "+download", "--file-token", uploaded["data"]["file_token"], "--output", scenario.ARTIFACT])
                        receipt = server.receipt()
                        receipt["output_audit"] = {"checked": True, "passed": scenario.audit_output("\n".join(json.dumps({"aggregated_output": value}) for value in outputs))}
                        answer = {"file_token": uploaded["data"]["file_token"], "path": scenario.ARTIFACT, "sha256": hashlib.sha256((workspace / scenario.ARTIFACT).read_bytes()).hexdigest()}
                        self.assertTrue(scenario.verify(workspace, answer, receipt, {}), receipt)


if __name__ == "__main__":
    unittest.main()
