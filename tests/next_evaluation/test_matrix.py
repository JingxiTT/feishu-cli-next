# Copyright (c) 2026 Lark Technologies Pte. Ltd.
# SPDX-License-Identifier: MIT

"""No-provider matrix cache, historical reuse and accounting regression checks."""
import copy
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / 'evaluation'))
import run_matrix as matrix


class MatrixTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.output = Path(self.temp.name)
        self.batch = matrix.read(ROOT / 'evidence/evaluation/token-efficiency-09/docs/batch-01.json')
        self.env = {'model': 'gpt-5.6-luna', 'reasoning': 'medium', 'guidance_delivery': 'file', 'agent_seconds': 180}
        self.job = next(j for j in matrix.jobs(self.env) if j['suite'] == 'docs-hard')
        self.job['expected_arms'] = self.batch['arms']

    def test_complete_inventory_unique_43_tasks_258_slots(self):
        jobs = matrix.jobs(self.env)
        ids = [task for j in jobs for task in j['tasks']]
        self.assertEqual(len(ids), 43)
        self.assertEqual(len(set(ids)), 43)
        self.assertEqual(sum(j['planned_attempts'] for j in jobs), 258)

    def test_changed_runtime_settings_invalidate_keys(self):
        for field in ('model', 'guidance_delivery', 'runtime_sources', 'launcher_sha256'):
            env = {**self.env, field: 'changed'}
            self.assertNotEqual(matrix.jobs(env)[0]['key'], matrix.jobs(self.env)[0]['key'])

    def test_batch_rejects_wrong_binary_skill_schedule_and_usage(self):
        matrix.validate_batch(self.batch, self.job, self.env)
        variants = []
        b = copy.deepcopy(self.batch); b['arms']['next']['skill_sha256'] = 'wrong'; variants.append(b)
        b = copy.deepcopy(self.batch); b['arms']['official']['cli_sha256'] = 'wrong'; variants.append(b)
        b = copy.deepcopy(self.batch); b['runs'][0]['trial'] = 10; variants.append(b)
        b = copy.deepcopy(self.batch); b['runs'][0]['usage']['total_tokens'] += 1; variants.append(b)
        for batch in variants:
            with self.assertRaises(ValueError):
                matrix.validate_batch(batch, self.job, self.env)

    def test_exact_prefix_allowed_only_as_incomplete(self):
        b = copy.deepcopy(self.batch); b['runs'] = b['runs'][:4]; b['completed'] = False
        matrix.validate_batch(b, self.job, self.env, complete=False)
        with self.assertRaises(ValueError):
            matrix.validate_batch(b, self.job, self.env)

    def test_reused_inconclusive_keeps_cost_and_unavailable_edit(self):
        folder = self.output / 'jobs' / self.job['key']
        batch = folder / 'docs/batch-01.json'
        matrix.write(batch, self.batch)
        matrix.write(folder / 'reuse.json', {'batch_sha256': matrix.sha(batch)})
        summary = matrix.report(self.output, [self.job], self.env)
        self.assertTrue(summary['completed'])
        self.assertEqual(summary['reused_known_tokens'], 901686)
        self.assertEqual(summary['fresh_known_tokens'], 0)
        self.assertIsNone(summary['pooled_saving_fraction'])
        edit = next(t for t in summary['tasks'] if t['task'] == 'docs-hard-multi-edit')
        self.assertIsNone(edit['saving_fraction'])
        self.assertEqual(edit['gate'], 'unavailable')
        self.assertEqual(edit['arms']['official']['known_total_tokens'], 366655)

    def test_strict_docs_boundary_and_success_regression(self):
        def row(tokens, successes=3, inconclusive=0):
            return {'known_total_tokens': tokens, 'tokens_per_success': tokens/successes if successes else None,
                    'successes': successes, 'inconclusive': inconclusive}
        rows = {'official': row(300), 'next': row(150)}
        self.assertEqual(matrix.score(rows, True)['gate'], 'missed')
        rows['next'] = row(149)
        self.assertEqual(matrix.score(rows, True)['gate'], 'met')
        rows['next'] = row(50, 2)
        self.assertEqual(matrix.score(rows, False)['gate'], 'missed')
        rows['next'] = row(210)
        self.assertEqual(matrix.score(rows, False)['gate'], 'met')

    def test_completed_cache_never_launches_even_inconclusive(self):
        matrix.write(self.output / 'jobs' / self.job['key'] / 'docs/batch-01.json', self.batch)
        config = self.output / 'config.json'; matrix.write(config, {})
        args = ['run_matrix.py', '--config', str(config), '--output', str(self.output), '--execute']
        with patch.object(sys, 'argv', args), patch.object(matrix, 'environment', return_value=(self.env, {})), \
                patch.object(matrix, 'jobs', return_value=[self.job]), patch.object(matrix, 'bind_jobs'), \
                patch.object(matrix, 'import_sibling_jobs') as import_jobs, patch.object(matrix.subprocess, 'run') as run:
            self.assertEqual(matrix.main(), 0)
            import_jobs.assert_called_once()
            run.assert_not_called()

    def test_changed_existing_environment_refuses_before_launch(self):
        matrix.write(self.output / 'environment.json', {'changed': True})
        config = self.output / 'config.json'; matrix.write(config, {})
        with patch.object(sys, 'argv', ['run_matrix.py', '--config', str(config), '--output', str(self.output), '--execute']), \
                patch.object(matrix, 'environment', return_value=(self.env, {})), patch.object(matrix.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'Inputs changed'):
                matrix.main()
            run.assert_not_called()

    def test_import_never_silently_replaces_failed_rows(self):
        bad = copy.deepcopy(self.batch); bad['runs'][0]['success'] = not bad['runs'][0]['success']
        folder = self.output / 'jobs' / self.job['key']
        path = folder / 'docs/batch-01.json'; matrix.write(path, bad)
        matrix.write(folder / 'reuse.json', {'batch_sha256': matrix.digest(self.batch)})
        with self.assertRaisesRegex(ValueError, 'Reused results changed'):
            matrix.report(self.output, [self.job], self.env)


if __name__ == '__main__':
    unittest.main()
