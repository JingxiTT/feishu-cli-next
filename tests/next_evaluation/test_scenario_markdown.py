# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""No live access/model calls; direct synthetic transitions and hostile artifacts."""
import copy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

EVALUATION = Path(__file__).resolve().parents[2] / "evaluation"
sys.path.insert(0, str(EVALUATION))
from scenarios import markdown as scenario
from section_fixture import signed_headers


class MarkdownScenarioTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.workspace = Path(self.temporary.name)
        self.backend = scenario.Scenario()
        self.backend.setup(self.workspace)
        self.requests = []
        self.answer = {"document_id": scenario.DOCUMENT_ID,
                       "artifact_path": "converted.xml", "unsupported": []}

    def dispatch(self, method, path, body=None, identity="user"):
        raw = json.dumps(body or {}, ensure_ascii=False).encode()
        reply = self.backend.dispatch(method, path, raw, signed_headers(method, path, raw, identity=identity))
        self.requests.append({"route": reply.route, "status": reply.status,
                              "returned_ids": [], "proof": reply.proof})
        return reply

    def successful(self, options=None):
        self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        reply = self.dispatch("POST", scenario.FETCH_PATH, {
            "format": "xml", "export_option": options or {"export_block_id": True}})
        (self.workspace / "converted.xml").write_text(reply.json["data"]["document"]["content"], encoding="utf-8")
        return {"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": self.requests}

    def check(self, receipt, answer=None):
        return scenario.verify(self.workspace, self.answer if answer is None else answer,
                               receipt, scenario.TASK["expected"])

    def test_predeclared_source_and_public_schema(self):
        tasks = json.loads((EVALUATION / "tasks.development.json").read_text())
        declared = next(t for t in tasks["tasks"] if t["id"] == scenario.TASK["id"])
        self.assertEqual(declared["fixture"]["markdown"].encode(), (self.workspace / "input.md").read_bytes())
        self.assertEqual({"scenario", "answer_schema", "allow_synthetic_mutations"}, set(scenario.TASK["fixture"]))
        self.assertNotIn(scenario.DOCUMENT_ID, json.dumps(scenario.TASK))
        self.assertNotIn(scenario.NATIVE_XML, json.dumps(scenario.TASK))

    def test_create_fetch_and_independent_native_structure(self):
        receipt = self.successful()
        self.assertTrue(self.check(receipt))
        # Equivalent XML serialization is valid without relying on exact output bytes.
        path = self.workspace / "converted.xml"
        path.write_text(scenario.NATIVE_XML.replace("<p></p>", "<p />") + "\n", encoding="utf-8")
        self.assertTrue(self.check(receipt))
        self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml", "export_option": {
            "export_block_id": True, "export_style_attrs": True, "export_cite_extra_data": True}})
        self.assertTrue(self.check({"scenario": scenario.SCENARIO,
                                   "state": self.backend.snapshot(), "requests": self.requests}))

    def test_foreign_identity_id_payload_and_effects_rejected(self):
        cases = [
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE}, "bot"),
            ("POST", scenario.CREATE_PATH, {"format": "xml", "content": scenario.NATIVE_XML}, "user"),
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE + " "}, "user"),
            ("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE, "parent_token": "foreign"}, "user"),
            ("POST", scenario.FETCH_PATH, {"format": "xml"}, "user"),
            ("POST", scenario.CREATE_PATH + "/foreign/fetch", {"format": "xml"}, "user"),
            ("PUT", scenario.CREATE_PATH + "/foreign", {}, "user"),
            ("POST", "/open-apis/drive/v1/files/upload_all", {}, "user"),
            ("POST", scenario.CREATE_PATH + "?share=true", {}, "user"),
        ]
        for args in cases:
            with self.subTest(args=args), self.assertRaises(ValueError):
                self.dispatch(*args)
        self.assertFalse(self.backend.snapshot()["final"]["created"])
        self.assertEqual(0, self.backend.snapshot()["final"]["create_attempts"])

    def test_duplicate_create_partial_fetch_and_bad_types_rejected(self):
        self.successful()
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        for extra in ({"read_option": {"read_mode": "outline"}}, {"revision_id": True},
                      {"revision_id": 2}, {"export_option": {"export_block_id": 1}}):
            with self.subTest(extra=extra), self.assertRaises(ValueError):
                self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml", **extra})
        with self.assertRaises(ValueError):
            self.backend.dispatch("POST", scenario.CREATE_PATH, b'{"content":"a","content":"b"}',
                                  {"X-Lark-Proxy-Identity": "user"})

    def test_forged_claim_without_backend_evidence_fails(self):
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        self.assertFalse(self.check({"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": []}))
        receipt = self.successful()
        for altered in ({}, {**receipt, "requests": []}, {**receipt, "state": {}}, {**receipt, "scenario": "other"}):
            self.assertFalse(self.check(altered))
        altered = copy.deepcopy(receipt)
        altered["requests"][-1]["proof"]["native_sha256"] = "forged"
        self.assertFalse(self.check(altered))
        altered = copy.deepcopy(receipt)
        altered["requests"].append({"route": "permission_grant", "status": 200})
        self.assertFalse(self.check(altered))
        for answer in ({**self.answer, "document_id": "foreign"}, {**self.answer, "unsupported": ["invented"]},
                       {**self.answer, "artifact_path": "../converted.xml"}):
            self.assertFalse(self.check(receipt, answer))

    def test_empty_cell_unicode_structure_and_ids_cannot_be_forged(self):
        receipt = self.successful()
        wrong = [scenario.SOURCE, scenario.NATIVE_XML.replace("你好", "??"),
                 scenario.NATIVE_XML.replace('<td block-id="cell_empty"><p></p></td>', ""),
                 scenario.NATIVE_XML.replace("<p></p>", "<p> </p>"),
                 scenario.NATIVE_XML.replace("<p></p>", "<p>null</p>"),
                 scenario.NATIVE_XML.replace("cell_empty", "cell_other"),
                 scenario.NATIVE_XML + "unrelated text", '<!DOCTYPE a [<!ENTITY b "x">]><a>&b;</a>']
        for content in wrong:
            with self.subTest(content=content):
                (self.workspace / "converted.xml").write_text(content, encoding="utf-8")
                self.assertFalse(self.check(receipt))
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        (self.workspace / "input.md").write_text("changed", encoding="utf-8")
        self.assertFalse(self.check(receipt))

    def test_simple_fetch_cannot_prove_observed_native_ids(self):
        self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        reply = self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml"})
        self.assertNotIn("block-id", reply.json["data"]["document"]["content"])
        (self.workspace / "converted.xml").write_text(scenario.NATIVE_XML, encoding="utf-8")
        self.assertFalse(self.check({"scenario": scenario.SCENARIO,
                                    "state": self.backend.snapshot(), "requests": self.requests}))

    def test_unsupported_conversion_preserves_failure_and_prevents_replay(self):
        self.backend = scenario.Scenario(unsupported=True)
        reply = self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        self.assertEqual("failed", reply.json["data"]["result"])
        self.assertEqual(["synthetic Markdown table conversion unsupported"], reply.json["data"]["warnings"])
        self.assertFalse(self.backend.snapshot()["final"]["created"])
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml"})

    def upload(self, overrides=None):
        fields = {"file_name": "input.md", "parent_type": "ccm_import_open",
                  "size": str(len(scenario.SOURCE.encode())),
                  "extra": json.dumps({"obj_type": "docx", "file_extension": "md"}),
                  "file": scenario.SOURCE.encode()}
        fields.update(overrides or {})
        chunks = []
        for name, value in fields.items():
            filename = '; filename="input.md"' if name == "file" else ""
            chunks.append(('Content-Disposition: form-data; name="' + name + '"' + filename +
                           '\r\n\r\n').encode() + (value if isinstance(value, bytes) else value.encode()))
        raw = b"--boundary\r\n" + b"\r\n--boundary\r\n".join(chunks) + b"\r\n--boundary--\r\n"
        headers = signed_headers("POST", scenario.UPLOAD_PATH, raw)
        headers["Content-Type"] = "multipart/form-data; boundary=boundary"
        reply = self.backend.dispatch("POST", scenario.UPLOAD_PATH, raw, headers)
        self.requests.append({"route": reply.route, "status": reply.status, "returned_ids": [], "proof": reply.proof})
        return reply

    def import_body(self, **changes):
        return {"file_extension": "md", "file_token": scenario.MEDIA_TOKEN, "type": "docx",
                "file_name": "input", "point": {"mount_type": 1, "mount_key": ""}, **changes}

    def test_drive_import_create_poll_and_native_xml_artifact(self):
        self.assertEqual(scenario.MEDIA_TOKEN, self.upload().json["data"]["file_token"])
        report = {"file_scene": "lark-cli", "scene": "upload", "operation": "upload", "tags": {
            "code": "", "api_path": scenario.UPLOAD_PATH, "command": "drive +import",
            "resource_type": "media", "status": "success", "mount_point": "ccm_import_open",
            "file_token": scenario.MEDIA_TOKEN}}
        self.dispatch("POST", scenario.REPORT_PATH, report)
        self.assertEqual(scenario.IMPORT_TICKET, self.dispatch("POST", scenario.IMPORT_PATH,
                         self.import_body()).json["data"]["ticket"])
        polled = self.dispatch("GET", scenario.IMPORT_PATH + "/" + scenario.IMPORT_TICKET)
        self.assertEqual(0, polled.json["data"]["result"]["job_status"])
        self.assertEqual(scenario.DOCUMENT_ID, polled.json["data"]["result"]["token"])
        fetched = self.dispatch("POST", scenario.FETCH_PATH, {"format": "xml", "export_option": {"export_block_id": True}})
        (self.workspace / "converted.xml").write_text(fetched.json["data"]["document"]["content"], encoding="utf-8")
        receipt = {"scenario": scenario.SCENARIO, "state": self.backend.snapshot(), "requests": self.requests}
        self.assertTrue(self.check(receipt))
        for route in ("markdown_import_upload", "markdown_import_status"):
            tampered = copy.deepcopy(receipt)
            next(r for r in tampered["requests"] if r["route"] == route)["proof"] = {}
            self.assertFalse(self.check(tampered))
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.IMPORT_PATH, self.import_body())
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.CREATE_PATH, {"format": "markdown", "content": scenario.SOURCE})
        with self.assertRaises(ValueError):
            self.upload()

    def test_drive_import_rejects_changed_bytes_mounts_and_tokens(self):
        for override in ({"file": scenario.SOURCE.encode() + b" "}, {"size": "1"},
                         {"parent_type": "docx_image"}, {"parent_node": "foreign"},
                         {"extra": '{"obj_type":"sheet","file_extension":"md"}'}, {"file_name": "../input.md"}):
            with self.subTest(override=override), self.assertRaises(ValueError):
                self.upload(override)
        with self.assertRaises(ValueError):
            self.dispatch("POST", scenario.IMPORT_PATH, self.import_body())
        self.upload()
        for changes in ({"type": "sheet"}, {"file_token": "foreign"}, {"file_extension": "html"},
                        {"point": {"mount_type": 1, "mount_key": "foreign"}},
                        {"point": {"mount_type": True, "mount_key": ""}}, {"token": "existing"}):
            with self.subTest(changes=changes), self.assertRaises(ValueError):
                self.dispatch("POST", scenario.IMPORT_PATH, self.import_body(**changes))
        with self.assertRaises(ValueError):
            self.dispatch("GET", scenario.IMPORT_PATH + "/foreign")
        self.assertFalse(self.backend.snapshot()["final"]["created"])

    def test_drive_import_unsupported_job_is_not_success(self):
        self.backend = scenario.Scenario(unsupported=True)
        self.upload()
        self.dispatch("POST", scenario.IMPORT_PATH, self.import_body())
        result = self.dispatch("GET", scenario.IMPORT_PATH + "/" + scenario.IMPORT_TICKET).json["data"]["result"]
        self.assertEqual(3, result["job_status"])
        self.assertIn("unsupported", result["job_error_msg"])
        self.assertNotIn("token", result)
        self.assertFalse(self.backend.snapshot()["final"]["created"])

    def test_multipart_beyond_small_authored_input_remains_explicit_gap(self):
        for path in scenario.DRIVE_MULTIPART_PATHS:
            with self.subTest(path=path), self.assertRaisesRegex(ValueError, "^unsupported strategy:"):
                self.dispatch("POST", path)

    def test_foreign_and_malformed_mutation_paths_are_not_strategy_gaps(self):
        routes = [
            ("PUT", "/open-apis/drive/v1/import_tasks"),
            ("GET", "/open-apis/drive/v1/import_tasks"),
            ("POST", "/open-apis/drive/v1/import_tasks/tk_import"),
            ("POST", "/open-apis/drive/v1/import_tasks/"),
            ("POST", "/open-apis/drive/v1/medias/upload_all/extra"),
            ("POST", "/open-apis/drive/v1/medias/upload_all?share=true"),
            ("POST", "/open-apis/drive/v1/files/upload_all"),
            ("POST", "/open-apis/drive/v1/permissions/docxForeign/members"),
            ("GET", "/open-apis/drive/v1/import_tasks/../foreign"),
            ("GET", "/open-apis/drive/v1/import_tasks/.."),
            ("GET", "/open-apis/drive/v1/import_tasks/%2Fforeign"),
        ]
        for method, path in routes:
            with self.subTest(method=method, path=path), self.assertRaises(ValueError) as error:
                self.dispatch(method, path)
            self.assertNotIn("unsupported strategy:", str(error.exception))

    def test_snapshot_cannot_mutate_server_state(self):
        state = self.backend.snapshot()
        state["final"]["created"] = True
        self.assertFalse(self.backend.snapshot()["final"]["created"])


@unittest.skipUnless(os.environ.get("EVAL_MARKDOWN_BINARIES"),
                     "set EVAL_MARKDOWN_BINARIES to clean archived authsidecar binaries")
class MarkdownBinaryTests(unittest.TestCase):
    def test_each_binary_preserves_native_structure_and_successful_outcomes(self):
        from codex_runner import cli_environment
        from section_fixture import SectionFixture

        binaries = json.loads(os.environ["EVAL_MARKDOWN_BINARIES"])
        self.assertIsInstance(binaries, list)
        self.assertTrue(binaries, "at least one archived binary is required")
        for binary in binaries:
            self.assertIsInstance(binary, str)
            self.assertTrue(Path(binary).is_absolute() and Path(binary).is_file())
            for strategy in ("docs_ai", "drive_import"):
                with self.subTest(binary=binary, strategy=strategy), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config"):
                        (workspace / name).mkdir()
                    backend = scenario.Scenario()
                    backend.setup(workspace)
                    with SectionFixture(scenario.SCENARIO, backend.dispatch, backend.snapshot) as server:
                        env = {**cli_environment(workspace), **server.environment()}
                        for name in ("HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"):
                            env[name] = "http://127.0.0.1:9"
                        env["NO_PROXY"] = env["no_proxy"] = "localhost,127.0.0.1,::1"
                        create_args = (["docs", "+create", "--content", "@input.md", "--doc-format", "markdown"]
                                       if strategy == "docs_ai" else
                                       ["drive", "+import", "--file", "input.md", "--type", "docx"])
                        created = subprocess.run([binary, *create_args, "--as", "user"], cwd=workspace,
                                                 env=env, capture_output=True, timeout=20)
                        # Read back once even after an uncertain exit; never replay a create.
                        fetched = subprocess.run([binary, "docs", "+fetch", "--doc", scenario.DOCUMENT_ID,
                                                  "--doc-format", "xml", "--detail", "full", "--as", "user",
                                                  "--format", "pretty"], cwd=workspace, env=env,
                                                 capture_output=True, timeout=20)
                        (workspace / "converted.xml").write_bytes(fetched.stdout)
                    receipt = server.receipt()
                    self.assertTrue(receipt["requests"])
                    self.assertTrue(all(record["status"] == 200 and record["signature_valid"] is True
                                        for record in receipt["requests"]))
                    self.assertEqual(receipt["state"]["final"]["create_attempts"], 1)
                    self.assertEqual(receipt["state"]["final"]["strategy"], strategy)
                    answer = {"document_id": scenario.DOCUMENT_ID, "artifact_path": "converted.xml", "unsupported": []}
                    self.assertEqual(fetched.returncode, 0, fetched.stderr.decode())
                    self.assertTrue(scenario.verify(workspace, answer, receipt, scenario.TASK["expected"]))
                    # Backend success and artifact fidelity cannot hide a false uncertain CLI outcome.
                    self.assertEqual(created.returncode, 0, created.stderr.decode())


if __name__ == "__main__":
    unittest.main()
