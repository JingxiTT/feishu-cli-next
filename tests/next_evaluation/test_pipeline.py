# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline matrix planning, accounting and complete custom-runner lifecycle."""
import copy
import csv
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
import pipeline
from harness import aggregate
from live_batch import batch_config


class PipelineTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.official = self.root / 'official'
        self.next = self.root / 'next'
        for path in (self.official / 'skills/lark-shared/SKILL.md',
                     self.next / 'agent-skills/feishu-cli-next/SKILL.md'):
            path.parent.mkdir(parents=True)
            path.write_text('Synthetic guide')
        self.cli = self.root / 'cli'
        self.cli.write_text('synthetic binary identity')
        self.runner = self.root / 'runner.py'
        self.runner.write_text('''import json, sys
from pathlib import Path
r = json.load(sys.stdin)
assert 'expected' not in r
with Path(__file__).with_suffix('.calls').open('a') as f:
    f.write(r['arm'] + '\\n')
print(json.dumps({'claimed_success': False, 'usage_source': 'provider',
 'model': r['settings']['model'], 'tokenizer': r['settings']['tokenizer'],
 'accounting_complete': True, 'turns': [
 {'input_tokens': 100, 'cached_input_tokens': 40, 'output_tokens': 20}]}))
''')
        self.adapter = self.root / 'adapter.json'
        self.settings = {'runner': [sys.executable, str(self.runner)], 'mode': 'real',
                         'accounting_method': 'synthetic test events, not model measurements',
                         'tokenizer': 'synthetic', 'tools': ['shell', 'files'],
                         'permissions': 'isolated synthetic test only',
                         'supported_backends': ['live-feishu-v1']}
        self.adapter.write_text(json.dumps(self.settings))
        self.config = {'schema_version': 1, 'official_source': str(self.official),
                       'next_source': str(self.next), 'official_cli': str(self.cli), 'next_cli': str(self.cli),
                       'repeats': 2, 'suites': ['foundation'], 'experiments': [
                           {'id': 'test-a', 'harness': 'test-adapter-v1', 'model': 'test-model',
                            'reasoning': 'none', 'runner_config': str(self.adapter)}]}
        self.driver = self.root / 'driver.py'
        self.driver.write_text('''import json, sys
from pathlib import Path
r = json.load(sys.stdin)
assert r['backend'] == 'live-feishu-v1'
with Path(__file__).with_suffix('.calls').open('a') as f:
    f.write(r['phase'] + '\\n')
if r['phase'] == 'setup':
    print(json.dumps({'backend': 'live-feishu-v1', 'ready': True,
      'account_verified': True,
      'privacy_verified': not Path(__file__).with_suffix('.fail-privacy').exists(),
      'prompt': r['task']['prompt']}))
elif r['phase'] == 'verify':
    print(json.dumps({'backend': 'live-feishu-v1', 'success': False}))
else:
    print(json.dumps({'cleaned': not Path(__file__).with_suffix('.fail-cleanup').exists()}))
''')
        credentials = self.root / 'protected-cli-config'
        credentials.mkdir()
        self.config['live'] = {'driver': [sys.executable, str(self.driver)],
                               'cli_config_dir': str(credentials), 'cli_data_dir': str(credentials)}
        self.config['live_manifests'] = {}
        sections = {'paired': ['docs', 'docs', 'contact'], 'docs-hard': ['docs'] * 3,
                    'read': ['wiki'], 'scenario': ['docs'], 'scaling': ['sheets'],
                    'business': ['business'], 'foundation': ['foundation'] * 2,
                    'contact': ['contact'], 'e2e': ['workflow']}
        for suite in pipeline.SUITES:
            # Mock lifecycle messages and token counters only; no Feishu API emulator.
            manifest = {'schema_version': 1, 'split': 'development', 'backend': 'live-feishu-v1',
                        'tasks': [{'id': f'{suite}-{i}', 'section': section, 'prompt': 'Unit-test goal',
                                   'fixture': {'answer_schema': {'type': 'object', 'properties': {},
                                               'required': [], 'additionalProperties': False}}, 'expected': {}}
                                  for i, section in enumerate(sections[suite])]}
            path = self.root / f'{suite}.json'
            path.write_text(json.dumps(manifest))
            self.config['live_manifests'][suite] = str(path)
        self.output = self.root / 'output'

    def test_plan_covers_all_configured_suites_without_reading_held_out(self):
        self.config.pop('suites')
        self.settings['supported_backends'] = ['live-feishu-v1']
        self.adapter.write_text(json.dumps(self.settings))
        plan = pipeline.prepare(self.config)
        self.assertEqual({j['suite'] for j in plan['jobs']}, set(pipeline.SUITES))
        self.assertEqual(sum(len(j['task_ids']) for j in plan['jobs'] if j['workload'] == 'diagnostic'), 8)
        self.assertEqual(sum(len(j['task_ids']) for j in plan['jobs'] if j['workload'] == 'workflow'), 6)
        self.assertEqual(plan['workloads'], ['workflow', 'diagnostic'])
        self.assertEqual(plan['planned_attempts'], 14 * 2 * 2)
        # Quarantined tasks are excluded from product identity too.
        held = self.next / 'evaluation/tasks.held-out.json'
        held.parent.mkdir()
        held.write_text('do not consume')
        self.assertEqual(pipeline.prepare(self.config), plan)
        held.write_text('still quarantined')
        self.assertEqual(pipeline.prepare(self.config), plan)

    def test_changed_runner_or_model_changes_identity(self):
        original = pipeline.prepare(self.config)
        self.runner.write_text(self.runner.read_text() + '\n# changed adapter\n')
        self.assertNotEqual(pipeline.prepare(self.config)['identity_sha256'], original['identity_sha256'])
        changed = copy.deepcopy(self.config)
        changed['experiments'][0]['model'] = 'different-model'
        self.assertNotEqual(pipeline.prepare(changed)['identity_sha256'], pipeline.prepare(self.config)['identity_sha256'])

    def test_workloads_select_and_report_separately(self):
        self.config.pop('suites')
        self.settings['supported_backends'] = ['live-feishu-v1']
        self.adapter.write_text(json.dumps(self.settings))
        self.output.mkdir()
        for category, tasks in (('diagnostic', 8), ('workflow', 6)):
            self.config['workloads'] = [category]
            plan = pipeline.prepare(self.config)
            self.assertEqual({j['workload'] for j in plan['jobs']}, {category})
            self.assertEqual(plan['planned_attempts'], tasks * 4)
            result = pipeline.report(plan, self.output)
            self.assertEqual(result['workloads'], {category: {'planned_attempts': tasks * 4, 'completed_attempts': 0}})
            report = (self.output / 'REPORT.md').read_text()
            self.assertIn('Realistic workflows' if category == 'workflow' else 'Diagnostics', report)
        for categories in ([], ['unknown'], ['workflow', 'workflow']):
            with self.assertRaises(ValueError):
                pipeline.prepare({**self.config, 'workloads': categories})
        with self.assertRaises(ValueError):
            pipeline.prepare({**self.config, 'workloads': ['workflow'], 'suites': ['foundation']})

    def test_unsupported_backend_and_invalid_selection_fail_before_calls(self):
        for update in ({'live_manifests': {}}, {'sections': ['unknown']}, {'repeats': 1},
                       {'suites': ['foundation', 'foundation']}):
            with self.subTest(update=update), self.assertRaises(ValueError):
                pipeline.prepare({**self.config, **update})
        self.assertFalse(self.runner.with_suffix('.calls').exists())

    def launch(self, *flags, ci=False):
        path = self.root / 'config.json'
        path.write_text(json.dumps(self.config))
        return subprocess.run([sys.executable, str(ROOT / 'evaluation/pipeline.py'),
                               '--config', str(path), '--output', str(self.output), *flags],
                              capture_output=True, text=True, timeout=60, env={**os.environ, 'CI': 'true' if ci else ''})

    def test_list_sections_needs_no_config_or_output(self):
        listed = subprocess.run([sys.executable, str(ROOT / 'evaluation/pipeline.py'),
                                 '--list-sections', '--suite', 'contact'],
                                capture_output=True, text=True, timeout=60)
        self.assertEqual(listed.returncode, 0, listed.stderr)
        self.assertIn('contact: contact (diagnostic, 1 task)', listed.stdout)
        self.assertNotIn('wiki:', listed.stdout)
        self.assertFalse(self.output.exists())

    def test_cli_filters_preview_and_forward_symmetric_budgets_without_preflight(self):
        self.settings['supported_backends'] = ['live-feishu-v1']
        self.adapter.write_text(json.dumps(self.settings))
        self.config['experiments'].append({**self.config['experiments'][0], 'id': 'test-b'})
        preview = self.launch('--dry-run', '--suite', 'foundation', 'contact',
                              '--section', 'foundation', '--section', 'contact',
                              '--workload', 'diagnostic', '--experiment', 'test-b',
                              '--repeats', '4', '--budget-seconds', '23')
        self.assertEqual(preview.returncode, 0, preview.stderr)
        plan = json.loads(preview.stdout)
        self.assertEqual({j['experiment'] for j in plan['jobs']}, {'test-b'})
        self.assertEqual({j['suite'] for j in plan['jobs']}, {'foundation', 'contact'})
        self.assertEqual(plan['planned_attempts'], 3 * 4 * 2)
        self.assertEqual((plan['repeats'], plan['budget_seconds']), (4, 23))
        self.assertFalse(self.output.exists())
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        config = {**self.config, 'repeats': 4, 'budget_seconds': 23}
        for job in plan['jobs']:
            command = pipeline.command(config, job, self.output)
            self.assertIn('live_batch.py', command[1])
            self.assertIn('--execute', command)
            self.assertNotIn('run_read_batch.py', ' '.join(command))
            self.assertIn('--progress', command)
            self.assertNotIn('--progress', pipeline.command(config, job, self.output, True))
            settings = batch_config(config, job['experiment'], self.output)
            self.assertEqual(settings['budget']['timeout_seconds'], 23)
            self.assertEqual(settings['timeout_seconds'], 53)

    def test_cli_section_selection_filters_tasks_within_suites(self):
        self.settings['supported_backends'] = ['live-feishu-v1']
        self.adapter.write_text(json.dumps(self.settings))
        preview = self.launch('--dry-run', '--suite', 'paired', 'scenario', 'docs-hard',
                              '--section', 'docs')
        self.assertEqual(preview.returncode, 0, preview.stderr)
        plan = json.loads(preview.stdout)
        self.assertEqual({j['suite'] for j in plan['jobs']}, {'paired', 'scenario', 'docs-hard'})
        self.assertTrue(all(j['sections'] == ['docs'] for j in plan['jobs']))
        self.assertEqual(plan['planned_attempts'], 6 * 2 * 2)

    def test_dry_run_needs_no_output_and_failures_explain_recovery(self):
        config = self.root / 'preview.json'
        config.write_text(json.dumps(self.config))
        preview = subprocess.run([sys.executable, str(ROOT / 'evaluation/pipeline.py'),
                                  '--config', str(config), '--dry-run'],
                                 capture_output=True, text=True, timeout=60)
        self.assertEqual(preview.returncode, 0, preview.stderr)
        self.assertGreater(json.loads(preview.stdout)['planned_attempts'], 0)
        for flags, hint in ((('--section', 'unknown'), '--list-sections'),
                            (('--experiment', 'unknown'), 'experiments array'),
                            (('--repeats', '1'), 'two repeats'),
                            (('--budget-seconds', '0'), '--budget-seconds'),
                            (('--workload', 'workflow'), 'conflicts with workloads')):
            with self.subTest(flags=flags):
                result = self.launch('--dry-run', *flags)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertIn(hint, json.loads(result.stderr)['message'])
        self.assertFalse(self.output.exists())
        self.assertFalse(self.runner.with_suffix('.calls').exists())

    def test_custom_harness_prepare_execute_resume_and_report_end_to_end(self):
        prepared = self.launch()
        self.assertEqual(prepared.returncode, 0, prepared.stderr)
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        self.assertFalse(json.loads((self.output / 'summary.json').read_text())['complete'])
        executed = self.launch('--execute')
        # Deliberate outcome failures are retained, all tasks are still attempted.
        self.assertEqual(executed.returncode, 1, executed.stderr)
        summary = json.loads((self.output / 'summary.json').read_text())
        self.assertTrue(summary['complete'])
        self.assertTrue(all(row['observed_token_difference'] is None for row in summary['comparisons']))
        calls = self.runner.with_suffix('.calls').read_text()
        self.assertEqual(len(calls.splitlines()), summary['coverage'][0]['planned_attempts'])
        self.assertEqual(executed.stderr.count(' checkpointed:'), len(calls.splitlines()))
        self.assertIn('tokens=120; accounting=complete', executed.stderr)
        self.assertIn('reports updated', executed.stderr)
        resumed = self.launch('--execute')
        self.assertEqual(resumed.returncode, 1)
        self.assertIn('0 remaining', resumed.stderr)
        self.assertNotIn(' checkpointed:', resumed.stderr)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        changed = self.launch('--execute', '--repeats', '3')
        self.assertEqual(changed.returncode, 2)
        self.assertIn('new --output directory', json.loads(changed.stderr)['message'])
        rendered = subprocess.run([sys.executable, str(ROOT / 'evaluation/pipeline.py'),
                                   '--output', str(self.output), '--report-only'],
                                  capture_output=True, text=True, timeout=60)
        self.assertEqual(rendered.returncode, 0, rendered.stderr)
        self.assertEqual(json.loads(rendered.stdout)['status'], 'report_rendered')
        self.assertEqual(self.launch('--report-only', '--section', 'foundation').returncode, 2)
        self.assertEqual(self.launch('--report-only').returncode, 0)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        self.config['experiments'][0]['model'] = 'changed'
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        self.assertTrue((self.output / 'summary.csv').is_file())

    def test_synthetic_manifests_and_controlled_runners_are_rejected(self):
        for backend in ('synthetic-scenario-v1', 'synthetic-read-v1', 'offline-foundation-v1'):
            path = Path(self.config['live_manifests']['foundation'])
            manifest = json.loads(path.read_text())
            manifest['backend'] = backend
            path.write_text(json.dumps(manifest))
            self.assertEqual(self.launch('--execute').returncode, 2)
        manifest['backend'] = 'live-feishu-v1'
        path.write_text(json.dumps(manifest))
        self.settings['mode'] = 'controlled'
        self.adapter.write_text(json.dumps(self.settings))
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        self.assertFalse(self.driver.with_suffix('.calls').exists())

    def test_live_lifecycle_is_not_run_during_preparation(self):
        self.assertEqual(self.launch().returncode, 0)
        self.assertFalse(self.driver.with_suffix('.calls').exists())
        self.assertEqual(self.launch('--execute').returncode, 1)
        calls = self.driver.with_suffix('.calls').read_text().splitlines()
        self.assertEqual(calls, ['setup', 'verify', 'cleanup'] * 8)
        self.assertEqual(self.launch('--execute').returncode, 1)
        self.assertEqual(self.driver.with_suffix('.calls').read_text().splitlines(), calls)

    def test_unverified_privacy_blocks_before_model_and_is_never_replayed(self):
        self.driver.with_suffix('.fail-privacy').touch()
        first = self.launch('--execute')
        self.assertEqual(first.returncode, 2, first.stderr)
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        calls = self.driver.with_suffix('.calls').read_text()
        self.assertEqual(calls.splitlines(), ['setup'])
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertEqual(self.driver.with_suffix('.calls').read_text(), calls)

    def test_manifest_and_driver_bytes_are_frozen(self):
        initial = pipeline.prepare(self.config)['identity_sha256']
        path = Path(self.config['live_manifests']['foundation'])
        data = json.loads(path.read_text())
        data['tasks'][0]['prompt'] += ' changed live goal'
        path.write_text(json.dumps(data))
        changed = pipeline.prepare(self.config)['identity_sha256']
        self.assertNotEqual(initial, changed)
        self.driver.write_text(self.driver.read_text() + '\n# changed live controller\n')
        self.assertNotEqual(changed, pipeline.prepare(self.config)['identity_sha256'])

    def test_live_execution_is_blocked_in_ci(self):
        result = self.launch('--execute', ci=True)
        self.assertEqual(result.returncode, 2, result.stderr)
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        self.assertFalse(self.driver.with_suffix('.calls').exists())

    def test_reconciliation_blocks_even_a_completed_batch(self):
        self.assertEqual(self.launch('--execute').returncode, 1)
        calls = self.runner.with_suffix('.calls').read_text()
        marker = self.output / 'test-a/foundation/.private/RECONCILE'
        marker.write_text('{"needs_review":true}')
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)

    def test_cleanup_failure_checkpoints_cost_and_blocks_resume(self):
        self.driver.with_suffix('.fail-cleanup').touch()
        failed = self.launch('--execute')
        self.assertEqual(failed.returncode, 2, failed.stderr)
        batch = json.loads((self.output / 'test-a/foundation/foundation/batch-01.json').read_text())
        self.assertEqual(len(batch['runs']), 1)
        self.assertEqual(batch['runs'][0]['usage']['total_tokens'], 120)
        self.assertIn('live_cleanup_unconfirmed', batch['runs'][0]['categories'])
        calls = self.runner.with_suffix('.calls').read_text()
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)

    def test_operator_stop_prevents_model_launches(self):
        self.assertEqual(self.launch().returncode, 0)
        (self.output / 'STOP').touch()
        stopped = self.launch('--execute')
        self.assertEqual(stopped.returncode, 1, stopped.stderr)
        self.assertFalse(self.runner.with_suffix('.calls').exists())
        self.assertFalse(json.loads((self.output / 'summary.json').read_text())['complete'])

    def test_report_counts_failed_cost_and_suppresses_unknown_usage(self):
        plan = pipeline.prepare(self.config)
        self.output.mkdir()
        job = plan['jobs'][0]
        job['task_ids'] = job['task_ids'][:1]
        job['section_tasks'] = {'foundation': job['task_ids']}
        job['planned_attempts'] = 4
        runs = []
        for arm in ('official', 'next'):
            for trial in range(2):
                tokens = 200 if arm == 'official' else 100
                runs.append({'task_id': job['task_ids'][0], 'section': 'foundation', 'arm': arm,
                             'trial': trial, 'success': trial == 0, 'outcome_inconclusive': False,
                             'usage': {'total_tokens': tokens}, 'categories': [],
                             'latency_seconds': 1, 'api_calls': 0, 'retries': 0})
        path = self.output / 'test-a/foundation/foundation/batch-01.json'
        path.parent.mkdir(parents=True)
        def save():
            path.write_text(json.dumps({'runs': runs, 'completed': True, 'mode': 'real',
                                        'environment': {'backend': 'live-feishu-v1'},
                                        **aggregate(runs, ('official', 'next'))}))
        save()
        result = pipeline.report(plan, self.output)
        row = next(r for r in result['comparisons'] if r['scope'] == 'task_id')
        self.assertEqual(row['official']['tokens_per_success'], 400)
        self.assertEqual(row['next']['tokens_per_success'], 200)
        self.assertEqual(row['observed_token_difference'], .5)
        report_text = (self.output / 'REPORT.md').read_text()
        self.assertIn('| Latency O/N | API calls O/N | Retries O/N |', report_text)
        summary_csv = list(csv.reader(io.StringIO((self.output / 'summary.csv').read_text())))
        self.assertIn('official_latency_median_seconds', summary_csv[0])
        self.assertIn('next_retries', summary_csv[0])
        path.write_text(json.dumps({'runs': runs[:1], 'completed': True, 'mode': 'real',
                                    'environment': {'backend': 'live-feishu-v1'},
                                    **aggregate(runs[:1], ('official', 'next'))}))
        self.assertTrue(all(r['observed_token_difference'] is None for r in pipeline.report(plan, self.output)['comparisons']))
        save()
        runs[-1]['known_usage'] = runs[-1]['usage']
        runs[-1]['usage'] = None
        save()
        result = pipeline.report(plan, self.output)
        self.assertTrue(all(r['observed_token_difference'] is None for r in result['comparisons']))
        self.assertEqual(result['comparisons'][0]['next']['known_total_tokens'], 200)
        data = json.loads(path.read_text())
        data['environment']['backend'] = 'synthetic-scenario-v1'
        path.write_text(json.dumps(data))
        with self.assertRaises(pipeline.PipelineError):
            pipeline.report(plan, self.output)
        runs[-1]['usage'] = runs[-1]['known_usage']
        runs[-1]['outcome_inconclusive'] = True
        save()
        self.assertTrue(all(r['observed_token_difference'] is None for r in pipeline.report(plan, self.output)['comparisons']))


if __name__ == '__main__':
    unittest.main()
