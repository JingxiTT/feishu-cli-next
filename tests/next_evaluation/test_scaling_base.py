"""Controlled scaling fixture checks; optional clean CLI runs, never models."""
from copy import deepcopy
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from urllib.parse import urlsplit

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "evaluation"))
from scaling.base import (Scenario, SCENARIO, SIZES, BASE, TABLE, PREFIX, SENTINEL,
                          INITIAL, FIELDS, MAPPING, task, source_row, native_row, verify)
from section_fixture import SectionFixture, signed_headers, valid_signature


class BaseScalingTests(unittest.TestCase):
    def start(self, n=100):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.workspace = Path(temp.name)
        self.scenario = Scenario(n)
        self.scenario.setup(self.workspace)
        self.spec = task(n)
        self.requests = []

    def call(self, method, uri, data=None, identity="user", raw=None):
        body = raw if raw is not None else (json.dumps(data, ensure_ascii=False).encode() if data is not None else b"")
        headers = signed_headers(method, uri, body, identity)
        record = {"method": method, "uri": urlsplit(uri).path, "signature_valid": valid_signature(method, uri, body, headers)}
        try:
            reply = self.scenario.dispatch(method, uri, body, headers)
        except ValueError:
            self.requests.append(dict(record, route="rejected", status=403))
            raise
        self.requests.append(dict(record, route=reply.route, status=reply.status, proof=deepcopy(reply.proof)))
        return reply.json["data"]

    def receipt(self):
        return {"scenario": SCENARIO, "requests": deepcopy(self.requests), "state": self.scenario.snapshot()}

    def fill(self, order=None, transform=None, names=False, batch_size=200):
        ids = []
        indexes = list(range(self.scenario.row_count)) if order is None else order
        for start in range(0, len(indexes), batch_size):
            rows = [native_row(i) for i in indexes[start:start+batch_size]]
            if transform:
                rows = [transform(deepcopy(row)) for row in rows]
            if names:
                alias = {f["id"]: f["name"] for f in FIELDS}
                rows = [{alias[key]: value for key, value in row.items()} for row in rows]
            ids.extend(self.call("POST", PREFIX + "/records/batch_create", {"create_records": rows})["record_id_list"])
        self.report(ids)
        return ids

    def report(self, ids):
        (self.workspace / "import-report.json").write_text(json.dumps({"created": len(ids), "record_ids": ids}), encoding="utf-8")

    def accepted(self, answer=None, receipt=None):
        return verify(self.workspace, answer if answer is not None else {"imported_count": self.scenario.row_count, "report_path": "import-report.json"}, receipt if receipt is not None else self.receipt(), self.spec["expected"])

    def test_all_sizes_complete_preserve_exact_native_content(self):
        for n in SIZES:
            with self.subTest(rows=n):
                self.start(n)
                original = {name: (self.workspace / name).read_bytes() for name in ("input.json", "mapping.json")}
                ids = self.fill()
                self.assertEqual(len(ids), n)
                self.assertTrue(self.accepted())
                snapshot = self.scenario.snapshot()
                self.assertEqual(len(snapshot["records"]), n+1)
                self.assertEqual(snapshot["records"][SENTINEL], INITIAL[SENTINEL])
                row = snapshot["records"][ids[0]]
                self.assertIs(row["fld_scale_checked"], True)
                self.assertIsNone(row["fld_scale_nullable"])
                self.assertNotIn("fld_scale_optional", row)
                self.assertEqual(row["fld_scale_empty"], "")
                self.assertEqual(row["fld_scale_code"], "000000")
                self.assertEqual(snapshot["records"][ids[30]]["fld_scale_number"], 9007199254740991)
                for name, raw in original.items():
                    self.assertEqual((self.workspace / name).read_bytes(), raw)

    def test_one_prefix_family_type_only_public_schema_and_fresh_state(self):
        inputs = {}
        for n in SIZES:
            self.start(n)
            inputs[n] = json.loads((self.workspace / "input.json").read_text())["result"]["data"]
            self.assertEqual(inputs[n]["range"], f"A1:G{n}")
            self.assertEqual(inputs[n]["count"], n)
            self.assertEqual(json.loads((self.workspace / "mapping.json").read_text()), MAPPING)
            public = json.dumps(self.spec["fixture"])
            for hidden in ("const", "enum", "preexisting-control", "fixture_self", "測定", "expected"):
                self.assertNotIn(hidden, public)
            self.assertEqual(self.scenario.snapshot()["records"], INITIAL)
        self.assertEqual(inputs[100]["cells"], inputs[1000]["cells"][:100])
        self.assertEqual(inputs[1000]["cells"], inputs[10000]["cells"][:1000])
        for n in (True, 0, 200, 100.0, "100", 10001):
            with self.subTest(invalid=n), self.assertRaises(ValueError):
                task(n)

    def test_equivalent_field_names_reordered_rows_and_single_create(self):
        self.start()
        indexes = list(reversed(range(100)))
        first = self.call("POST", PREFIX + "/records", native_row(indexes.pop(0)))["record_id"]
        ids = [first]
        alias = {f["id"]: f["name"] for f in FIELDS}
        rows = [{alias[k]: v for k, v in native_row(i).items()} for i in indexes]
        ids += self.call("POST", PREFIX + "/records/batch_create", {"create_records": rows})["record_id_list"]
        self.report(list(reversed(ids)))
        self.assertTrue(self.accepted(), "input order, report order and create strategy are not prescribed")

    def test_schema_projection_pagination_and_missing_ids(self):
        self.start()
        fields = self.call("GET", PREFIX + "/fields?offset=1&limit=2")
        self.assertEqual(fields["items"], FIELDS[1:3])
        self.assertIs(fields["has_more"], True)
        self.assertEqual(self.call("GET", PREFIX + "/fields/fld_scale_checked"), FIELDS[2])
        ids = self.fill()
        page = self.call("GET", PREFIX + "/records?offset=0&limit=1&field_id=Code")
        self.assertEqual(page["record_id_list"], [SENTINEL])
        self.assertEqual(page["data"], [["009900"]])
        self.assertIs(page["has_more"], True)
        page = self.call("GET", PREFIX + "/records?offset=100&limit=1")
        self.assertEqual(page["record_id_list"], [ids[-1]])
        self.assertIs(page["has_more"], False)
        detail = self.call("POST", PREFIX + "/records/batch_get", {"record_id_list": [ids[0], "missing"], "select_fields": ["Optional", "Nullable", "Empty"]})
        self.assertEqual(detail["record_not_found"], ["missing"])
        self.assertEqual(detail["data"], [[None, None, ""]])
        self.assertTrue(self.accepted(), "native read null representation does not erase stored key presence")

    def test_invalid_batches_reject_atomically_and_no_extra_row(self):
        cases = [
            {"create_records": [native_row(0), dict(native_row(1), fld_scale_checked=1)]},
            {"create_records": [native_row(0), native_row(0)]},
            {"create_records": [dict(native_row(0), fld_unknown="x")]},
            {"create_records": [dict(native_row(0), **{"Row key": "duplicate alias"})]},
            {"create_records": [dict(native_row(0), fld_scale_number=False)]},
            {"create_records": [dict(native_row(0), fld_scale_number="0")]},
            {"create_records": [dict(native_row(0), fld_scale_code=0)]},
            {"create_records": [dict(native_row(0), fld_scale_key=None)]},
            {"create_records": [dict(native_row(0), fld_scale_key="preexisting-control")]},
            {"create_records": []}, {"create_records": [native_row(i) for i in range(201)]},
            {"create_records": [native_row(0)], "extra": True},
        ]
        for data in cases:
            self.start()
            with self.subTest(data=str(data)[:100]), self.assertRaises(ValueError):
                self.call("POST", PREFIX + "/records/batch_create", data)
            self.assertEqual(self.scenario.snapshot()["records"], INITIAL)
            self.assertEqual(self.scenario.snapshot()["effects"], [])
        self.start()
        self.fill()
        with self.assertRaises(ValueError):
            self.call("POST", PREFIX + "/records", native_row(101))
        self.assertFalse(self.accepted(), "an unintended rejected create cannot be hidden by a correct prior import")

    def test_unintended_transient_routes_actor_and_malformed_requests_fail(self):
        for method, uri, data, identity in (
            ("PATCH", PREFIX + "/records/" + SENTINEL, {"fld_scale_number": 0}, "user"),
            ("DELETE", PREFIX + "/records/" + SENTINEL, None, "user"),
            ("PUT", PREFIX + "/fields/fld_scale_number", {"type": "text"}, "user"),
            ("POST", PREFIX + "/records", native_row(0), "bot"),
            ("POST", PREFIX.replace(BASE, "foreign") + "/records", native_row(0), "user"),
        ):
            self.start()
            with self.assertRaises(ValueError):
                self.call(method, uri, data, identity=identity)
            self.fill()
            self.assertFalse(self.accepted(), "later success must not hide a transient unintended attempt")
        self.start()
        for raw in (b'{"create_records":[],"create_records":[]}', b'{"create_records":NaN}', b'[]'):
            with self.assertRaises(ValueError):
                self.call("POST", PREFIX + "/records/batch_create", raw=raw)
        for uri in (PREFIX + "/records?limit=0", PREFIX + "/records?offset=-1", PREFIX + "/records?limit=1&limit=2", PREFIX + "/fields?foreign=1"):
            with self.assertRaises(ValueError):
                self.call("GET", uri)

    def test_supported_but_corrupted_content_cannot_pass_or_be_restored_silently(self):
        for change in (
            lambda row: dict(row, fld_scale_optional=None),
            lambda row: dict(row, fld_scale_nullable=""),
            lambda row: dict(row, fld_scale_number=1),
            lambda row: dict(row, fld_scale_code=str(int(row["fld_scale_code"]))),
            lambda row: dict(row, fld_scale_key=row["fld_scale_key"].replace("café", "cafe")),
        ):
            self.start()
            ids = self.fill(transform=change)
            self.assertFalse(self.accepted())
            # Simulate a backend snapshot restoration without changing its actual
            # historical create effects; independent proof must still reject it.
            for i, record_id in enumerate(ids):
                self.scenario.records[record_id] = native_row(i)
            self.assertFalse(self.accepted(), "restored final state cannot conceal corrupted create evidence")

    def test_answer_state_proof_report_and_input_tampering_fail(self):
        self.start()
        ids = self.fill()
        good = self.receipt()
        self.assertTrue(self.accepted())
        for answer in ({"imported_count": "100", "report_path": "import-report.json"},
                       {"imported_count": True, "report_path": "import-report.json"},
                       {"imported_count": 100, "report_path": "../import-report.json"}):
            self.assertFalse(self.accepted(answer=answer))
        for modify in (
            lambda r: r["state"]["records"].pop(ids[-1]),
            lambda r: r["state"]["records"][SENTINEL].update(fld_scale_checked=False),
            lambda r: r["state"]["records"][ids[0]].update(fld_scale_checked=1),
            lambda r: r["state"]["records"][ids[0]].update(fld_scale_optional=None),
            lambda r: r["state"]["records"][ids[1]].update(fld_scale_key=native_row(0)["fld_scale_key"]),
            lambda r: r["state"]["fields"][0].update(name="Changed"),
            lambda r: r["state"].update(effects=[]),
            lambda r: r["requests"][0].update(signature_valid=False),
            lambda r: r["requests"][0].update(proof={}),
            lambda r: r["requests"][0].update(uri=PREFIX + "/foreign"),
            lambda r: r.update(requests=[]),
        ):
            altered = deepcopy(good)
            modify(altered)
            self.assertFalse(self.accepted(receipt=altered))
        for report in ({"created": 100, "record_ids": ids[:-1]}, {"created": 100, "record_ids": ids[:-1]+[ids[0]]},
                       {"created": 100, "record_ids": ids[:-1]+[SENTINEL]}, {"created": 100, "record_ids": ids, "status": "unknown"},
                       {"created": 100, "record_ids": ids, "completed_batches": 2}):
            (self.workspace / "import-report.json").write_text(json.dumps(report))
            self.assertFalse(self.accepted())
        self.report(ids)
        raw = (self.workspace / "input.json").read_bytes()
        (self.workspace / "input.json").write_bytes(raw+b" ")
        self.assertFalse(self.accepted(), "even a semantically identical input rewrite changes required bytes")
        (self.workspace / "input.json").write_bytes(raw)
        (self.workspace / "backend-receipt.json").write_text(json.dumps(good))
        self.assertFalse(self.accepted(receipt={"scenario": SCENARIO, "state": Scenario(100).snapshot(), "requests": []}))
        report = self.workspace / "import-report.json"
        report.rename(self.workspace / "alternate.json")
        report.symlink_to(self.workspace / "alternate.json")
        self.assertFalse(self.accepted(), "report must be the requested regular file")

    def test_generator_matches_fresh_contract_and_all_batches_fit_server(self):
        contract = json.loads((ROOT / "evaluation/scaling/base-contract.json").read_text())
        namespace = {}
        exec(compile(contract["input"]["generator_python"], "<declared generator>", "exec"), namespace)
        for i in range(10000):
            self.assertEqual(source_row(i), namespace["source_row"](i))
        for start in range(0, 10000, 200):
            raw = json.dumps({"create_records": [native_row(i) for i in range(start, start+200)]}, ensure_ascii=False).encode()
            self.assertLess(len(raw), 65536)


@unittest.skipUnless(os.environ.get("EVAL_SCALING_BASE_CLI"), "set EVAL_SCALING_BASE_CLI=1 for supplied clean archive binary checks")
class BaseScalingBinaryTests(unittest.TestCase):
    def test_clean_official_and_next_complete_each_size(self):
        from codex_runner import cli_environment
        paths = json.loads((ROOT / ".tmp/evaluation-paths.json").read_text())
        contract = json.loads((ROOT / "evaluation/scaling/base-contract.json").read_text())
        for n in SIZES:
            for arm in ("official", "next"):
                with self.subTest(rows=n, arm=arm), tempfile.TemporaryDirectory() as directory:
                    workspace = Path(directory)
                    for name in ("home", "tmp", "feishu-config", "bin"):
                        (workspace / name).mkdir()
                    binary = paths[arm+"_cli"]
                    (workspace / "bin/lark-cli").symlink_to(binary)
                    scenario = Scenario(n)
                    scenario.setup(workspace)
                    with SectionFixture(SCENARIO, dispatch=scenario.dispatch, snapshot=scenario.snapshot) as server:
                        env = {**cli_environment(workspace), **server.environment()}
                        if arm == "official":
                            # Admissible local file batching, with one CLI output
                            # captured per batch and only a bounded summary printed.
                            command = [sys.executable, "-c", contract["valid_strategies"]["baseline_file_helper_python"]]
                        else:
                            command = [binary, "base", "+record-import", "--base-token", BASE, "--table-id", TABLE,
                                       "--artifact", "@input.json", "--mapping", "@mapping.json", "--output", "import-report.json", "--as", "user"]
                        process = subprocess.run(command, cwd=workspace, env=env, capture_output=True, text=True, timeout=180)
                        self.assertEqual(process.returncode, 0, process.stderr)
                        answer = {"imported_count": n, "report_path": "import-report.json"}
                        self.assertTrue(verify(workspace, answer, server.receipt(), task(n)["expected"]), process.stdout[:1000])
                        self.assertEqual(len(scenario.effects), (n+199)//200)
                        self.assertLess(len(process.stdout.encode()), 10000, "output-bound check is not provider token evidence")


if __name__ == "__main__":
    unittest.main()
