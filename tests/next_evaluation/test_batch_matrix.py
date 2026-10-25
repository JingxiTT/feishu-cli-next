# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Launcher contract checks: no provider, native CLI, or host compiler execution."""
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
import codex_runner
import harness
import scenarios.extensions

SPEC = importlib.util.spec_from_file_location('matrix_batch_launcher', ROOT / 'evaluation/run_read_batch.py')
batch = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(batch)
MANIFESTS = {'read': 'read-sections', 'scenario': 'scenarios', 'scaling': 'scaling',
             'business': 'business', 'foundation': 'foundation', 'contact': 'contact',
             'paired': 'paired', 'docs-hard': 'docs-hard', 'e2e': 'e2e'}


class BatchMatrixTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / 'official'
        sections = {t['section'] for name in MANIFESTS.values() for t in
                    json.loads((ROOT / ('evaluation/tasks.' + name + '.json')).read_text())['tasks']}
        for section in sections | {'drive'}:
            if section == 'extensions':
                continue
            name = {'docs': 'doc', 'events': 'event', 'packaging': 'shared',
                    'business': 'shared', 'foundation': 'shared'}.get(section, section)
            self.write(self.source / ('skills/lark-' + name + '/SKILL.md'), 'Full original ' + name)
        for name in ('extension/README.md', 'extension/platform/README.md',
                     'extension/command/examples/chat-brief/main.go'):
            self.write(self.source / name, 'Original ' + name)
        self.next_source = self.root / 'next'
        self.write(self.next_source / 'agent-skills/feishu-cli-next/SKILL.md', 'Next guide')
        for source in (self.source, self.next_source):
            self.write(source / 'go.mod', 'module synthetic')
            self.write(source / 'go.sum', '')
            self.write(source / 'main.go', 'package main')
            self.write(source / 'baseline/registrations.json.gz', 'catalog')
        self.official = self.root / 'official-cli'
        self.next_cli = self.root / 'next-cli'
        self.official.touch()
        self.next_cli.touch()
        self.output = self.root / 'output'
        self.host_builds = []

    def write(self, path, text):
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def argv(self, kind, *extra):
        args = ['run_read_batch.py', '--task-kind', kind]
        for key, value in {'official-source': self.source, 'next-source': self.next_source,
                           'official-cli': self.official, 'next-cli': self.next_cli,
                           'codex': self.root / 'codex', 'auth-home': self.root / 'auth',
                           'output-root': self.output}.items():
            args.extend(['--' + key, str(value)])
        return args + list(extra)

    def build(self, source, output, next_mode):
        self.host_builds.append((source, next_mode))
        self.write(output, 'host-' + str(next_mode))
        return output

    def invoke(self, args, prepare=False):
        configs, commands, skill_text = [], [], []
        original_validate = harness.validate_config

        def validate(config):
            original_validate(config)
            configs.append(config)
            skill_text.append({a: Path(v['skill']).read_text() for a, v in config['arms'].items()})

        def launch(command, **kwargs):
            self.assertEqual(Path(command[1]).name, 'harness.py')
            commands.append(command)
            return subprocess.CompletedProcess(command, 0)

        with patch.object(sys, 'argv', args), patch.object(harness, 'validate_config', side_effect=validate), \
             patch.object(scenarios.extensions, 'build_host', side_effect=self.build), \
             patch.object(batch.subprocess, 'run', side_effect=launch), \
             patch.object(codex_runner, 'validate_output_schema', wraps=codex_runner.validate_output_schema) as schema, \
             patch.object(codex_runner.subprocess, 'Popen') as provider, contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(batch.main(), 0)
            provider.assert_not_called()
            self.assertGreater(schema.call_count, 0)
        if prepare:
            self.assertEqual(commands, [])
        return configs, commands, skill_text

    def test_interrupted_harness_stops_before_later_sections(self):
        args = self.argv('read', '--original-comparison', '--resume')
        with patch.object(sys, 'argv', args), contextlib.redirect_stdout(io.StringIO()), \
             patch.object(batch.subprocess, 'run', return_value=subprocess.CompletedProcess([], 2)) as launch:
            self.assertEqual(batch.main(), 2)
            self.assertEqual(launch.call_count, 1)

    def test_all_suites_prepare_original_two_arms_with_correct_contracts(self):
        for kind, manifest_name in MANIFESTS.items():
            with self.subTest(kind=kind):
                manifest = json.loads((ROOT / ('evaluation/tasks.' + manifest_name + '.json')).read_text())
                configs, _, _ = self.invoke(self.argv(kind, '--original-comparison', '--prepare-only',
                    '--model', 'gpt-5.6-luna', '--reasoning', 'medium', '--guidance-delivery', 'file'), prepare=True)
                self.assertEqual(len(configs), len({t['section'] for t in manifest['tasks']}))
                for config in configs:
                    self.assertEqual(set(config['arms']), {'official', 'next'})
                    self.assertEqual(config['backend'], manifest['backend'])
                    self.assertEqual((config['model'], config['reasoning'], config['guidance_delivery']),
                                     ('gpt-5.6-luna', 'medium', 'file'))
                    verifier = {'foundation': 'foundation_verifier.py', 'contact': 'contact_verifier.py',
                                'read': 'read_verifier.py'}.get(kind, 'scenario_verifier.py')
                    self.assertEqual(Path(config['verifier'][1]).name, verifier)
                if kind == 'foundation':
                    self.assertIn('network disabled', configs[0]['permissions'])
                    self.assertEqual(Path(configs[0]['arms']['official']['skill']).parent.name, 'lark-shared')
                if kind == 'e2e':
                    self.assertEqual(Path(configs[0]['arms']['official']['skill']).parent.name, 'lark-shared')
                if kind == 'scenario':
                    packaging = next(c for c in configs if 'selected_skill_source' in c['arms']['official'])
                    self.assertEqual(packaging['arms']['official']['selected_skill_source'], str(self.source / 'skills/lark-shared'))
                    self.assertEqual(packaging['arms']['next']['selected_skill_source'], str(self.next_source / 'agent-skills/feishu-cli-next'))
        self.assertEqual(len(self.host_builds), 2)

    def test_all_suites_forward_original_schedule_and_resume(self):
        for kind, manifest_name in MANIFESTS.items():
            with self.subTest(kind=kind):
                configs, commands, _ = self.invoke(self.argv(kind, '--original-comparison', '--resume'))
                self.assertEqual(len(commands), len(configs))
                for command in commands:
                    self.assertIn('--original-comparison', command)
                    self.assertIn('--resume', command)
                    self.assertEqual(Path(command[command.index('--tasks') + 1]).name,
                                     'tasks.' + manifest_name + '.json')
                    self.assertEqual(command[command.index('--repeats') + 1], '3')
                    if kind in ('foundation', 'contact'):
                        self.assertEqual(command[command.index('--seed') + 1], '20260907')

    def test_foundation_contact_and_packaging_default_concise_sources(self):
        for kind, section, concise in [('foundation', 'foundation', 'official-concise'),
                                       ('contact', 'contact', 'official-contact-concise'),
                                       ('scenario', 'packaging', 'concise/packaging')]:
            with self.subTest(kind=kind):
                configs, _, texts = self.invoke(self.argv(kind, '--section', section, '--prepare-only'), prepare=True)
                self.assertEqual(set(configs[0]['arms']), {'official', 'official_concise', 'next'})
                self.assertEqual(texts[0]['official_concise'], (ROOT / 'evaluation' / concise / 'SKILL.md').read_text())
                self.assertEqual((configs[0]['model'], configs[0]['reasoning']), ('gpt-6-astra', 'high'))

    def test_progress_forwarding_is_opt_in(self):
        _, commands, _ = self.invoke(self.argv('contact', '--original-comparison', '--progress'))
        self.assertIn('--progress', commands[0])
        _, commands, _ = self.invoke(self.argv('contact', '--original-comparison'))
        self.assertNotIn('--progress', commands[0])

    def test_resume_forwards_without_overwriting_or_reading_checkpoint(self):
        checkpoint = self.output / 'contact/batch-01.json'
        self.write(checkpoint, 'existing evidence belongs to harness validation')
        configs, commands, _ = self.invoke(self.argv('contact', '--original-comparison', '--resume'))
        self.assertEqual(len(commands), 1)
        self.assertIn('--resume', commands[0])
        self.assertIn('--original-comparison', commands[0])
        self.assertEqual(checkpoint.read_text(), 'existing evidence belongs to harness validation')
        with self.assertRaisesRegex(ValueError, 'refusing to overwrite'):
            self.invoke(self.argv('contact', '--original-comparison'))

    def test_extension_hosts_stable_across_resume_and_reject_tamper(self):
        args = self.argv('scenario', '--section', 'extensions', '--original-comparison', '--prepare-only', '--resume')
        first, _, _ = self.invoke(args, prepare=True)
        second, _, _ = self.invoke(args, prepare=True)
        self.assertEqual({a: v['cli'] for a, v in first[0]['arms'].items()},
                         {a: v['cli'] for a, v in second[0]['arms'].items()})
        self.assertEqual(len(self.host_builds), 2)
        host = Path(first[0]['arms']['official']['cli'])
        host.write_text('tampered')
        with self.assertRaisesRegex(ValueError, 'cached extension host'):
            self.invoke(args, prepare=True)

    def test_extension_python_cache_does_not_change_host_identity(self):
        with patch.object(scenarios.extensions, 'build_host', side_effect=self.build):
            first = batch.extension_host(self.source, self.output, False)
            self.write(self.source / 'evaluation/__pycache__/helper.cpython-314.pyc', 'generated bytecode')
            second = batch.extension_host(self.source, self.output, False)
        self.assertEqual(first, second)
        self.assertEqual(len(self.host_builds), 1)

    def test_extension_source_change_uses_new_cache(self):
        with patch.object(scenarios.extensions, 'build_host', side_effect=self.build):
            first = batch.extension_host(self.source, self.output, False)
            self.write(self.source / 'main.go', 'changed source')
            second = batch.extension_host(self.source, self.output, False)
        self.assertNotEqual(first, second)
        self.assertTrue(first.is_file())
        self.assertEqual(len(self.host_builds), 2)

    def test_all_selected_schemas_preflight_before_any_section_launch(self):
        archive = self.root / 'bad-archive'
        manifest = json.loads((ROOT / 'evaluation/tasks.foundation.json').read_text())
        manifest['tasks'][-1]['fixture']['answer_schema']['additionalProperties'] = True
        self.write(archive / 'evaluation/tasks.foundation.json', json.dumps(manifest))
        with patch.object(batch, 'ROOT', archive), patch.object(sys, 'argv', self.argv('foundation', '--original-comparison')), \
             patch.object(batch.subprocess, 'run') as launch, patch.object(harness, 'validate_config') as config, \
             patch.object(codex_runner.subprocess, 'Popen') as provider:
            with self.assertRaisesRegex(ValueError, 'closed'):
                batch.main()
            launch.assert_not_called()
            config.assert_not_called()
            provider.assert_not_called()


if __name__ == '__main__':
    unittest.main()
