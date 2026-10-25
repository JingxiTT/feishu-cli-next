# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""Offline matrix planning, accounting and complete custom-runner lifecycle."""
import copy
import csv
import io
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
import pipeline
from harness import aggregate


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
print(json.dumps({'claimed_success': False, 'usage_source': 'synthetic',
 'model': r['settings']['model'], 'tokenizer': r['settings']['tokenizer'],
 'accounting_complete': True, 'turns': [
 {'input_tokens': 100, 'cached_input_tokens': 40, 'output_tokens': 20}]}))
''')
        self.adapter = self.root / 'adapter.json'
        self.settings = {'runner': [sys.executable, str(self.runner)], 'mode': 'controlled',
                         'accounting_method': 'synthetic test events, not model measurements',
                         'tokenizer': 'synthetic', 'tools': ['shell', 'files'],
                         'permissions': 'isolated synthetic test only',
                         'supported_backends': ['offline-foundation-v1']}
        self.adapter.write_text(json.dumps(self.settings))
        self.config = {'schema_version': 1, 'official_source': str(self.official),
                       'next_source': str(self.next), 'official_cli': str(self.cli), 'next_cli': str(self.cli),
                       'repeats': 2, 'suites': ['foundation'], 'experiments': [
                           {'id': 'test-a', 'harness': 'test-adapter-v1', 'model': 'test-model',
                            'reasoning': 'none', 'runner_config': str(self.adapter)}]}
        self.output = self.root / 'output'

    def test_plan_covers_all_implemented_suites_without_reading_held_out(self):
        self.config.pop('suites')
        self.settings['supported_backends'] = ['offline-foundation-v1', 'synthetic-contact-v1',
                                               'synthetic-read-v1', 'synthetic-scenario-v1']
        self.adapter.write_text(json.dumps(self.settings))
        plan = pipeline.prepare(self.config)
        self.assertEqual({j['suite'] for j in plan['jobs']}, set(pipeline.SUITES))
        self.assertEqual(sum(len(j['task_ids']) for j in plan['jobs'] if j['workload'] == 'diagnostic'), 24)
        self.assertEqual(sum(len(j['task_ids']) for j in plan['jobs'] if j['workload'] == 'workflow'), 21)
        self.assertEqual(plan['workloads'], ['workflow', 'diagnostic'])
        self.assertEqual(plan['planned_attempts'], 45 * 2 * 2)
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
        self.settings['supported_backends'] = ['offline-foundation-v1', 'synthetic-contact-v1',
                                               'synthetic-read-v1', 'synthetic-scenario-v1']
        self.adapter.write_text(json.dumps(self.settings))
        self.output.mkdir()
        for category, tasks in (('diagnostic', 24), ('workflow', 21)):
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
        for update in ({'suites': ['read']}, {'sections': ['unknown']}, {'repeats': 1},
                       {'suites': ['foundation', 'foundation']}):
            with self.subTest(update=update), self.assertRaises(ValueError):
                pipeline.prepare({**self.config, **update})
        self.assertFalse(self.runner.with_suffix('.calls').exists())

    def launch(self, *flags):
        path = self.root / 'config.json'
        path.write_text(json.dumps(self.config))
        return subprocess.run([sys.executable, str(ROOT / 'evaluation/pipeline.py'),
                               '--config', str(path), '--output', str(self.output), *flags],
                              capture_output=True, text=True, timeout=60)

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
        self.assertEqual(self.launch('--execute').returncode, 1)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        self.assertEqual(self.launch('--report-only').returncode, 0)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        self.config['experiments'][0]['model'] = 'changed'
        self.assertEqual(self.launch('--execute').returncode, 2)
        self.assertEqual(self.runner.with_suffix('.calls').read_text(), calls)
        self.assertTrue((self.output / 'summary.csv').is_file())

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
                                    **aggregate(runs[:1], ('official', 'next'))}))
        self.assertTrue(all(r['observed_token_difference'] is None for r in pipeline.report(plan, self.output)['comparisons']))
        save()
        runs[-1]['known_usage'] = runs[-1]['usage']
        runs[-1]['usage'] = None
        save()
        result = pipeline.report(plan, self.output)
        self.assertTrue(all(r['observed_token_difference'] is None for r in result['comparisons']))
        self.assertEqual(result['comparisons'][0]['next']['known_total_tokens'], 200)
        runs[-1]['usage'] = runs[-1]['known_usage']
        runs[-1]['outcome_inconclusive'] = True
        save()
        self.assertTrue(all(r['observed_token_difference'] is None for r in pipeline.report(plan, self.output)['comparisons']))


if __name__ == '__main__':
    unittest.main()
