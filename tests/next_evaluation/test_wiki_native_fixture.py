"""Source-backed Wiki-to-native-Docx fixture equivalence; no model or live runs.

Native route/identity/shape evidence: official oapi-sdk-go/v3 v3.7.2
service/docx/v1/resource.go document.RawContent and model.go
RawContentDocumentReqBuilder.Lang / RawContentDocumentRespData.
The SDK supplies lang=0 as its example, not a documented language-name enum.
"""
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from codex_runner import cli_environment
from read_fixture import dispatcher, load_runtime, verify
from section_fixture import SectionFixture, signed_headers


def workspace_setup(directory):
    workspace = Path(directory)
    for name in ("home", "tmp", "feishu-config"):
        (workspace / name).mkdir()
    return workspace


def verifier_request(runtime, workspace):
    return {"workspace": str(workspace), "fixture": {"runtime_section": "wiki"},
            "expected": runtime["task"]["expected"]}


class WikiNativeFixtureTests(unittest.TestCase):
    def setUp(self):
        self.runtime = load_runtime("wiki")
        self.answer = self.runtime["task"]["expected"]["answer"]
        self.path = "/open-apis/docx/v1/documents/" + self.answer["document_id"] + "/raw_content"
        self.dispatch = dispatcher(self.runtime)

    def test_full_native_text_is_an_equivalent_read_for_both_identities(self):
        before = copy.deepcopy(self.runtime)
        for identity in ("user", "bot"):
            for suffix in ("", "?lang=0"):
                with self.subTest(identity=identity, suffix=suffix):
                    route, payload, _ = self.dispatch("GET", self.path + suffix, b"", {"X-Lark-Proxy-Identity": identity})
                    self.assertEqual(route, "docx_raw_content")
                    self.assertEqual(payload, {"code": 0, "msg": "success", "data": {"content": self.answer["text"]}})
        self.assertEqual(self.runtime, before)
        groups = self.runtime["task"]["expected"]["required_reads"]
        self.assertEqual(groups[0], ["wiki_resolve"])
        self.assertCountEqual(groups[1], ["docs_fetch_xml", "docs_fetch_markdown", "docx_raw_content"])

    def test_invalid_target_query_identity_and_method_never_match(self):
        cases = [("GET", self.path.replace(self.answer["document_id"], "wrongDocument"), "user"),
                 ("GET", self.path + "?lang=", "user"),
                 ("GET", self.path + "?lang=English", "user"),
                 ("GET", self.path + "?lang=999", "user"),
                 ("GET", self.path + "?lang=0&lang=0", "user"),
                 ("GET", self.path + "?page_token=next", "user"),
                 ("GET", self.path + "?revision_id=999", "user"),
                 ("GET", self.path + "?lang=0&scope=keyword", "user"),
                 ("GET", self.path, "other"), ("POST", self.path, "user")]
        before = copy.deepcopy(self.runtime)
        for method, uri, identity in cases:
            with self.subTest(method=method, uri=uri, identity=identity), self.assertRaises(ValueError):
                self.dispatch(method, uri, b"", {"X-Lark-Proxy-Identity": identity})
        self.assertEqual(self.runtime, before)

    def test_signed_native_read_alone_does_not_replace_wiki_resolution_proof(self):
        with tempfile.TemporaryDirectory() as directory, SectionFixture(self.runtime["task"]["fixture"]["scenario"], self.dispatch) as server:
            workspace = workspace_setup(directory)
            request = Request(server.environment()["LARKSUITE_CLI_AUTH_PROXY"] + self.path,
                              headers=signed_headers("GET", self.path), method="GET")
            with urlopen(request, timeout=5) as response:
                self.assertEqual(json.load(response)["data"]["content"], self.answer["text"])
            (workspace / "answer.json").write_text(json.dumps(self.answer), encoding="utf-8")
            server.write_receipt(workspace)
            failures = []
            self.assertFalse(verify(verifier_request(self.runtime, workspace), failures))
            self.assertEqual(failures, ["required_reads_missing"])
            record = server.receipt()["requests"][0]
            self.assertTrue(record["signature_valid"])
            self.assertEqual(record["native_read"], "docx_raw_content")

    def test_signed_invalid_read_is_rejected_and_ledgered(self):
        with SectionFixture(self.runtime["task"]["fixture"]["scenario"], self.dispatch) as server:
            for uri in (self.path.replace(self.answer["document_id"], "wrongDocument"), self.path + "?lang=0&lang=0", self.path + "?revision_id=9"):
                request = Request(server.environment()["LARKSUITE_CLI_AUTH_PROXY"] + uri,
                                  headers=signed_headers("GET", uri), method="GET")
                with self.assertRaises(HTTPError) as caught:
                    urlopen(request, timeout=5)
                self.assertEqual(caught.exception.code, 403)
                caught.exception.close()
            for record in server.receipt()["requests"]:
                self.assertEqual(record["route"], "rejected")
                self.assertEqual(record["status"], 403)
                self.assertTrue(record["signature_valid"])


@unittest.skipUnless(os.environ.get("EVAL_WIKI_BINARY_PATHS"), "set EVAL_WIKI_BINARY_PATHS to the archived evaluation paths JSON")
class WikiNativeBinaryTests(unittest.TestCase):
    def test_pinned_official_and_candidate_native_read_proof(self):
        paths = json.loads(Path(os.environ["EVAL_WIKI_BINARY_PATHS"]).read_text())
        self.assertEqual(paths["official_revision"], "7fd6ef3c07182257ce776cdc5a614e122d5bd4b3")
        self.assertRegex(paths["next_revision"], r"^[0-9a-f]{40}$")
        self.assertRegex(paths["next_binary_sha256"], r"^[0-9a-f]{64}$")
        self.assertEqual(hashlib.sha256(Path(paths["next_cli"]).read_bytes()).hexdigest(), paths["next_binary_sha256"])
        runtime = load_runtime("wiki")
        expected = runtime["task"]["expected"]["answer"]
        initial_runtime = copy.deepcopy(runtime)
        for arm in ("official", "next"):
            binary = paths[arm + "_cli"]
            self.assertTrue(Path(binary).is_file())
            for identity in ("user", "bot"):
                for params in (None, {"lang": 0}):
                    with self.subTest(arm=arm, identity=identity, params=params), tempfile.TemporaryDirectory() as directory, SectionFixture(runtime["task"]["fixture"]["scenario"], dispatcher(runtime)) as server:
                        workspace = workspace_setup(directory)
                        env = {**cli_environment(workspace), **server.environment()}
                        resolved = subprocess.run([binary, "wiki", "+node-get", "--node-token", expected["node_token"], "--as", identity],
                                                  cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                        self.assertEqual(resolved.returncode, 0, resolved.stderr)
                        node = json.loads(resolved.stdout)["data"]
                        self.assertEqual(node["obj_token"], expected["document_id"])
                        self.assertEqual(node["obj_type"], "docx")
                        uri = "/open-apis/docx/v1/documents/" + node["obj_token"] + "/raw_content"
                        args = [binary, "api", "GET", uri, "--as", identity]
                        if params is not None:
                            args += ["--params", json.dumps(params)]
                        native = subprocess.run(args, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                        self.assertEqual(native.returncode, 0, native.stderr)
                        content = json.loads(native.stdout)["data"]["content"]
                        self.assertEqual(content, expected["text"])
                        answer = {"node_token": node["node_token"], "document_id": node["obj_token"], "obj_type": node["obj_type"], "text": content}
                        (workspace / "answer.json").write_text(json.dumps(answer, ensure_ascii=False), encoding="utf-8")
                        server.write_receipt(workspace)
                        failures = []
                        self.assertTrue(verify(verifier_request(runtime, workspace), failures), failures)
                        records = server.receipt()["requests"]
                        self.assertTrue(all(record["status"] == 200 and record["signature_valid"] for record in records))
                        self.assertEqual(sum(record["route"] == "wiki_resolve" for record in records), 1)
                        native_records = [record for record in records if record["route"] == "docx_raw_content"]
                        self.assertEqual(len(native_records), 1)
                        self.assertEqual(native_records[0]["native_read"], "docx_raw_content")
                        self.assertFalse(any(record["route"].startswith("docs_fetch") for record in records))
        self.assertEqual(runtime, initial_runtime)


if __name__ == "__main__":
    unittest.main()
