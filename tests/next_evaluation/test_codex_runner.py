import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("codex_runner", ROOT / "evaluation/codex_runner.py")
runner = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(runner)


class CodexAccountingTests(unittest.TestCase):
    def stream(self, extra=()):
        return "\n".join(json.dumps(event) for event in [
            {"type": "thread.started", "thread_id": "synthetic"},
            {"type": "turn.started"},
            {"type": "item.completed", "item": {"type": "command_execution"}},
            {"type": "turn.completed", "usage": {"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}},
            *extra])

    def test_all_executable_manifests_have_provider_compatible_object_schemas(self):
        for name in ('foundation', 'contact', 'read-sections', 'scenarios', 'scaling', 'business'):
            tasks = json.loads((ROOT / ('evaluation/tasks.' + name + '.json')).read_text())['tasks']
            for task in tasks:
                with self.subTest(task=task['id']):
                    runner.validate_output_schema(task['fixture']['answer_schema'])

    def test_invalid_nested_output_schema_fails_before_process_or_workspace_setup(self):
        from unittest.mock import patch
        invalid = {'type': 'object', 'properties': {}, 'required': [], 'additionalProperties': True}
        for schema in (invalid, {'type': 'array', 'items': invalid},
                       {'type': 'object', 'properties': {'value': {'type': 'string'}},
                        'required': [], 'additionalProperties': False}):
            with self.subTest(schema=schema), patch.object(runner.subprocess, 'Popen') as process:
                with self.assertRaises(ValueError):
                    runner.run_agent({'fixture': {'answer_schema': schema}}, None, None, None)
                process.assert_not_called()

    def test_uses_provider_aggregate_without_byte_estimates(self):
        result = runner.parse_events(self.stream(), 0)
        self.assertTrue(result["accounting_complete"])
        self.assertEqual(result["turns"], [{"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}])
        self.assertEqual(result["usage_granularity"], "codex_user_turn")
        self.assertEqual(result["command_executions"], 1)
        self.assertEqual(result["control_trace"], [{"choices": [{"kind": "unparsed_shell"}], "exit_code": None}])

    def test_trace_keeps_only_public_names_not_content_paths_or_unknown_flags(self):
        command = 'TOKEN=private-value /private/workspace/feishu-cli-next docs +fetch --doc secret-document-id --keyword="private user text" --not-a-real-private-flag=secret && cat /private/secret-file'
        result = runner.command_choices(command)
        self.assertEqual(result, [{"kind": "cli", "command": ["docs", "+fetch"], "flags": ["--doc", "--keyword"]}, {"kind": "file_read"}])
        self.assertNotIn("private", json.dumps(result))
        self.assertNotIn("secret", json.dumps(result))

    def test_trace_supports_shell_wrapper_and_does_not_count_echo_as_cli(self):
        result = runner.command_choices('zsh -lc "lark-cli auth status --json; echo lark-cli docs +fetch"')
        self.assertEqual(result, [{"kind": "cli", "command": ["auth", "status"], "flags": ["--json"]}, {"kind": "other_shell"}])

    def test_discovery_trace_drops_query_and_payload(self):
        self.assertEqual(runner.command_choices('feishu-cli-next discover "private query" --limit 5'),
                         [{"kind": "cli", "command": ["discover"], "flags": ["--limit"]}])

    def test_new_public_guide_and_output_choices_are_retained(self):
        self.assertEqual(runner.command_choices('feishu-cli-next guide install --project private/path'),
                         [{"kind": "cli", "command": ["guide", "install"], "flags": ["--project"]}])
        self.assertEqual(runner.command_choices('feishu-cli-next discover private --flags --result-fields private'),
                         [{"kind": "cli", "command": ["discover"], "flags": ["--flags", "--result-fields"]}])

    def test_incomplete_and_failed_turns_are_unknown(self):
        for tail in ([{"type": "turn.started"}], [{"type": "turn.failed"}],
                     [{"type": "item.completed", "item": {"type": "mcp_tool_call"}}]):
            result = runner.parse_events(self.stream(tail), 1)
            self.assertFalse(result["accounting_complete"])
            self.assertEqual(result["turns"][0]["input_tokens"], 200)

    def test_nonzero_exit_keeps_complete_reported_cost(self):
        result = runner.parse_events(self.stream(), 7)
        self.assertTrue(result["accounting_complete"])

    def test_malformed_usage_never_passes(self):
        self.assertFalse(runner.parse_events("not-json", 0)["accounting_complete"])
        self.assertFalse(runner.parse_events('{"type":"turn.completed","usage":{}}', 0)["accounting_complete"])

    def test_config_disables_installed_skills_without_reading_them(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / "skills" / "sensitive-skill" / "SKILL.md"
            skill.parent.mkdir(parents=True)
            skill.write_text("DO NOT INJECT THIS")
            config = runner.configuration(root, root, "high")
            self.assertIn('model_reasoning_effort="high"', config)
            self.assertIn('sandbox_workspace_write.network_access=false', config)
            self.assertIn('features.plugins=false', config)
            self.assertIn('features.multi_agent=false', config)
            self.assertIn('features.apps=false', config)
            self.assertIn('project_doc_max_bytes=0', config)
            self.assertIn('enabled=false', next(value for value in config if value.startswith('skills.config=')))
            self.assertNotIn('DO NOT INJECT THIS', '\n'.join(config))


class CodexAdapterProcessTests(unittest.TestCase):
    def test_independent_verifier_checks_state_and_answer(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "feishu-config").mkdir()
            fake_cli = root / "cli.py"
            fake_cli.write_text('''import json, sys
assert sys.argv[1:] == ['auth','status','--json']
print(json.dumps({'ok':False,'error':{'subtype':'not_configured'}}),file=sys.stderr)
sys.exit(3)
''')
            task = json.loads((ROOT / "evaluation/tasks.foundation.json").read_text())["tasks"][0]
            request = {"workspace": str(root), "cli": str(fake_cli), "expected": task["expected"], "claimed_success": True}
            for correct_answer, mutation, expected in ((True, False, True), (False, False, False), (True, True, False)):
                (root / "answer.json").write_text(json.dumps(task["expected"]["answer"] if correct_answer else {"configured": True}))
                if mutation:
                    (root / "feishu-config/profile.json").write_text('{}')
                result = subprocess.run([sys.executable, str(ROOT / "tests/next_evaluation/python_fixture_entry.py"),
                                         "--entry", str(ROOT / "evaluation/foundation_verifier.py"), "--fixture", str(fake_cli)],
                                        input=json.dumps(request), capture_output=True, text=True, timeout=10)
                self.assertEqual(json.loads(result.stdout)["success"], expected)

    def test_complete_adapter_flow_with_fake_codex(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            workspace = root / "workspace"
            workspace.mkdir()
            guidance = root / "skill"
            guidance.mkdir()
            (guidance / "SKILL.md").write_text("Synthetic test guidance")
            fake = root / "codex.py"
            fake.write_text('''import json, pathlib, sys, os
if '--version' in sys.argv:
 print('codex-cli 0.146.0'); sys.exit(0)
assert '--ignore-user-config' in sys.argv
assert '--ephemeral' in sys.argv
assert '--ignore-rules' in sys.argv
assert sys.argv[sys.argv.index('--model')+1] == 'synthetic-model'
assert 'LARKSUITE_CLI_APP_SECRET' not in os.environ
assert os.environ['LARKSUITE_CLI_REMOTE_META'] == 'off'
assert 'Synthetic test guidance' in sys.stdin.read()
pathlib.Path(sys.argv[sys.argv.index('--output-last-message')+1]).write_text('{"configured":false}')
print(json.dumps({'type':'turn.started'}))
print(json.dumps({'type':'turn.completed','usage':{'input_tokens':200,'cached_input_tokens':80,'output_tokens':30}}))
''')
            request = {"workspace": str(workspace), "cli": str(fake), "skill": str(guidance / "SKILL.md"),
                       "prompt": "Inspect the synthetic config.", "settings": {"backend": "offline-foundation-v1",
                       "model": "synthetic-model", "tokenizer": "provider-undisclosed", "reasoning": "high",
                       "budget": {"timeout_seconds": 5}}, "fixture": {"answer_schema": {"type": "object", "properties": {}, "required": [], "additionalProperties": False}}}
            process = subprocess.run([sys.executable, str(ROOT / "tests/next_evaluation/python_fixture_entry.py"),
                                      "--entry", str(ROOT / "evaluation/codex_runner.py"), "--fixture", str(fake), "--",
                                      "--codex", str(fake), "--auth-home", str(root / "auth")],
                                     input=json.dumps(request), capture_output=True, text=True, timeout=15)
            self.assertEqual(process.returncode, 0, process.stderr)
            result = json.loads(process.stdout)
            self.assertTrue(result["accounting_complete"])
            self.assertEqual(result["turns"][0]["cached_input_tokens"], 80)
            self.assertTrue((workspace / "answer.json").is_file())
            for name in ("lark-cli", "feishu-cli-next"):
                alias = workspace / "bin" / (name + ".exe" if os.name == "nt" else name)
                self.assertEqual(alias.read_bytes(), fake.read_bytes())
                if os.name != "nt":
                    self.assertEqual(alias.resolve(), fake.resolve())
            self.assertNotIn("Synthetic test guidance", process.stdout)


if __name__ == "__main__":
    unittest.main()
