# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

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
    def test_recipient_rule_defaults_to_no_delivery_and_uses_only_verified_stable_identity(self):
        self.assertEqual(runner.recipient_rule({}),
                         'Do not deliver content or grant access to any other account. ')
        recipient = {'open_id': 'test-recipient', 'tenant_key': 'test-tenant'}
        rule = runner.recipient_rule({'authorized_recipient': recipient})
        self.assertIn(json.dumps(recipient), rule)
        self.assertIn('only when explicitly authorized by the task', rule)
        self.assertIn('groups containing other accounts', rule)
        for value in ({}, [], {**recipient, 'name': 'unnecessary'}, {**recipient, 'open_id': ''}):
            with self.subTest(value=value), self.assertRaises(ValueError):
                runner.recipient_rule({'authorized_recipient': value})

    def error_stream(self, output, command="lark-cli docs +push --file private.md --json", exit_code=2):
        return "\n".join(json.dumps(event) for event in [
            {"type": "turn.started"},
            {"type": "item.completed", "item": {"type": "command_execution", "command": command,
                "exit_code": exit_code, "aggregated_output": output}},
            {"type": "turn.completed", "usage": {"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}}])

    def test_cli_diagnostics_are_finite_and_do_not_change_accounting(self):
        for message, reason in runner.PUSH_ERROR_REASONS.items():
            output = json.dumps({"ok": False, "identity": "private identity", "error": {
                "type": "validation", "subtype": "failed_precondition", "message": message,
                "param": "--file", "hint": "private hint", "code": 987654321,
                "log_id": "private id", "cause": {"secret": "private cause"}}}, indent=2)
            result = runner.parse_events(self.error_stream(output), 0)
            baseline = runner.parse_events(self.error_stream(""), 0)
            self.assertEqual({k: v for k, v in result.items() if k != "control_trace"},
                             {k: v for k, v in baseline.items() if k != "control_trace"})
            diagnostic = result["control_trace"][0].pop("reported_cli_error")
            self.assertEqual(diagnostic, {"type": "validation", "subtype": "failed_precondition",
                                         "param": "--file", "reason": reason})
            self.assertEqual(result, baseline)
            self.assertNotIn("private", json.dumps(diagnostic))

    def test_mutation_error_keeps_one_finite_cause_without_receipts(self):
        output = json.dumps({"ok": False, "data": {"private": "partial output"}}) + "\n" + json.dumps({
            "ok": False, "error": {"type": "api", "subtype": "mutation_outcome", "outcome": "unknown",
                "receipts": ["private receipt"], "persistence_failed": True,
                "cause": {"type": "validation", "subtype": "failed_precondition",
                    "message": "native readback does not match the planned block edit", "param": "private path",
                    "cause": {"type": "internal", "subtype": "file_io", "message": "private"}}}})
        result = runner.parse_events(self.error_stream(output, exit_code=22), 0)
        self.assertEqual(result["control_trace"][0]["reported_cli_error"], {
            "type": "api", "subtype": "mutation_outcome", "outcome": "unknown", "cause": {
                "type": "validation", "subtype": "failed_precondition", "reason": "workspace_readback_mismatch"}})
        self.assertTrue(result["accounting_complete"])

    def test_diagnostic_parser_rejects_adversarial_output_without_losing_usage(self):
        error = {"ok": False, "error": {"type": "validation", "subtype": "failed_precondition"}}
        encoded = json.dumps(error)
        outputs = [None, [], "", "private text " + encoded, encoded + " private text", encoded[:-1],
                   encoded + encoded, encoded + "{}{}", " " * 65537 + encoded, "[" * 1500 + "]" * 1500,
                   '{"ok":false,"ok":false,"error":{"type":"validation","subtype":"failed_precondition"}}',
                   '{"ok":false,"error":{"type":"validation","subtype":"failed_precondition","code":NaN}}',
                   json.dumps({"ok": True, "error": error["error"]}),
                   json.dumps({"ok": False, "error": {"type": ["validation"], "subtype": {}}}),
                   json.dumps({"ok": False, "error": {"type": "private type", "subtype": "private subtype"}})]
        for output in outputs:
            with self.subTest(output_type=type(output).__name__):
                result = runner.parse_events(self.error_stream(output), 0)
                self.assertNotIn("reported_cli_error", result["control_trace"][0])
                self.assertTrue(result["accounting_complete"])
                self.assertEqual(result["turns"], [{"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}])
        for command, code in [("lark-cli docs +push --file private.md; cat private.txt", 2),
                              ("python3 private.py", 2), ("lark-cli docs +push --file private.md", 0),
                              ("lark-cli docs +push --file private.md", True)]:
            result = runner.parse_events(self.error_stream(encoded, command, code), 0)
            self.assertNotIn("reported_cli_error", result["control_trace"][0])

    def test_message_labels_require_exact_source_literal_and_push_command(self):
        for command, message in [("lark-cli docs +pull", "local document edit is unsupported or ambiguous"),
                                 ("lark-cli docs +push", "local document edit is unsupported or ambiguous private"),
                                 ("lark-cli docs +push", "workspace_edit_unsupported")]:
            output = json.dumps({"ok": False, "error": {"type": "validation", "subtype": "failed_precondition",
                                "message": message, "reason": "workspace_edit_unsupported"}})
            result = runner.parse_events(self.error_stream(output, command), 0)
            self.assertEqual(result["control_trace"][0]["reported_cli_error"],
                             {"type": "validation", "subtype": "failed_precondition"})

    def stream(self, extra=()):
        return "\n".join(json.dumps(event) for event in [
            {"type": "thread.started", "thread_id": "synthetic"},
            {"type": "turn.started"},
            {"type": "item.completed", "item": {"type": "command_execution"}},
            {"type": "turn.completed", "usage": {"input_tokens": 200, "cached_input_tokens": 80, "output_tokens": 30}},
            *extra])

    def test_all_executable_manifests_have_provider_compatible_object_schemas(self):
        manifest = json.loads((ROOT / 'evaluation/tasks.live-contact.json').read_text())
        for task in manifest['tasks']:
            runner.validate_output_schema(task['fixture']['answer_schema'])

    def test_removed_backends_are_rejected_before_any_model_process(self):
        from unittest.mock import patch
        for backend in ('synthetic-contact-v1', 'synthetic-read-v1', 'synthetic-scenario-v1', 'offline-foundation-v1'):
            with self.subTest(backend=backend), patch.object(runner.subprocess, 'Popen') as process:
                with self.assertRaises(ValueError):
                    runner.run({'settings': {'backend': backend}}, None, None, None)
                process.assert_not_called()

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

    def test_file_guidance_preserves_both_arm_files_and_inline_prompt(self):
        from unittest.mock import patch
        schema = {'type': 'object', 'properties': {}, 'required': [], 'additionalProperties': False}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for arm in ('official', 'next'):
                distribution = root / arm / ('skills' if arm == 'official' else 'agent-skills')
                entry = distribution / ('lark-doc' if arm == 'official' else 'feishu-cli-next') / 'SKILL.md'
                entry.parent.mkdir(parents=True)
                content = ('Complete ' + arm + ' guidance — 你好\nRead references/details.md when needed.\n').encode()
                entry.write_bytes(content)
                reference = entry.parent / 'references/details.md'
                reference.parent.mkdir()
                reference.write_bytes(b'Exact reference bytes\r\n')
                if arm == 'official':
                    shared = distribution / 'lark-shared/SKILL.md'
                    shared.parent.mkdir()
                    shared.write_bytes(b'Original cross-skill dependency\n')
                prompts, copied = {}, {}
                for mode in ('default', 'inline', 'file'):
                    workspace = root / (arm + '-' + mode)
                    workspace.mkdir()
                    settings = {'model': 'gpt-5.6-luna', 'tokenizer': 'provider-undisclosed',
                                'reasoning': 'medium', 'budget': {'timeout_seconds': 5}}
                    if mode != 'default':
                        settings['guidance_delivery'] = mode
                    request = {'workspace': str(workspace), 'cli': str(root / 'unused-cli'), 'skill': str(entry),
                               'prompt': 'Perform the synthetic task.', 'settings': settings,
                               'fixture': {'answer_schema': schema, 'cli_config_dir': str(root), 'cli_data_dir': str(root)}, 'arm': arm}
                    with patch.object(runner.subprocess, 'run', return_value=subprocess.CompletedProcess(
                            [], 0, stdout='codex-cli test\n')), patch.object(runner.subprocess, 'Popen') as process:
                        process.return_value.returncode = 0
                        process.return_value.communicate.return_value = (self.stream(), '')
                        result, code = runner.run_agent(request, root / 'unused-codex', root / 'empty-auth', 'codex-cli test')
                        prompts[mode] = process.return_value.communicate.call_args.args[0]
                    self.assertEqual(code, 0)
                    self.assertTrue(result['accounting_complete'])
                    self.assertEqual(entry.read_bytes(), content)
                    self.assertEqual((workspace / 'guidance/SKILL.md').read_bytes(), content)
                    self.assertEqual((workspace / 'guidance/references/details.md').read_bytes(), reference.read_bytes())
                    copied[mode] = {str(path.relative_to(workspace)): path.read_bytes()
                                    for path in workspace.rglob('*') if path.is_file() and
                                    path.relative_to(workspace).parts[0] in ('guidance', 'lark-doc', 'lark-shared')}
                    if arm == 'official':
                        self.assertEqual((workspace / 'lark-shared/SKILL.md').read_bytes(), shared.read_bytes())
                self.assertEqual(prompts['default'], prompts['inline'])
                self.assertEqual(copied['default'], copied['file'])
                self.assertEqual(prompts['inline'].count(content.decode()), 1)
                self.assertNotIn(content.decode(), prompts['file'])
                self.assertIn('./guidance/SKILL.md. Read it completely before task actions', prompts['file'])
                self.assertTrue(prompts['file'].endswith('\n\nTask:\nPerform the synthetic task.'))
                self.assertIn('This task uses real Feishu', prompts['inline'])
                self.assertNotIn('synthetic read-only fixture', prompts['inline'])
                self.assertIn('unknown outcome', prompts['inline'])
                self.assertIn(runner.recipient_rule({}), prompts['inline'])

    def test_invalid_guidance_delivery_fails_before_workspace_or_provider(self):
        from unittest.mock import patch
        schema = {'type': 'object', 'properties': {}, 'required': [], 'additionalProperties': False}
        for mode in (None, True, [], {}, 'private-invalid-mode'):
            with self.subTest(mode=mode), patch.object(runner.subprocess, 'Popen') as process:
                with self.assertRaises(ValueError):
                    runner.run_agent({'fixture': {'answer_schema': schema}, 'settings': {'guidance_delivery': mode}},
                                     None, None, None)
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

    def test_docs_workspace_trace_retains_commands_without_resource_or_file_values(self):
        command = 'feishu-cli-next docs +pull --doc secret-id --output ./private.md && feishu-cli-next docs +push --file ./private.md'
        result = runner.command_choices(command)
        self.assertEqual(result, [
            {'kind': 'cli', 'command': ['docs', '+pull'], 'flags': ['--doc', '--output']},
            {'kind': 'cli', 'command': ['docs', '+push'], 'flags': ['--file']}])
        self.assertNotIn('secret', json.dumps(result))
        self.assertNotIn('private', json.dumps(result))

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
            config = runner.configuration(root, root, "high", {
                "LARKSUITE_CLI_CONFIG_DIR": str(root), "LARKSUITE_CLI_DATA_DIR": str(root)})
            self.assertIn('model_reasoning_effort="high"', config)
            self.assertIn('sandbox_workspace_write.network_access=true', config)
            self.assertIn('features.plugins=false', config)
            self.assertIn('features.multi_agent=false', config)
            self.assertIn('features.apps=false', config)
            self.assertIn('project_doc_max_bytes=0', config)
            self.assertEqual(next(value for value in config if value.startswith('skills.config=')),
                             'skills.config=[{path=' + json.dumps(str(skill)) + ',enabled=false}]')
            self.assertNotIn('DO NOT INJECT THIS', '\n'.join(config))

    def test_live_configuration_uses_real_cli_credentials_and_network(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            config_dir = root / 'protected-cli'
            config_dir.mkdir()
            settings = runner.configuration(root, root, 'high',
                {'LARKSUITE_CLI_CONFIG_DIR': str(config_dir), 'LARKSUITE_CLI_DATA_DIR': str(config_dir)})
            self.assertIn('sandbox_workspace_write.network_access=true', settings)
            self.assertIn('sandbox_workspace_write.writable_roots=' + json.dumps([str(config_dir), str(config_dir)]), settings)
            self.assertIn(str(config_dir), '\n'.join(settings))
            self.assertNotIn('127.0.0.1', '\n'.join(settings))
            self.assertNotIn('LARKSUITE_CLI_AUTH_PROXY', '\n'.join(settings))

    @unittest.skipUnless(sys.platform == 'darwin' and os.environ.get('EVAL_NATIVE_CODEX'),
                         'set EVAL_NATIVE_CODEX on macOS for no-model native skill isolation')
    def test_native_skill_entry_file_disables_catalog_without_model_or_auth(self):
        with tempfile.TemporaryDirectory(prefix='feishu-skill-isolation-') as directory:
            root = Path(directory)
            auth_home, workspace = root / 'codex-home', root / 'workspace'
            for path in (auth_home, workspace, workspace / 'home', workspace / 'tmp'):
                path.mkdir()
            skill = auth_home / 'skills' / 'isolation-probe-sentinel' / 'SKILL.md'
            skill.parent.mkdir(parents=True)
            skill.write_text('---\nname: isolation-probe-sentinel\ndescription: Synthetic no-action probe.\n---\nSynthetic only.\n')
            env = {**runner.cli_environment(workspace), 'CODEX_HOME': str(auth_home)}

            def prompt(disable_directory):
                # Offline configuration avoids opening managed proxy listeners;
                # the skill path/feature overrides are shared with paired runs.
                settings = runner.configuration(auth_home, workspace, 'medium', {
                    'LARKSUITE_CLI_CONFIG_DIR': str(workspace), 'LARKSUITE_CLI_DATA_DIR': str(workspace)})
                if disable_directory:
                    # Negative control reproduces the old ineffective path shape.
                    settings = [value if not value.startswith('skills.config=') else
                                'skills.config=[{path=' + json.dumps(str(skill.parent)) + ',enabled=false}]'
                                for value in settings]
                command = ['/usr/bin/sandbox-exec', '-p', '(version 1) (allow default) (deny network*)',
                           os.environ['EVAL_NATIVE_CODEX'], '-C', str(workspace),
                           'debug', 'prompt-input', 'Synthetic isolation probe.']
                for value in settings:
                    command += ['-c', value]
                result = subprocess.run(command, cwd=workspace, env=env, capture_output=True, text=True, timeout=20)
                self.assertEqual(result.returncode, 0, 'native prompt-input failed')
                self.assertIsInstance(json.loads(result.stdout), list)
                return result.stdout

            visible = prompt(True)
            self.assertIn('isolation-probe-sentinel', visible)
            self.assertIn('skills_instructions', visible)
            # Re-enumeration also covers system entry files bootstrapped by Codex.
            isolated = prompt(False)
            self.assertNotIn('isolation-probe-sentinel', isolated)
            self.assertNotIn('skills_instructions', isolated)
            self.assertFalse((auth_home / 'auth.json').exists())


class CodexAdapterProcessTests(unittest.TestCase):

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
                       "prompt": "Inspect the current account.", "settings": {"backend": "live-feishu-v1",
                       "model": "gpt-5.6-luna", "tokenizer": "provider-undisclosed", "reasoning": "medium",
                       "budget": {"timeout_seconds": 5}}, "fixture": {"cli_config_dir": str(root), "cli_data_dir": str(root), "answer_schema": {"type": "object", "properties": {}, "required": [], "additionalProperties": False}}}
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
