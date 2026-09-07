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
                          INITIAL, FIELDS, MAPPING, BASE_PREFIX, BASE_META, TABLE_META, VIEWS, VIEW_ID, FIXTURE_VERSION,
                          VERIFICATION_DIAGNOSTICS, verification_diagnostics, task, source_row, native_row, verify)
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
        except ValueError as error:
            rejection = "unsupported_strategy" if str(error).startswith("unsupported strategy:") else "rejected_request"
            self.requests.append(dict(record, route="rejected", status=403, rejection=rejection))
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
        self.assertEqual(fields["fields"], FIELDS[1:3])
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
                       {"created": 100, "record_ids": ids[:-1]+[SENTINEL]}, {"created": 99, "record_ids": ids},
                       {"created": "100", "record_ids": ids}):
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

    def test_metadata_read_prerequisites_are_native_and_preserved(self):
        self.start()
        self.assertEqual(self.scenario.snapshot()["fixture_version"], FIXTURE_VERSION)
        self.assertEqual(self.call("GET", BASE_PREFIX), BASE_META)
        self.assertEqual(self.call("GET", PREFIX), TABLE_META)
        tables = self.call("GET", BASE_PREFIX + "/tables?offset=0&limit=50")
        self.assertEqual(tables, {"tables": [TABLE_META], "total": 1, "has_more": False})
        fields = self.call("GET", PREFIX + "/fields?offset=0&limit=100")
        self.assertEqual(fields, {"fields": FIELDS, "total": 7, "has_more": False})
        views = self.call("GET", PREFIX + "/views?offset=0&limit=100")
        self.assertEqual(views, {"views": VIEWS, "total": 1, "has_more": False})
        self.assertEqual(self.call("GET", PREFIX + "/views/" + VIEW_ID), VIEWS[0])
        self.fill()
        self.call("GET", PREFIX + "/records?view_id=" + VIEW_ID + "&limit=2000")
        self.assertTrue(self.accepted())
        for kind in ("base", "table", "views"):
            altered = self.receipt()
            if kind == "views":
                altered["state"][kind][0]["name"] = "changed"
            else:
                altered["state"][kind]["name"] = "changed"
            self.assertFalse(self.accepted(receipt=altered))
        for uri in (PREFIX + "?foreign=1", BASE_PREFIX + "/tables?limit=101", PREFIX + "/views?limit=201", PREFIX + "/views?offset=0&offset=1", PREFIX + "/views/foreign", PREFIX + "/records?view_id=foreign", PREFIX + "/records?view_id=" + VIEW_ID + "&view_id=" + VIEW_ID):
            with self.subTest(uri=uri), self.assertRaises(ValueError):
                self.call("GET", uri)
        self.assertFalse(self.accepted(), "valid metadata reads never excuse rejected requests")

    def test_known_unsupported_read_is_distinct_from_unknown_or_invalid_route(self):
        self.start()
        with self.assertRaisesRegex(ValueError, "^unsupported strategy:"):
            self.call("GET", PREFIX + "/views/" + VIEW_ID + "/filter")
        self.assertEqual(self.requests[-1]["rejection"], "unsupported_strategy")
        for method, uri in (("GET", PREFIX + "/views/foreign/filter"), ("GET", PREFIX + "/views/" + VIEW_ID + "/filter?invalid=1"), ("GET", PREFIX + "/unrecognized"), ("PUT", PREFIX + "/views/" + VIEW_ID + "/filter")):
            with self.assertRaises(ValueError):
                self.call(method, uri)
            self.assertEqual(self.requests[-1]["rejection"], "rejected_request")
        self.assertEqual(self.scenario.snapshot()["records"], INITIAL)

    def test_optional_report_metadata_is_not_a_native_schema_requirement(self):
        self.start()
        ids = self.fill(batch_size=50)
        minimal = {"created": 100, "record_ids": ids}
        for report in (minimal, dict(minimal, status="success"), dict(minimal, completed_batches=1),
                       dict(minimal, binding="local batch client", table_id="display label", requested="all input rows")):
            (self.workspace / "import-report.json").write_text(json.dumps(report))
            self.assertTrue(self.accepted(), "only required report fields contribute evidence")
        for report in ({"created": 99, "record_ids": ids}, {"created": 100, "record_ids": ids[:-1]}, {"created": True, "record_ids": ids}):
            (self.workspace / "import-report.json").write_text(json.dumps(report))
            self.assertFalse(self.accepted())

    def test_fixed_diagnostics_discriminate_failures_without_content(self):
        self.start()
        ids = self.fill()
        good = self.receipt()
        answer = {"imported_count": 100, "report_path": "import-report.json"}
        def reasons(receipt=good, claimed=answer):
            result = verification_diagnostics(self.workspace, claimed, receipt, self.spec["expected"])
            self.assertTrue(set(result).issubset(VERIFICATION_DIAGNOSTICS))
            return result
        self.assertEqual(reasons(), [])
        self.assertEqual(reasons(claimed=dict(answer, report_path="elsewhere")), ["base_scaling_report_path"])
        (self.workspace / "import-report.json").write_text("not JSON: private content")
        self.assertEqual(reasons(), ["base_scaling_report_format"])
        self.report(ids)
        for code, modify in (
            ("base_scaling_missing_rows", lambda r: r["state"]["records"].pop(ids[-1])),
            ("base_scaling_typed_cells", lambda r: r["state"]["records"][ids[0]].update(fld_scale_optional=None)),
            ("base_scaling_metadata_changed", lambda r: r["state"]["table"].update(name="private content")),
            ("base_scaling_bad_mutation_proof", lambda r: r["requests"][0].update(proof={})),
            ("base_scaling_api_rejected", lambda r: r["requests"].append({"route": "rejected", "status": 403, "rejection": "rejected_request"})),
            ("base_scaling_unsupported_strategy", lambda r: r["requests"].append({"route": "rejected", "status": 403, "rejection": "unsupported_strategy"})),
        ):
            altered = deepcopy(good)
            modify(altered)
            self.assertEqual(reasons(receipt=altered), [code])

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
                        common = ["--base-token", BASE, "--as", "user", "--format", "json"]
                        table_get = subprocess.run([binary, "base", "+table-get", "--table-id", TABLE, *common], cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                        self.assertEqual(table_get.returncode, 0, table_get.stderr)
                        detail = json.loads(table_get.stdout)["data"]
                        self.assertEqual(detail["table"], TABLE_META)
                        self.assertEqual(detail["fields"], FIELDS)
                        self.assertEqual(detail["views"], VIEWS)
                        for args in (("+base-get",), ("+table-list",), ("+view-list", "--table-id", TABLE), ("+view-get", "--table-id", TABLE, "--view-id", VIEW_ID)):
                            read = subprocess.run([binary, "base", *args, *common], cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                            self.assertEqual(read.returncode, 0, read.stderr)
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
                        native = json.loads((workspace / "import-report.json").read_text())
                        minimal = {key: native[key] for key in ("created", "record_ids")}
                        for report in (native, minimal, dict(minimal, status="success"), dict(minimal, completed_batches=len(scenario.effects)+1)):
                            (workspace / "import-report.json").write_text(json.dumps(report))
                            self.assertTrue(verify(workspace, answer, server.receipt(), task(n)["expected"]))
                        for report in (dict(minimal, created=n-1), dict(minimal, record_ids=minimal["record_ids"][:-1])):
                            (workspace / "import-report.json").write_text(json.dumps(report))
                            self.assertFalse(verify(workspace, answer, server.receipt(), task(n)["expected"]))
                        (workspace / "import-report.json").write_text(json.dumps(native))
                        self.assertEqual(len(scenario.effects), (n+199)//200)
                        self.assertLess(len(process.stdout.encode()), 10000, "output-bound check is not provider token evidence")

    def test_clean_binaries_mark_known_unmodeled_view_read_as_fixture_unsupported(self):
        from codex_runner import cli_environment
        paths = json.loads((ROOT / ".tmp/evaluation-paths.json").read_text())
        for arm in ("official", "next"):
            with self.subTest(arm=arm), tempfile.TemporaryDirectory() as directory:
                workspace = Path(directory)
                for name in ("home", "tmp", "feishu-config", "bin"):
                    (workspace / name).mkdir()
                scenario = Scenario(100)
                scenario.setup(workspace)
                with SectionFixture(SCENARIO, dispatch=scenario.dispatch, snapshot=scenario.snapshot) as server:
                    process = subprocess.run([paths[arm+"_cli"], "base", "+view-get-filter", "--base-token", BASE, "--table-id", TABLE, "--view-id", VIEW_ID, "--as", "user"], cwd=workspace, env={**cli_environment(workspace), **server.environment()}, capture_output=True, text=True, timeout=20)
                    self.assertNotEqual(process.returncode, 0)
                    receipt = server.receipt()
                    rejected = [r for r in receipt["requests"] if r["route"] == "rejected"]
                    self.assertEqual(len(rejected), 1)
                    self.assertEqual(rejected[0]["rejection"], "unsupported_strategy")
                    self.assertEqual(rejected[0]["unsupported_reason"], "unsupported_strategy_other")
                    self.assertEqual(receipt["state"]["records"], INITIAL)


if __name__ == "__main__":
    unittest.main()
