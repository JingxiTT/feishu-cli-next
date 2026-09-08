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
        for name in ('foundation', 'contact', 'read-sections', 'scenarios', 'scaling', 'business', 'paired'):
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
        self.assertEqual(result["accounting_diagnostics"], [])
        self.assertEqual(result["turns"], [{"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}])
        self.assertEqual(result["usage_granularity"], "codex_user_turn")
        self.assertEqual(result["command_executions"], 1)
        self.assertEqual(result["control_trace"], [{"choices": [{"kind": "unparsed_shell"}], "exit_code": None}])

    def test_missing_trace_registry_rejects_before_provider_or_workspace_setup(self):
        from unittest.mock import patch
        schema = {'type': 'object', 'properties': {}, 'required': [], 'additionalProperties': False}
        with patch.object(runner, 'trace_vocabulary', side_effect=FileNotFoundError('controlled registry')), \
             patch.object(runner.subprocess, 'Popen') as process:
            with self.assertRaises(FileNotFoundError):
                runner.run_agent({'fixture': {'answer_schema': schema}}, None, None, None)
            process.assert_not_called()

    def test_batch_prepare_rejects_archive_without_trace_registry(self):
        with tempfile.TemporaryDirectory() as directory:
            archive = Path(directory)
            for name in ('run_read_batch.py', 'codex_runner.py', 'tasks.scaling.json'):
                target = archive / 'evaluation' / name
                target.parent.mkdir(exist_ok=True)
                target.write_bytes((ROOT / 'evaluation' / name).read_bytes())
            args = [sys.executable, str(archive / 'evaluation/run_read_batch.py'),
                    '--prepare-only', '--task-kind', 'scaling']
            for key in ('official-source', 'official-cli', 'next-source', 'next-cli', 'codex', 'auth-home', 'output-root'):
                args.extend(['--' + key, str(archive / 'not-created')])
            result = subprocess.run(args, capture_output=True, text=True, encoding='utf-8')
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('registrations.json.gz', result.stderr)
            self.assertFalse((archive / 'not-created').exists())

    def test_batch_pins_luna_medium_in_shared_codex_config(self):
        from unittest.mock import patch
        spec = importlib.util.spec_from_file_location('run_read_batch', ROOT / 'evaluation/run_read_batch.py')
        batch = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(batch)
        captured = []
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source = root / 'official-source'
            entry = source / 'skills/lark-contact/SKILL.md'
            entry.parent.mkdir(parents=True)
            entry.write_text('Controlled official guidance')
            official, next_cli = root / 'official-cli', root / 'next-cli'
            official.touch()
            next_cli.touch()
            args = ['run_read_batch.py', '--section', 'contact', '--task-kind', 'paired',
                    '--model', 'gpt-5.6-luna', '--reasoning', 'medium']
            values = {'official-source': source, 'official-cli': official,
                      'next-source': ROOT, 'next-cli': next_cli, 'codex': root / 'codex',
                      'auth-home': root / 'unused-auth', 'output-root': root / 'results'}
            for name, path in values.items():
                args.extend(['--' + name, str(path)])

            def inspect_config(command, **kwargs):
                self.assertEqual(Path(command[1]).name, 'harness.py')
                config = json.loads(Path(command[command.index('--config') + 1]).read_text())
                captured.append(config)
                self.assertEqual(config['model'], 'gpt-5.6-luna')
                self.assertEqual(config['reasoning'], 'medium')
                self.assertEqual(Path(config['arms']['official_concise']['skill']).read_text(),
                                 (ROOT / 'evaluation/official-contact-concise/SKILL.md').read_text())
                self.assertEqual(set(config['arms']), {'official', 'official_concise', 'next'})
                self.assertEqual(config['arms']['official']['cli'], config['arms']['official_concise']['cli'])
                self.assertEqual(Path(config['runner'][1]).name, 'codex_runner.py')
                self.assertEqual(config['runner'][-1], 'codex-cli 0.153.4')
                return subprocess.CompletedProcess(command, 0)

            with patch.object(sys, 'argv', args), \
                 patch.object(sys, 'path', [str(ROOT / 'evaluation'), *sys.path]), \
                 patch.object(batch.subprocess, 'run', side_effect=inspect_config), \
                 patch('builtins.print'):
                self.assertEqual(batch.main(), 0)
            self.assertEqual(len(captured), 1)
            self.assertFalse(values['auth-home'].exists())

    def test_original_comparison_and_paired_markdown_keep_full_drive_source(self):
        from unittest.mock import patch
        spec = importlib.util.spec_from_file_location('run_read_batch', ROOT / 'evaluation/run_read_batch.py')
        batch = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(batch)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'cli'
            binary.touch()
            args = ['run_read_batch.py', '--section', 'markdown', '--task-kind', 'paired',
                    '--model', 'gpt-5.6-luna', '--reasoning', 'medium']
            for name, path in {'official-source': ROOT, 'official-cli': binary,
                'next-source': ROOT, 'next-cli': binary, 'codex': binary,
                'auth-home': root / 'auth', 'output-root': root / 'results'}.items():
                args.extend(['--' + name, str(path)])
            configs = []

            def inspect_config(command, **kwargs):
                config = json.loads(Path(command[command.index('--config') + 1]).read_text())
                configs.append(config)
                self.assertEqual(Path(config['arms']['official']['skill']), ROOT / 'skills/lark-drive/SKILL.md')
                self.assertEqual(Path(config['arms']['official']['skill']).read_bytes(),
                                 (ROOT / 'skills/lark-drive/SKILL.md').read_bytes())
                self.assertEqual(Path(config['arms']['next']['skill']), ROOT / 'agent-skills/feishu-cli-next/SKILL.md')
                if '--original-comparison' in command:
                    self.assertEqual(set(config['arms']), {'official', 'next'})
                else:
                    self.assertEqual(set(config['arms']), {'official', 'official_concise', 'next'})
                    self.assertEqual(Path(config['arms']['official_concise']['skill']).read_bytes(),
                                     (ROOT / 'evaluation/concise/markdown/SKILL.md').read_bytes())
                return subprocess.CompletedProcess(command, 0)

            for extra, attempts in (([], 9), (['--original-comparison'], 6)):
                with patch.object(sys, 'argv', [*args, *extra]), \
                     patch.object(sys, 'path', [str(ROOT / 'evaluation'), *sys.path]), \
                     patch.object(batch.subprocess, 'run', side_effect=inspect_config) as run, \
                     patch('builtins.print') as printed:
                    self.assertEqual(batch.main(), 0)
                self.assertEqual('--original-comparison' in run.call_args.args[0], bool(extra))
                self.assertEqual(json.loads(printed.call_args_list[0].args[0])['planned_attempts'], attempts)
            expected = {**configs[0], 'arms': {k: v for k, v in configs[0]['arms'].items() if k != 'official_concise'}}
            self.assertEqual(configs[1], expected)
            self.assertFalse((root / 'auth').exists())
            for conflicting in (['--previous-next-source', str(ROOT)],
                                ['--previous-next-cli', str(binary)], ['--only-arm', 'next']):
                with patch.object(sys, 'argv', [*args, '--original-comparison', *conflicting]), \
                     patch.object(batch.subprocess, 'run') as run, patch('sys.stderr'), self.assertRaises(SystemExit):
                    batch.main()
                run.assert_not_called()

    def test_batch_before_after_uses_frozen_next_without_mislabeling_concise(self):
        from unittest.mock import patch
        spec = importlib.util.spec_from_file_location('run_read_batch', ROOT / 'evaluation/run_read_batch.py')
        batch = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(batch)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            binary = root / 'cli'
            binary.touch()
            args = ['run_read_batch.py', '--section', 'docs', '--task-kind', 'paired',
                    '--model', 'gpt-5.6-luna', '--reasoning', 'medium']
            for name, path in {
                'official-source': ROOT, 'official-cli': binary,
                'next-source': ROOT, 'next-cli': binary,
                'previous-next-source': ROOT, 'previous-next-cli': binary,
                'codex': binary, 'auth-home': root / 'auth', 'output-root': root / 'results',
            }.items():
                args.extend(['--' + name, str(path)])
            configs = []

            def inspect_config(command, **kwargs):
                config = json.loads(Path(command[command.index('--config') + 1]).read_text())
                configs.append(config)
                self.assertEqual(set(config['arms']), {'official', 'next_before', 'next'})
                self.assertEqual(config['arms']['next_before'], {
                    'cli': str(binary), 'skill': str(ROOT / 'agent-skills/feishu-cli-next/SKILL.md')})
                self.assertEqual(config['model'], 'gpt-5.6-luna')
                self.assertEqual(config['reasoning'], 'medium')
                return subprocess.CompletedProcess(command, 0)

            with patch.object(sys, 'argv', args), \
                 patch.object(sys, 'path', [str(ROOT / 'evaluation'), *sys.path]), \
                 patch.object(batch.subprocess, 'run', side_effect=inspect_config), patch('builtins.print'):
                self.assertEqual(batch.main(), 0)
            self.assertEqual(len(configs), 1)
            with patch.object(sys, 'argv', [*args, '--only-arm', 'next']), \
                 patch.object(sys, 'path', [str(ROOT / 'evaluation'), *sys.path]), \
                 patch.object(batch.subprocess, 'run', side_effect=inspect_config) as run, \
                 patch('builtins.print') as printed:
                self.assertEqual(batch.main(), 0)
            self.assertEqual(configs[1], configs[0])
            command = run.call_args.args[0]
            self.assertEqual(command[-2:], ['--only-arm', 'next'])
            starting = json.loads(printed.call_args_list[0].args[0])
            self.assertEqual(starting, {'section': 'docs', 'status': 'starting', 'planned_attempts': 6,
                                       'only_arm': 'next', 'balanced_comparison': False})
            index = args.index('--previous-next-cli')
            missing = args[:index] + args[index + 2:]
            with patch.object(sys, 'argv', missing), patch.object(batch.subprocess, 'run') as run, \
                 patch('sys.stderr'), self.assertRaises(SystemExit):
                batch.main()
            run.assert_not_called()

    def test_trace_keeps_only_public_names_not_content_paths_or_unknown_flags(self):
        self.assertEqual(runner.command_choices('feishu-cli-next sheets +data-export --spreadsheet-token private --save-result private.json'),
                         [{'kind': 'cli', 'command': ['sheets', '+data-export'], 'flags': ['--save-result', '--spreadsheet-token']}])
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

    def test_accounting_reasons_are_fixed_codes_without_provider_content(self):
        raw = self.stream([{"type": "turn.started"}, {"type": "turn.failed", "error": "private provider details"},
                           {"type": "item.completed", "item": {"type": "mcp_tool_call", "name": "private tool"}}])
        result = runner.parse_events(raw + "\nprivate non-JSON error", 0)
        self.assertEqual(set(result["accounting_diagnostics"]), {
            "provider_event_invalid", "provider_turn_failed", "provider_turn_count_mismatch", "unexpected_provider_tool"})
        self.assertFalse(result["accounting_complete"])
        self.assertEqual(result["turns"][0]["input_tokens"], 200)
        self.assertNotIn("private", json.dumps(result))

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
assert sys.argv[sys.argv.index('--model')+1] == 'gpt-5.6-luna'
assert 'model_reasoning_effort="medium"' in sys.argv
assert 'LARKSUITE_CLI_APP_SECRET' not in os.environ
assert os.environ['LARKSUITE_CLI_REMOTE_META'] == 'off'
assert 'Synthetic test guidance' in sys.stdin.read()
pathlib.Path(sys.argv[sys.argv.index('--output-last-message')+1]).write_text('{"configured":false}')
print(json.dumps({'type':'turn.started'}))
print(json.dumps({'type':'turn.completed','usage':{'input_tokens':200,'cached_input_tokens':80,'output_tokens':30}}))
''')
            request = {"workspace": str(workspace), "cli": str(fake), "skill": str(guidance / "SKILL.md"),
                       "prompt": "Inspect the synthetic config.", "settings": {"backend": "offline-foundation-v1",
                       "model": "gpt-5.6-luna", "tokenizer": "provider-undisclosed", "reasoning": "medium",
                       "budget": {"timeout_seconds": 5}}, "fixture": {"answer_schema": {"type": "object", "properties": {}, "required": [], "additionalProperties": False}}}
            process = subprocess.run([sys.executable, str(ROOT / "tests/next_evaluation/python_fixture_entry.py"),
                                      "--entry", str(ROOT / "evaluation/codex_runner.py"), "--fixture", str(fake), "--",
                                      "--codex", str(fake), "--auth-home", str(root / "auth")],
                                     input=json.dumps(request), capture_output=True, text=True, timeout=15)
            self.assertEqual(process.returncode, 0, process.stderr)
            result = json.loads(process.stdout)
            self.assertTrue(result["accounting_complete"])
            self.assertEqual(result['model'], 'gpt-5.6-luna')
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
